package models_test

import (
	"testing"

	"github.com/D4nRusu/band4band/backend/internal/models"
)

func TestExpensePositionForViewer(t *testing.T) {
	expense := models.Expense{
		PaidBy: "payer", CreatedBy: "creator", AmountMinor: 5000, SplitType: "equal",
		Contributions: []models.ExpenseContribution{
			{UID: "payer", AmountMinor: 1000},
			{UID: "usr2", AmountMinor: 1000},
			{UID: "usr3", AmountMinor: 1000},
			{UID: "usr4", AmountMinor: 1000},
			{UID: "usr5", AmountMinor: 1000},
		},
	}
	for _, tc := range []struct {
		uid, kind string
		amount    int64
	}{
		{"payer", "lent", 4000},
		{"usr2", "borrowed", 1000},
		{"usr3", "borrowed", 1000},
		{"creator", "not_involved", 0},
		{"outsider", "not_involved", 0},
	} {
		t.Run(tc.uid, func(t *testing.T) {
			got := expense.PositionFor(tc.uid)
			if got.Kind != tc.kind || got.AmountMinor != tc.amount {
				t.Fatalf("got=%+v", got)
			}
		})
	}

	var net int64
	for _, uid := range []string{"payer", "usr2", "usr3", "usr4", "usr5"} {
		position := expense.PositionFor(uid)
		if position.Kind == "lent" {
			net += position.AmountMinor
		}
		if position.Kind == "borrowed" {
			net -= position.AmountMinor
		}
	}

	if net != 0 {
		t.Fatalf("positions do not balance: %d", net)
	}
}

func TestExpensePositionSpecialCases(t *testing.T) {
	for _, tc := range []struct {
		name, viewer, kind string
		expense            models.Expense
		amount             int64
	}{
		{"payer excluded", "payer", "lent", models.Expense{PaidBy: "payer", AmountMinor: 5000, Contributions: []models.ExpenseContribution{
			{UID: "usr2", AmountMinor: 2500}, {UID: "usr3", AmountMinor: 2500},
		}}, 5000},
		{"payer only", "payer", "none", models.Expense{PaidBy: "payer", AmountMinor: 5000, Contributions: []models.ExpenseContribution{
			{UID: "payer", AmountMinor: 5000},
		}}, 0},
		{"zero-cent participant", "usr3", "none", models.Expense{PaidBy: "payer", AmountMinor: 1, Contributions: []models.ExpenseContribution{
			{UID: "usr2", AmountMinor: 1}, {UID: "usr3", AmountMinor: 0},
		}}, 0},
		{"legacy payer", "payer", "unallocated", models.Expense{PaidBy: "payer", AmountMinor: 5000}, 0},
		{"legacy viewer", "viewer", "unallocated", models.Expense{PaidBy: "payer", AmountMinor: 5000}, 0},
		{"rounded share", "usr2", "borrowed", models.Expense{PaidBy: "payer", AmountMinor: 1000, Contributions: []models.ExpenseContribution{
			{UID: "usr2", AmountMinor: 334}, {UID: "usr3", AmountMinor: 333}, {UID: "usr4", AmountMinor: 333},
		}}, 334},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.expense.PositionFor(tc.viewer)
			if got.Kind != tc.kind || got.AmountMinor != tc.amount {
				t.Fatalf("got=%+v", got)
			}
		})
	}
}
