package database_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/D4nRusu/band4band/backend/internal/database"
	"github.com/D4nRusu/band4band/backend/internal/models"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func debtRef(db *firestore.Client, groupID string) *firestore.DocumentRef {
	return db.Collection("groups").Doc(groupID).Collection("debts").Doc("current")
}

func readStoredGraph(t *testing.T, db *firestore.Client, ctx context.Context, groupID string) models.DebtGraph {
	t.Helper()
	snapshot, err := debtRef(db, groupID).Get(ctx)
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

func checkDebt(t *testing.T, graph models.DebtGraph, from, to string, want int64) {
	t.Helper()
	var amount int64
	for _, edge := range graph.Edges {
		if edge.FromUID == from && edge.ToUID == to {
			amount = edge.AmountMinor
		}
	}

	if amount != want {
		t.Fatalf("%s -> %s = %d, want %d; graph=%+v", from, to, amount, want, graph)
	}
}

func graphGroup(t *testing.T, groups *database.GroupStore, db *firestore.Client, ctx context.Context) models.Group {
	t.Helper()
	group, err := groups.Create(ctx, "alice", models.GroupInput{Name: "Trip"})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := db.Collection("groups").Doc(group.ID).Update(ctx, []firestore.Update{
		{Path: "memberUIDs", Value: firestore.ArrayUnion("bob", "charlie")},
	}); err != nil {
		t.Fatal(err)
	}

	return group
}

func graphExpense(t *testing.T, groups *database.GroupStore, ctx context.Context, groupID, payer string, amount int64, participants ...string) models.Expense {
	t.Helper()
	expense, err := groups.CreateExpense(ctx, "alice", groupID, models.ExpenseInput{
		Name: "Test expense", PaidBy: payer, CategoryID: "default_food", AmountMinor: amount, ParticipantUIDs: participants,
	})
	if err != nil {
		t.Fatal(err)
	}

	return expense
}

func TestGraphUpdatesAndReversesWithoutNetting(t *testing.T) {
	_, db, ctx := emulatorStore(t)
	groups := database.NewGroupStore(db)
	group := graphGroup(t, groups, db, ctx)
	first := graphExpense(t, groups, ctx, group.ID, "alice", 3000, "alice", "bob", "charlie")
	second := graphExpense(t, groups, ctx, group.ID, "bob", 400, "alice")
	third := graphExpense(t, groups, ctx, group.ID, "alice", 600, "bob")
	graph := readStoredGraph(t, db, ctx, group.ID)
	checkDebt(t, graph, "bob", "alice", 1600)
	checkDebt(t, graph, "charlie", "alice", 1000)
	checkDebt(t, graph, "alice", "bob", 400)

	for range 2 {
		if err := groups.DeleteExpense(ctx, "charlie", group.ID, first.ID); err != nil {
			t.Fatal(err)
		}
	}

	graph = readStoredGraph(t, db, ctx, group.ID)
	checkDebt(t, graph, "bob", "alice", 600)
	checkDebt(t, graph, "charlie", "alice", 0)
	checkDebt(t, graph, "alice", "bob", 400)

	for _, id := range []string{second.ID, third.ID} {
		if err := groups.DeleteExpense(ctx, "bob", group.ID, id); err != nil {
			t.Fatal(err)
		}
	}

	if graph := readStoredGraph(t, db, ctx, group.ID); len(graph.Edges) != 0 {
		t.Fatalf("edges=%+v", graph.Edges)
	}
}

func TestGraphInitializesOnlyWhenMissing(t *testing.T) {
	_, db, ctx := emulatorStore(t)
	groups := database.NewGroupStore(db)
	group := graphGroup(t, groups, db, ctx)
	graphExpense(t, groups, ctx, group.ID, "alice", 1000, "bob")
	deleted := graphExpense(t, groups, ctx, group.ID, "alice", 200, "charlie")
	if err := groups.DeleteExpense(ctx, "alice", group.ID, deleted.ID); err != nil {
		t.Fatal(err)
	}

	if _, err := db.Collection("groups").Doc(group.ID).Collection("expenses").Doc("legacy").Create(ctx, map[string]any{
		"name": "Legacy", "paidBy": "alice", "amountMinor": 900, "currency": "EUR", "createdAt": time.Now(),
	}); err != nil {
		t.Fatal(err)
	}

	if _, err := debtRef(db, group.ID).Delete(ctx); err != nil {
		t.Fatal(err)
	}

	if _, err := groups.Expenses(ctx, "outsider", group.ID); !errors.Is(err, models.ErrGroupAccessForbidden) {
		t.Fatalf("outsider=%v", err)
	}

	_, err := debtRef(db, group.ID).Get(ctx)
	if status.Code(err) != codes.NotFound {
		t.Fatalf("unauthorized read initialized graph: %v", err)
	}

	if _, err := groups.Expenses(ctx, "alice", group.ID); err != nil {
		t.Fatal(err)
	}

	graph := readStoredGraph(t, db, ctx, group.ID)
	checkDebt(t, graph, "bob", "alice", 1000)
	checkDebt(t, graph, "charlie", "alice", 0)
	before, err := debtRef(db, group.ID).Get(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := groups.Expenses(ctx, "alice", group.ID); err != nil {
		t.Fatal(err)
	}

	after, err := debtRef(db, group.ID).Get(ctx)
	if err != nil || !before.UpdateTime.Equal(after.UpdateTime) {
		t.Fatal("reading rewrote existing graph")
	}

	// A valid preexisting edge distinguishes incremental updates from expense replay.
	if err := graph.AdjustEdge("bob", "charlie", 777); err != nil {
		t.Fatal(err)
	}

	if _, err := debtRef(db, group.ID).Update(ctx, []firestore.Update{{Path: "edges", Value: graph.Edges}}); err != nil {
		t.Fatal(err)
	}

	graphExpense(t, groups, ctx, group.ID, "alice", 500, "bob")
	graph = readStoredGraph(t, db, ctx, group.ID)
	checkDebt(t, graph, "bob", "alice", 1500)
	checkDebt(t, graph, "bob", "charlie", 777)
}

func TestMissingGraphCreationAndDeletionAvoidDoubleCounting(t *testing.T) {
	_, db, ctx := emulatorStore(t)
	groups := database.NewGroupStore(db)
	group := graphGroup(t, groups, db, ctx)
	first := graphExpense(t, groups, ctx, group.ID, "alice", 1000, "bob")
	if _, err := debtRef(db, group.ID).Delete(ctx); err != nil {
		t.Fatal(err)
	}

	graphExpense(t, groups, ctx, group.ID, "alice", 500, "bob")
	checkDebt(t, readStoredGraph(t, db, ctx, group.ID), "bob", "alice", 1500)

	if _, err := debtRef(db, group.ID).Delete(ctx); err != nil {
		t.Fatal(err)
	}

	if err := groups.DeleteExpense(ctx, "bob", group.ID, first.ID); err != nil {
		t.Fatal(err)
	}

	checkDebt(t, readStoredGraph(t, db, ctx, group.ID), "bob", "alice", 500)
}

func TestConcurrentExpenseGraphUpdates(t *testing.T) {
	_, db, ctx := emulatorStore(t)
	groups := database.NewGroupStore(db)
	group := graphGroup(t, groups, db, ctx)
	results := make(chan error, 4)
	var wg sync.WaitGroup
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := groups.CreateExpense(ctx, "alice", group.ID, models.ExpenseInput{
				Name: "Concurrent", PaidBy: "alice", CategoryID: "default_food", AmountMinor: 100, ParticipantUIDs: []string{"bob"},
			})
			results <- err
		}()
	}
	wg.Wait()
	close(results)
	for err := range results {
		if err != nil {
			t.Fatal(err)
		}
	}

	checkDebt(t, readStoredGraph(t, db, ctx, group.ID), "bob", "alice", 400)
}

func TestInvalidGraphAbortsExpenseChanges(t *testing.T) {
	_, db, ctx := emulatorStore(t)
	groups := database.NewGroupStore(db)
	group := graphGroup(t, groups, db, ctx)
	expense := graphExpense(t, groups, ctx, group.ID, "alice", 1000, "bob")
	if _, err := debtRef(db, group.ID).Update(ctx, []firestore.Update{{Path: "edges", Value: []models.DebtEdge{}}}); err != nil {
		t.Fatal(err)
	}

	if err := groups.DeleteExpense(ctx, "bob", group.ID, expense.ID); !errors.Is(err, models.ErrDebtUnderflow) {
		t.Fatalf("err=%v", err)
	}

	snapshot, err := db.Collection("groups").Doc(group.ID).Collection("expenses").Doc(expense.ID).Get(ctx)
	if err != nil {
		t.Fatal(err)
	}

	var stored models.Expense
	if err := snapshot.DataTo(&stored); err != nil {
		t.Fatal(err)
	}

	if !stored.DeletedAt.IsZero() {
		t.Fatal("failed reversal deleted expense")
	}

	if _, err := debtRef(db, group.ID).Update(ctx, []firestore.Update{{Path: "currency", Value: "USD"}}); err != nil {
		t.Fatal(err)
	}

	_, err = groups.CreateExpense(ctx, "alice", group.ID, models.ExpenseInput{
		Name: "Should fail", PaidBy: "alice", CategoryID: "default_food", AmountMinor: 100, ParticipantUIDs: []string{"bob"},
	})
	if !errors.Is(err, models.ErrInvalidDebtGraph) {
		t.Fatalf("err=%v", err)
	}

	all, err := db.Collection("groups").Doc(group.ID).Collection("expenses").Documents(ctx).GetAll()
	if err != nil || len(all) != 1 {
		t.Fatalf("failed transaction saved expense: count=%d err=%v", len(all), err)
	}
}
