package models_test

import (
	"errors"
	"math"
	"reflect"
	"testing"

	"github.com/D4nRusu/band4band/backend/internal/models"
)

func TestUnequalContributions(t *testing.T) {
	shares := []models.ExpenseContributionInput{{UID: "b", AmountMinor: 700}, {UID: "a", AmountMinor: 300}, {UID: "c", AmountMinor: 0}}
	got, err := models.UnequalContributions(1000, shares)
	if err != nil {
		t.Fatal(err)
	}

	want := []models.ExpenseContribution{{UID: "a", AmountMinor: 300}, {UID: "b", AmountMinor: 700}, {UID: "c", AmountMinor: 0}}
	if !reflect.DeepEqual(got, want) || shares[0].UID != "b" {
		t.Fatalf("allocations=%+v; input=%+v", got, shares)
	}

	if _, err := models.UnequalContributions(models.MaxAmountMinor, []models.ExpenseContributionInput{
		{UID: "a", AmountMinor: models.MaxAmountMinor - 1}, {UID: "b", AmountMinor: 1},
	}); err != nil {
		t.Fatal(err)
	}
}

func TestUnequalContributionsRejectInvalidShares(t *testing.T) {
	for _, tc := range []struct {
		name   string
		shares []models.ExpenseContributionInput
		want   error
	}{
		{"missing", nil, models.ErrInvalidExpenseParticipants},
		{"empty UID", []models.ExpenseContributionInput{{AmountMinor: 1000}}, models.ErrInvalidExpenseParticipants},
		{"duplicate UID", []models.ExpenseContributionInput{{UID: "a", AmountMinor: 500}, {UID: "a", AmountMinor: 500}}, models.ErrInvalidExpenseParticipants},
		{"negative", []models.ExpenseContributionInput{{UID: "a", AmountMinor: -1}, {UID: "b", AmountMinor: 1001}}, models.ErrInvalidExpenseShares},
		{"too little", []models.ExpenseContributionInput{{UID: "a", AmountMinor: 999}}, models.ErrInvalidExpenseShares},
		{"too much", []models.ExpenseContributionInput{{UID: "a", AmountMinor: 1001}}, models.ErrInvalidExpenseShares},
		{"overflow attempt", []models.ExpenseContributionInput{{UID: "a", AmountMinor: math.MaxInt64}, {UID: "b", AmountMinor: math.MaxInt64}}, models.ErrInvalidExpenseShares},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := models.UnequalContributions(1000, tc.shares); !errors.Is(err, tc.want) {
				t.Fatalf("err=%v, want %v", err, tc.want)
			}
		})
	}
}

func TestExpenseSplitModeValidation(t *testing.T) {
	base := models.ExpenseInput{Name: "Dinner", CategoryID: "default_food", PaidBy: "payer", AmountMinor: 1000}
	equal := base
	equal.ParticipantUIDs = []string{"a", "b"}
	normalized, err := equal.Normalize()
	if err != nil || normalized.SplitType != "equal" {
		t.Fatalf("default split=%+v err=%v", normalized, err)
	}

	unequal := base
	unequal.SplitType = "unequal"
	unequal.Contributions = []models.ExpenseContributionInput{{UID: "a", AmountMinor: 1000}}
	if _, err := unequal.Normalize(); err != nil {
		t.Fatal(err)
	}

	for _, input := range []models.ExpenseInput{
		{SplitType: "percent"},
		{SplitType: "equal", ParticipantUIDs: []string{"a"}, Contributions: unequal.Contributions},
		{SplitType: "unequal", ParticipantUIDs: []string{"a"}, Contributions: unequal.Contributions},
	} {
		if _, err := input.Allocations(); !errors.Is(err, models.ErrInvalidExpenseSplit) {
			t.Fatalf("accepted conflicting split=%+v err=%v", input, err)
		}
	}
}
