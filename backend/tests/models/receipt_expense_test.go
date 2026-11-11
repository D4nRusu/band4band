package models_test

import (
	"reflect"
	"testing"

	"github.com/D4nRusu/band4band/backend/internal/models"
)

func receiptExpenseInput() models.ReceiptExpenseInput {
	return models.ReceiptExpenseInput{
		Name: " Receipt ", CategoryID: "default_other", PaidBy: "payer",
		Currency: "EUR", TotalMinor: 1120,
		Items: []models.ReceiptExpenseItem{
			{AmountMinor: 1001, Units: 3, Assignments: []models.ReceiptUnitAssignment{{UID: "b", Units: 1}, {UID: "a", Units: 2}}},
			{AmountMinor: 119, Units: 1, Assignments: []models.ReceiptUnitAssignment{{UID: "b", Units: 1}}},
		},
	}
}

func TestReceiptExpenseCombinesUnitsAndAssignsExactCents(t *testing.T) {
	input := receiptExpenseInput()
	expense, err := input.Expense("EUR")
	if err != nil {
		t.Fatal(err)
	}

	want := []models.ExpenseContributionInput{{UID: "a", AmountMinor: 668}, {UID: "b", AmountMinor: 452}}
	if expense.Name != "Receipt" || expense.SplitType != "unequal" || expense.AmountMinor != 1120 || !reflect.DeepEqual(expense.Contributions, want) {
		t.Fatalf("expense=%+v", expense)
	}

	// Caller order must not affect cent allocation or mutate the input.
	input.Items[0].Assignments[0], input.Items[0].Assignments[1] = input.Items[0].Assignments[1], input.Items[0].Assignments[0]
	reordered, err := input.Expense("EUR")
	if err != nil || !reflect.DeepEqual(reordered.Contributions, want) {
		t.Fatalf("reordered=%+v err=%v", reordered, err)
	}
}

func TestReceiptExpenseWholeLineDiscountAndLargestAmount(t *testing.T) {
	input := receiptExpenseInput()
	input.Items[0].Assignments = []models.ReceiptUnitAssignment{{UID: "b", Units: 3}}
	input.Items = append(input.Items, models.ReceiptExpenseItem{
		AmountMinor: -120, Units: 1, Assignments: []models.ReceiptUnitAssignment{{UID: "b", Units: 1}},
	})
	input.TotalMinor = 1000
	expense, err := input.Expense("EUR")
	if err != nil || len(expense.Contributions) != 1 || expense.Contributions[0].AmountMinor != 1000 {
		t.Fatalf("expense=%+v err=%v", expense, err)
	}

	input.Items = []models.ReceiptExpenseItem{{
		AmountMinor: models.MaxAmountMinor, Units: models.MaxAmountMinor,
		Assignments: []models.ReceiptUnitAssignment{{UID: "a", Units: models.MaxAmountMinor - 1}, {UID: "b", Units: 1}},
	}}
	input.TotalMinor = models.MaxAmountMinor
	if _, err := input.Expense("EUR"); err != nil {
		t.Fatal(err)
	}
}

func TestReceiptExpenseRejectsIncompleteOrUnsafeReview(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*models.ReceiptExpenseInput)
	}{
		{"missing items", func(i *models.ReceiptExpenseInput) { i.Items = nil }},
		{"unassigned units", func(i *models.ReceiptExpenseInput) { i.Items[0].Assignments[1].Units = 1 }},
		{"too many units", func(i *models.ReceiptExpenseInput) { i.Items[0].Assignments[1].Units = 3 }},
		{"zero units", func(i *models.ReceiptExpenseInput) { i.Items[0].Units = 0 }},
		{"duplicate member", func(i *models.ReceiptExpenseInput) { i.Items[0].Assignments[1].UID = "b" }},
		{"missing member", func(i *models.ReceiptExpenseInput) { i.Items[0].Assignments[1].UID = "" }},
		{"wrong total", func(i *models.ReceiptExpenseInput) { i.TotalMinor++ }},
		{"currency needs confirmation", func(i *models.ReceiptExpenseInput) { i.Currency = "RON" }},
		{"oversized amount", func(i *models.ReceiptExpenseInput) { i.Items[0].AmountMinor = models.MaxAmountMinor + 1 }},
		{"negative final share", func(i *models.ReceiptExpenseInput) {
			i.Items = append(i.Items, models.ReceiptExpenseItem{
				AmountMinor: -1, Units: 1, Assignments: []models.ReceiptUnitAssignment{{UID: "discount-only", Units: 1}},
			})
			i.TotalMinor--
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := receiptExpenseInput()
			tc.change(&input)
			if _, err := input.Expense("EUR"); err == nil {
				t.Fatal("invalid review was accepted")
			}
		})
	}
}

func TestReceiptExpenseExplicitCurrencyConfirmation(t *testing.T) {
	input := receiptExpenseInput()
	input.Currency = "RON"
	input.ConfirmCurrency = true
	expense, err := input.Expense("EUR")
	if err != nil || expense.AmountMinor != input.TotalMinor {
		t.Fatalf("expense=%+v err=%v", expense, err)
	}
}
