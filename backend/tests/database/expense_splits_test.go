package database_test

import (
	"errors"
	"testing"

	"cloud.google.com/go/firestore"
	"github.com/D4nRusu/band4band/backend/internal/database"
	"github.com/D4nRusu/band4band/backend/internal/models"
)

func TestExpenseSharesValidateMembershipAndPreserveHistory(t *testing.T) {
	users, db, ctx := emulatorStore(t)
	groups := database.NewGroupStore(db)
	if _, err := users.GetOrCreate(ctx, models.Identity{UID: "usr2", Name: "Alice"}); err != nil {
		t.Fatal(err)
	}

	group, err := groups.Create(ctx, "usr1", models.GroupInput{Name: "Trip"})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := db.Collection("groups").Doc(group.ID).Update(ctx, []firestore.Update{
		{Path: "memberUIDs", Value: firestore.ArrayUnion("usr2", "usr3")},
	}); err != nil {
		t.Fatal(err)
	}

	input := models.ExpenseInput{Name: "Dinner", CategoryID: "default_food", PaidBy: "usr1", AmountMinor: 1000, ParticipantUIDs: []string{"usr3", "usr2", "usr1"}}
	expense, err := groups.CreateExpense(ctx, "usr2", group.ID, input)
	if err != nil {
		t.Fatal(err)
	}

	if expense.SplitType != "equal" || len(expense.Contributions) != 3 || expense.Contributions[0].UID != "usr1" || expense.Contributions[0].AmountMinor != 334 || expense.Contributions[1].Name != "Alice" {
		t.Fatalf("expense=%+v", expense)
	}

	input.AmountMinor = 5000
	input.ParticipantUIDs = []string{"usr3", "usr2"}
	excludedPayer, err := groups.CreateExpense(ctx, "usr1", group.ID, input)
	if err != nil || len(excludedPayer.Contributions) != 2 || excludedPayer.Contributions[0].AmountMinor != 2500 {
		t.Fatalf("excluded payer expense=%+v err=%v", excludedPayer, err)
	}

	input.ParticipantUIDs = []string{"outsider"}
	if _, err := groups.CreateExpense(ctx, "usr1", group.ID, input); !errors.Is(err, models.ErrInvalidExpenseParticipants) {
		t.Fatalf("outsider contribution=%v", err)
	}

	if err := groups.Leave(ctx, "usr2", group.ID); err != nil {
		t.Fatal(err)
	}

	input.ParticipantUIDs = []string{"usr2"}
	if _, err := groups.CreateExpense(ctx, "usr1", group.ID, input); !errors.Is(err, models.ErrInvalidExpenseParticipants) {
		t.Fatalf("former member contribution=%v", err)
	}

	expenses, err := groups.Expenses(ctx, "usr1", group.ID)
	if err != nil {
		t.Fatal(err)
	}

	found := false
	for _, stored := range expenses {
		if stored.ID == expense.ID {
			found = true
			if len(stored.Contributions) != 3 || stored.Contributions[1].UID != "usr2" || stored.Contributions[1].Name != "Alice" || stored.Contributions[1].AmountMinor != 333 {
				t.Fatalf("shares changed after leaving: %+v", stored.Contributions)
			}
		}
	}

	if !found {
		t.Fatal("expense disappeared")
	}

	if err := groups.DeleteExpense(ctx, "usr3", group.ID, expense.ID); err != nil {
		t.Fatal(err)
	}

	snapshot, err := db.Collection("groups").Doc(group.ID).Collection("expenses").Doc(expense.ID).Get(ctx)
	if err != nil {
		t.Fatal(err)
	}

	var deleted models.Expense
	if err := snapshot.DataTo(&deleted); err != nil {
		t.Fatal(err)
	}

	if len(deleted.Contributions) != 3 || deleted.Contributions[0].AmountMinor != 334 || deleted.DeletedAt.IsZero() {
		t.Fatalf("deletion lost shares: %+v", deleted)
	}
}

func TestLegacyExpensesRemainUnallocated(t *testing.T) {
	_, db, ctx := emulatorStore(t)
	groups := database.NewGroupStore(db)
	group, err := groups.Create(ctx, "owner", models.GroupInput{Name: "Trip"})
	if err != nil {
		t.Fatal(err)
	}

	expense, err := groups.CreateExpense(ctx, "owner", group.ID, models.ExpenseInput{
		Name: "Old expense", CategoryID: "default_other", PaidBy: "owner", AmountMinor: 100, ParticipantUIDs: []string{"owner"},
	})
	if err != nil {
		t.Fatal(err)
	}

	ref := db.Collection("groups").Doc(group.ID).Collection("expenses").Doc(expense.ID)
	if _, err := ref.Update(ctx, []firestore.Update{
		{Path: "splitType", Value: firestore.Delete},
		{Path: "contributions", Value: firestore.Delete},
	}); err != nil {
		t.Fatal(err)
	}

	expenses, err := groups.Expenses(ctx, "owner", group.ID)
	if err != nil || len(expenses) != 1 || expenses[0].SplitType != "" || len(expenses[0].Contributions) != 0 {
		t.Fatalf("legacy shares inferred: %+v err=%v", expenses, err)
	}
}
