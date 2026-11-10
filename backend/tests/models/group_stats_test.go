package models_test

import (
	"errors"
	"math"
	"testing"
	"time"

	"github.com/D4nRusu/band4band/backend/internal/models"
)

func statsExpense(category string, paidBy string, amount int64, shares ...models.ExpenseContribution) models.Expense {
	return models.Expense{
		CategoryID: category, CategoryName: category, CategoryIcon: "food",
		PaidBy: paidBy, AmountMinor: amount, Currency: "EUR", Contributions: shares,
	}
}

func TestBuildGroupStats(t *testing.T) {
	deleted := statsExpense("food", "alice", 5000, models.ExpenseContribution{UID: "alice", AmountMinor: 5000})
	deleted.DeletedAt = time.Now()

	expenses := []models.Expense{
		statsExpense("food", "alice", 900,
			models.ExpenseContribution{UID: "alice", Name: "Alice", AmountMinor: 300},
			models.ExpenseContribution{UID: "bob", Name: "Bob", AmountMinor: 300},
			models.ExpenseContribution{UID: "charlie", Name: "Charlie", AmountMinor: 300}),
		statsExpense("transport", "bob", 1000,
			models.ExpenseContribution{UID: "alice", Name: "Alice", AmountMinor: 0},
			models.ExpenseContribution{UID: "bob", Name: "Bob", AmountMinor: 1000}),
		statsExpense("food", "charlie", 400,
			models.ExpenseContribution{UID: "alice", Name: "Alice", AmountMinor: 400}),
		statsExpense("other", "alice", 200),
		deleted,
	}

	stats, err := models.BuildGroupStats(expenses, "alice")
	if err != nil {
		t.Fatal(err)
	}

	if stats.TotalAmountMinor != 2500 || stats.PersonalTotalAmountMinor != 700 {
		t.Fatalf("stats=%+v", stats)
	}

	wantCategories := []models.CategoryTotal{
		{CategoryID: "food", CategoryName: "food", CategoryIcon: "food", AmountMinor: 1300},
		{CategoryID: "transport", CategoryName: "transport", CategoryIcon: "food", AmountMinor: 1000},
		{CategoryID: "other", CategoryName: "other", CategoryIcon: "food", AmountMinor: 200},
	}
	if len(stats.ByCategory) != len(wantCategories) {
		t.Fatalf("byCategory=%+v", stats.ByCategory)
	}
	for i, want := range wantCategories {
		if stats.ByCategory[i] != want {
			t.Fatalf("byCategory[%d]=%+v, want %+v", i, stats.ByCategory[i], want)
		}
	}

	wantMembers := []models.MemberShareTotal{
		{UID: "bob", Name: "Bob", AmountMinor: 1300},
		{UID: "alice", Name: "Alice", AmountMinor: 700},
		{UID: "charlie", Name: "Charlie", AmountMinor: 300},
	}
	if len(stats.ByMember) != len(wantMembers) {
		t.Fatalf("byMember=%+v", stats.ByMember)
	}
	for i, want := range wantMembers {
		if stats.ByMember[i] != want {
			t.Fatalf("byMember[%d]=%+v, want %+v", i, stats.ByMember[i], want)
		}
	}

	if len(stats.PersonalByCategory) != 1 || stats.PersonalByCategory[0].CategoryID != "food" || stats.PersonalByCategory[0].AmountMinor != 700 {
		t.Fatalf("personalByCategory=%+v", stats.PersonalByCategory)
	}
}

func TestBuildGroupStatsEmpty(t *testing.T) {
	stats, err := models.BuildGroupStats(nil, "alice")
	if err != nil {
		t.Fatal(err)
	}

	if stats.TotalAmountMinor != 0 || stats.ByCategory == nil || stats.ByMember == nil || stats.PersonalByCategory == nil {
		t.Fatalf("stats=%+v", stats)
	}
}

func TestBuildGroupStatsRejectsOverflow(t *testing.T) {
	expenses := []models.Expense{
		statsExpense("food", "alice", math.MaxInt64),
		statsExpense("food", "alice", 1),
	}

	if _, err := models.BuildGroupStats(expenses, "alice"); !errors.Is(err, models.ErrDebtOverflow) {
		t.Fatalf("err=%v", err)
	}
}
