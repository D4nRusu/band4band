package database_test

import (
	"errors"
	"testing"

	"github.com/D4nRusu/band4band/backend/internal/database"
	"github.com/D4nRusu/band4band/backend/internal/models"
)

func TestUnequalExpenseSharesAndDebtReversal(t *testing.T) {
	users, db, ctx := emulatorStore(t)
	groups := database.NewGroupStore(db)
	group := graphGroup(t, groups, db, ctx)
	if _, err := users.GetOrCreate(ctx, models.Identity{UID: "bob", Name: "Bob"}); err != nil {
		t.Fatal(err)
	}

	input := models.ExpenseInput{Name: "Dinner", CategoryID: "default_food", PaidBy: "alice", AmountMinor: 1000,
		SplitType: "unequal", Contributions: []models.ExpenseContributionInput{
			{UID: "alice", AmountMinor: 100}, {UID: "bob", AmountMinor: 300}, {UID: "charlie", AmountMinor: 600},
		},
	}
	expense, err := groups.CreateExpense(ctx, "charlie", group.ID, input)
	if err != nil {
		t.Fatal(err)
	}

	if expense.SplitType != "unequal" || expense.Contributions[1].Name != "Bob" || expense.Contributions[1].AmountMinor != 300 {
		t.Fatalf("expense=%+v", expense)
	}

	if position := expense.PositionFor("alice"); position.Kind != "lent" || position.AmountMinor != 900 {
		t.Fatalf("payer position=%+v", position)
	}

	checkDebt(t, readStoredGraph(t, db, ctx, group.ID), "bob", "alice", 300)
	checkDebt(t, readSimplifiedGraph(t, db, ctx, group.ID), "charlie", "alice", 600)
	if err := groups.DeleteExpense(ctx, "bob", group.ID, expense.ID); err != nil {
		t.Fatal(err)
	}

	if len(readStoredGraph(t, db, ctx, group.ID).Adjacency) != 0 || len(readSimplifiedGraph(t, db, ctx, group.ID).Adjacency) != 0 {
		t.Fatal("unequal debts were not reversed")
	}

	input.Contributions = []models.ExpenseContributionInput{{UID: "outsider", AmountMinor: 1000}}
	if _, err := groups.CreateExpense(ctx, "alice", group.ID, input); !errors.Is(err, models.ErrInvalidExpenseParticipants) {
		t.Fatalf("outsider=%v", err)
	}

	if err := groups.Leave(ctx, "bob", group.ID); err != nil {
		t.Fatal(err)
	}

	input.Contributions = []models.ExpenseContributionInput{{UID: "bob", AmountMinor: 1000}}
	if _, err := groups.CreateExpense(ctx, "alice", group.ID, input); !errors.Is(err, models.ErrInvalidExpenseParticipants) {
		t.Fatalf("former member=%v", err)
	}
}
