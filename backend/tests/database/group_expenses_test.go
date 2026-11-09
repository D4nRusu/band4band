package database_test

import (
	"errors"
	"testing"

	"cloud.google.com/go/firestore"
	"github.com/D4nRusu/band4band/backend/internal/database"
	"github.com/D4nRusu/band4band/backend/internal/models"
)

func TestExpenseCreationAndHistory(t *testing.T) {
	users, db, ctx := emulatorStore(t)
	groups := database.NewGroupStore(db)
	if _, err := users.GetOrCreate(ctx, models.Identity{UID: "payer", Name: "Alice"}); err != nil {
		t.Fatal(err)
	}

	group, err := groups.Create(ctx, "owner", models.GroupInput{Name: "Trip"})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := db.Collection("groups").Doc(group.ID).Update(ctx, []firestore.Update{
		{Path: "memberUIDs", Value: firestore.ArrayUnion("payer")},
		{Path: "currency", Value: firestore.Delete},
	}); err != nil {
		t.Fatal(err)
	}

	category, err := groups.AddCategory(ctx, "owner", group.ID, models.CategoryInput{Name: "Pet care", Icon: "other"})
	if err != nil {
		t.Fatal(err)
	}

	input := models.ExpenseInput{Name: " Supplies ", CategoryID: category.ID, PaidBy: "payer", AmountMinor: 1250}
	expense, err := groups.CreateExpense(ctx, "owner", group.ID, input)
	if err != nil {
		t.Fatal(err)
	}

	if expense.Name != "Supplies" || expense.AmountMinor != 1250 || expense.Currency != "EUR" || expense.CreatedBy != "owner" || expense.PayerName != "Alice" || expense.CategoryName != "Pet care" || expense.CreatedAt.IsZero() {
		t.Fatalf("expense=%+v", expense)
	}

	if err := groups.Delete(ctx, "owner", group.ID); !errors.Is(err, models.ErrGroupHasExpenses) {
		t.Fatalf("expense group deletion=%v", err)
	}

	if _, err := groups.CreateExpense(ctx, "outsider", group.ID, input); !errors.Is(err, models.ErrGroupAccessForbidden) {
		t.Fatalf("outsider write=%v", err)
	}

	if _, err := groups.Expenses(ctx, "outsider", group.ID); !errors.Is(err, models.ErrGroupAccessForbidden) {
		t.Fatalf("outsider read=%v", err)
	}

	if err := groups.DeleteCategory(ctx, "owner", group.ID, category.ID); err != nil {
		t.Fatal(err)
	}

	if _, err := groups.CreateExpense(ctx, "owner", group.ID, input); !errors.Is(err, models.ErrInvalidExpenseCategory) {
		t.Fatalf("removed category=%v", err)
	}

	if err := groups.Leave(ctx, "payer", group.ID); err != nil {
		t.Fatal(err)
	}

	input.CategoryID = "default_food"
	if _, err := groups.CreateExpense(ctx, "owner", group.ID, input); !errors.Is(err, models.ErrInvalidExpensePayer) {
		t.Fatalf("former payer=%v", err)
	}

	if _, err := groups.Expenses(ctx, "payer", group.ID); !errors.Is(err, models.ErrGroupAccessForbidden) {
		t.Fatalf("former member read=%v", err)
	}

	expenses, err := groups.Expenses(ctx, "owner", group.ID)
	if err != nil || len(expenses) != 1 || expenses[0].CategoryName != "Pet care" || expenses[0].PayerName != "Alice" || expenses[0].PaidBy != "payer" {
		t.Fatalf("history=%+v err=%v", expenses, err)
	}

	other, err := groups.Create(ctx, "owner", models.GroupInput{Name: "Other"})
	if err != nil {
		t.Fatal(err)
	}

	expenses, err = groups.Expenses(ctx, "owner", other.ID)
	if err != nil || len(expenses) != 0 {
		t.Fatalf("cross-group data=%+v err=%v", expenses, err)
	}
}

func TestExpenseCreationRacingGroupDeletion(t *testing.T) {
	_, db, ctx := emulatorStore(t)
	groups := database.NewGroupStore(db)
	group, err := groups.Create(ctx, "owner", models.GroupInput{Name: "Trip"})
	if err != nil {
		t.Fatal(err)
	}

	createResult := make(chan error, 1)
	deleteResult := make(chan error, 1)
	start := make(chan struct{})
	go func() {
		<-start
		_, err := groups.CreateExpense(ctx, "owner", group.ID, models.ExpenseInput{
			Name: "Lunch", CategoryID: "default_food", PaidBy: "owner", AmountMinor: 100,
		})
		createResult <- err
	}()
	go func() {
		<-start
		deleteResult <- groups.Delete(ctx, "owner", group.ID)
	}()
	close(start)

	createErr, deleteErr := <-createResult, <-deleteResult
	if createErr == nil {
		if !errors.Is(deleteErr, models.ErrGroupHasExpenses) {
			t.Fatalf("created expense but deletion=%v", deleteErr)
		}

		expenses, err := groups.Expenses(ctx, "owner", group.ID)
		if err != nil || len(expenses) != 1 {
			t.Fatalf("expenses=%+v err=%v", expenses, err)
		}
		return
	}

	if !errors.Is(createErr, models.ErrGroupNotFound) || deleteErr != nil {
		t.Fatalf("create=%v delete=%v", createErr, deleteErr)
	}

	snapshots, err := db.Collection("groups").Doc(group.ID).Collection("expenses").Documents(ctx).GetAll()
	if err != nil || len(snapshots) != 0 {
		t.Fatalf("orphan expenses=%d err=%v", len(snapshots), err)
	}
}
