package database_test

import (
	"context"
	"testing"

	"cloud.google.com/go/firestore"
	"github.com/D4nRusu/band4band/backend/internal/database"
	"github.com/D4nRusu/band4band/backend/internal/models"
)

func readSimplifiedGraph(t *testing.T, db *firestore.Client, ctx context.Context, groupID string) models.DebtGraph {
	t.Helper()
	snapshot, err := debtRef(db, groupID).Parent.Doc("simplified").Get(ctx)
	if err != nil {
		t.Fatal(err)
	}

	var graph models.DebtGraph
	if err := snapshot.DataTo(&graph); err != nil {
		t.Fatal(err)
	}

	if err := graph.Validate(); err != nil {
		t.Fatal(err)
	}

	return graph
}

func TestExpenseChangesPersistSimplifiedDebts(t *testing.T) {
	_, db, ctx := emulatorStore(t)
	groups := database.NewGroupStore(db)
	group := graphGroup(t, groups, db, ctx)
	first := graphExpense(t, groups, ctx, group.ID, "bob", 1000, "alice")
	checkDebt(t, readSimplifiedGraph(t, db, ctx, group.ID), "alice", "bob", 1000)
	graphExpense(t, groups, ctx, group.ID, "charlie", 700, "bob")
	closing := graphExpense(t, groups, ctx, group.ID, "alice", 400, "charlie")

	raw := readStoredGraph(t, db, ctx, group.ID)
	checkDebt(t, raw, "alice", "bob", 1000)
	checkDebt(t, raw, "bob", "charlie", 700)
	checkDebt(t, raw, "charlie", "alice", 400)
	simplified := readSimplifiedGraph(t, db, ctx, group.ID)
	checkDebt(t, simplified, "alice", "bob", 300)
	checkDebt(t, simplified, "alice", "charlie", 300)
	checkDebt(t, simplified, "bob", "charlie", 0)
	checkDebt(t, simplified, "charlie", "alice", 0)
	if !raw.UpdatedAt.Equal(simplified.UpdatedAt) {
		t.Fatal("raw and simplified graphs were not updated together")
	}

	// Deletion reverses the raw allocation, including an edge cancelled in the derived graph.
	if err := groups.DeleteExpense(ctx, "alice", group.ID, closing.ID); err != nil {
		t.Fatal(err)
	}

	simplified = readSimplifiedGraph(t, db, ctx, group.ID)
	checkDebt(t, simplified, "alice", "bob", 300)
	checkDebt(t, simplified, "alice", "charlie", 700)
	checkDebt(t, simplified, "bob", "charlie", 0)
	checkDebt(t, simplified, "charlie", "alice", 0)
	beforeRetry := simplified.UpdatedAt
	if err := groups.DeleteExpense(ctx, "alice", group.ID, closing.ID); err != nil {
		t.Fatal(err)
	}

	if !readSimplifiedGraph(t, db, ctx, group.ID).UpdatedAt.Equal(beforeRetry) {
		t.Fatal("repeated deletion rewrote simplified debts")
	}

	if err := groups.DeleteExpense(ctx, "bob", group.ID, first.ID); err != nil {
		t.Fatal(err)
	}

	checkDebt(t, readSimplifiedGraph(t, db, ctx, group.ID), "alice", "bob", 0)
}

func TestGraphInitializationAlsoWritesSimplifiedDebts(t *testing.T) {
	_, db, ctx := emulatorStore(t)
	groups := database.NewGroupStore(db)
	group := graphGroup(t, groups, db, ctx)
	graphExpense(t, groups, ctx, group.ID, "alice", 1000, "bob")
	graphExpense(t, groups, ctx, group.ID, "bob", 400, "alice")
	for _, id := range []string{"current", "simplified"} {
		if _, err := debtRef(db, group.ID).Parent.Doc(id).Delete(ctx); err != nil {
			t.Fatal(err)
		}
	}

	if _, err := groups.Expenses(ctx, "alice", group.ID); err != nil {
		t.Fatal(err)
	}

	checkDebt(t, readStoredGraph(t, db, ctx, group.ID), "bob", "alice", 1000)
	simplified := readSimplifiedGraph(t, db, ctx, group.ID)
	checkDebt(t, simplified, "bob", "alice", 600)
	checkDebt(t, simplified, "alice", "bob", 0)
}
