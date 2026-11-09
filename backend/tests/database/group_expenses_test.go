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

	input := models.ExpenseInput{Name: " Supplies ", CategoryID: category.ID, PaidBy: "payer", AmountMinor: 1250, ParticipantUIDs: []string{"owner", "payer"}}
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
			Name: "Lunch", CategoryID: "default_food", PaidBy: "owner", AmountMinor: 100, ParticipantUIDs: []string{"owner"},
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

func TestExpenseDeletionPreservesHistory(t *testing.T) {
	_, db, ctx := emulatorStore(t)
	groups := database.NewGroupStore(db)
	group, err := groups.Create(ctx, "owner", models.GroupInput{Name: "Trip"})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := db.Collection("groups").Doc(group.ID).Update(ctx, []firestore.Update{
		{Path: "memberUIDs", Value: firestore.ArrayUnion("member", "other-member")},
	}); err != nil {
		t.Fatal(err)
	}

	expense, err := groups.CreateExpense(ctx, "owner", group.ID, models.ExpenseInput{
		Name: "Lunch", CategoryID: "default_food", PaidBy: "owner", AmountMinor: 1250, ParticipantUIDs: []string{"owner"},
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := groups.DeleteExpense(ctx, "outsider", group.ID, expense.ID); !errors.Is(err, models.ErrGroupAccessForbidden) {
		t.Fatalf("outsider deletion=%v", err)
	}

	otherGroup, err := groups.Create(ctx, "member", models.GroupInput{Name: "Other"})
	if err != nil {
		t.Fatal(err)
	}

	if err := groups.DeleteExpense(ctx, "member", otherGroup.ID, expense.ID); err != nil {
		t.Fatal(err)
	}

	visible, err := groups.Expenses(ctx, "owner", group.ID)
	if err != nil || len(visible) != 1 {
		t.Fatalf("cross-group delete changed expense: %+v %v", visible, err)
	}

	if err := groups.DeleteExpense(ctx, "member", group.ID, expense.ID); err != nil {
		t.Fatal(err)
	}

	ref := db.Collection("groups").Doc(group.ID).Collection("expenses").Doc(expense.ID)
	snapshot, err := ref.Get(ctx)
	if err != nil {
		t.Fatal(err)
	}

	var deleted models.Expense
	if err := snapshot.DataTo(&deleted); err != nil {
		t.Fatal(err)
	}

	if deleted.DeletedBy != "member" || deleted.DeletedAt.IsZero() || deleted.Name != expense.Name || deleted.AmountMinor != 1250 || deleted.CreatedBy != "owner" || !deleted.CreatedAt.Equal(expense.CreatedAt) {
		t.Fatalf("audit record=%+v", deleted)
	}

	if err := groups.DeleteExpense(ctx, "other-member", group.ID, expense.ID); err != nil {
		t.Fatal(err)
	}

	repeated, err := ref.Get(ctx)
	if err != nil {
		t.Fatal(err)
	}

	var afterRetry models.Expense
	if err := repeated.DataTo(&afterRetry); err != nil {
		t.Fatal(err)
	}

	if afterRetry.DeletedBy != deleted.DeletedBy || !afterRetry.DeletedAt.Equal(deleted.DeletedAt) {
		t.Fatal("retry overwrote original deletion metadata")
	}

	visible, err = groups.Expenses(ctx, "owner", group.ID)
	if err != nil || len(visible) != 0 {
		t.Fatalf("deleted expense still visible=%+v err=%v", visible, err)
	}

	if err := groups.Delete(ctx, "owner", group.ID); !errors.Is(err, models.ErrGroupHasExpenses) {
		t.Fatalf("history group deletion=%v", err)
	}

	if err := groups.Leave(ctx, "member", group.ID); err != nil {
		t.Fatal(err)
	}

	if err := groups.DeleteExpense(ctx, "member", group.ID, expense.ID); !errors.Is(err, models.ErrGroupAccessForbidden) {
		t.Fatalf("former member deletion=%v", err)
	}
}
