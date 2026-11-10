package models_test

import (
	"errors"
	"math"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/D4nRusu/band4band/backend/internal/models"
)

func TestDirectedDebtEdgeChanges(t *testing.T) {
	graph := models.DebtGraph{Currency: "EUR"}
	for _, change := range []struct {
		from, to string
		delta    int64
	}{
		{"alice", "bob", 1000},
		{"alice", "bob", 500},
		{"bob", "alice", 400},
		{"alice", "bob", -500},
	} {
		if err := graph.AdjustEdge(change.from, change.to, change.delta); err != nil {
			t.Fatal(err)
		}
	}

	if len(graph.Adjacency) != 2 || debtAmount(graph.Adjacency, "alice", "bob") != 1000 || debtAmount(graph.Adjacency, "bob", "alice") != 400 {
		t.Fatalf("opposite directions were netted: %+v", graph.Adjacency)
	}

	if err := graph.AdjustEdge("alice", "bob", -1000); err != nil {
		t.Fatal(err)
	}

	if len(graph.Adjacency) != 1 || debtAmount(graph.Adjacency, "bob", "alice") != 400 {
		t.Fatalf("edges=%+v", graph.Adjacency)
	}

	if err := graph.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestDebtEdgeRejectsInvalidArithmetic(t *testing.T) {
	for _, tc := range []struct {
		name, from, to string
		delta          int64
		expected       error
	}{
		{"overflow", "a", "b", 1, models.ErrDebtOverflow},
		{"underflow", "b", "a", -1, models.ErrDebtUnderflow},
		{"minimum integer", "a", "b", math.MinInt64, models.ErrDebtUnderflow},
		{"self debt", "a", "a", 1, models.ErrInvalidDebtGraph},
		{"missing user", "", "b", 1, models.ErrInvalidDebtGraph},
	} {
		t.Run(tc.name, func(t *testing.T) {
			graph := models.DebtGraph{Currency: "EUR", Adjacency: map[string][]models.DebtNeighbor{"a": {{ToUID: "b", AmountMinor: math.MaxInt64}}}}
			before := cloneAdjacency(graph.Adjacency)
			if err := graph.AdjustEdge(tc.from, tc.to, tc.delta); !errors.Is(err, tc.expected) {
				t.Fatalf("err=%v", err)
			}

			if !reflect.DeepEqual(before, graph.Adjacency) {
				t.Fatal("failed adjustment changed graph")
			}
		})
	}
}

func TestExpenseGraphApplicationAndReversal(t *testing.T) {
	graph := models.DebtGraph{Currency: "EUR"}
	expense := models.Expense{PaidBy: "alice", Currency: "EUR", AmountMinor: 1500,
		Contributions: []models.ExpenseContribution{
			{UID: "alice", AmountMinor: 500}, {UID: "bob", AmountMinor: 500}, {UID: "charlie", AmountMinor: 500},
		},
	}
	if err := graph.ApplyExpense(expense, false); err != nil {
		t.Fatal(err)
	}

	if err := graph.ApplyExpense(expense, false); err != nil {
		t.Fatal(err)
	}

	if len(graph.Adjacency) != 2 || debtAmount(graph.Adjacency, "bob", "alice") != 1000 || debtAmount(graph.Adjacency, "charlie", "alice") != 1000 {
		t.Fatalf("edges=%+v", graph.Adjacency)
	}

	if err := graph.ApplyExpense(expense, true); err != nil {
		t.Fatal(err)
	}

	if len(graph.Adjacency) != 2 || debtAmount(graph.Adjacency, "bob", "alice") != 500 || debtAmount(graph.Adjacency, "charlie", "alice") != 500 {
		t.Fatalf("edges=%+v", graph.Adjacency)
	}

	if err := graph.ApplyExpense(expense, true); err != nil {
		t.Fatal(err)
	}

	if len(graph.Adjacency) != 0 {
		t.Fatalf("edges=%+v", graph.Adjacency)
	}

	if err := graph.ApplyExpense(expense, true); !errors.Is(err, models.ErrDebtUnderflow) {
		t.Fatalf("underflow=%v", err)
	}
}

func TestGraphSkipsSelfZeroDeletedAndLegacyAllocations(t *testing.T) {
	graph := models.DebtGraph{Currency: "EUR"}
	for _, expense := range []models.Expense{
		{PaidBy: "a", Currency: "EUR", AmountMinor: 1, Contributions: []models.ExpenseContribution{{UID: "a", AmountMinor: 1}, {UID: "b", AmountMinor: 0}}},
		{PaidBy: "a", Currency: "EUR", AmountMinor: 100},
		{PaidBy: "a", Currency: "EUR", AmountMinor: 100, DeletedAt: time.Now(), Contributions: []models.ExpenseContribution{{UID: "b", AmountMinor: 100}}},
	} {
		if err := graph.ApplyExpense(expense, false); err != nil {
			t.Fatal(err)
		}
	}

	if len(graph.Adjacency) != 0 {
		t.Fatalf("unexpected edges=%+v", graph.Adjacency)
	}

	excludedPayer := models.Expense{PaidBy: "a", Currency: "EUR", AmountMinor: 100,
		Contributions: []models.ExpenseContribution{{UID: "b", AmountMinor: 100}},
	}
	if err := graph.ApplyExpense(excludedPayer, false); err != nil {
		t.Fatal(err)
	}

	if len(graph.Adjacency) != 1 || debtAmount(graph.Adjacency, "b", "a") != 100 {
		t.Fatalf("edges=%+v", graph.Adjacency)
	}
}

func TestExpenseGraphApplicationIsAtomic(t *testing.T) {
	graph := models.DebtGraph{Currency: "EUR", Adjacency: map[string][]models.DebtNeighbor{"bob": {{ToUID: "alice", AmountMinor: 100}}}}
	before := cloneAdjacency(graph.Adjacency)
	expense := models.Expense{PaidBy: "alice", Currency: "EUR", AmountMinor: 200,
		Contributions: []models.ExpenseContribution{{UID: "bob", AmountMinor: 100}, {UID: "charlie", AmountMinor: 100}},
	}
	if err := graph.ApplyExpense(expense, true); !errors.Is(err, models.ErrDebtUnderflow) {
		t.Fatalf("err=%v", err)
	}

	if !reflect.DeepEqual(before, graph.Adjacency) {
		t.Fatal("partial reversal changed graph")
	}

	for _, expense := range []models.Expense{
		{PaidBy: "alice", Currency: "USD", AmountMinor: 100, Contributions: []models.ExpenseContribution{{UID: "bob", AmountMinor: 100}}},
		{PaidBy: "alice", Currency: "EUR", AmountMinor: 100, Contributions: []models.ExpenseContribution{{UID: "bob", AmountMinor: 101}}},
		{PaidBy: "alice", Currency: "EUR", AmountMinor: 100, Contributions: []models.ExpenseContribution{{UID: "bob", AmountMinor: 50}, {UID: "bob", AmountMinor: 50}}},
	} {
		if err := graph.ApplyExpense(expense, false); !errors.Is(err, models.ErrInvalidDebtExpense) {
			t.Fatalf("err=%v", err)
		}
	}
}

func cloneAdjacency(adjacency map[string][]models.DebtNeighbor) map[string][]models.DebtNeighbor {
	copy := make(map[string][]models.DebtNeighbor, len(adjacency))
	for uid, creditors := range adjacency {
		copy[uid] = slices.Clone(creditors)
	}

	return copy
}

func TestDebtGraphValidation(t *testing.T) {
	for _, adjacency := range []map[string][]models.DebtNeighbor{
		{"": {{ToUID: "bob", AmountMinor: 100}}},
		{"alice": {}},
		{"alice": {{ToUID: "bob", AmountMinor: 100}, {ToUID: "bob", AmountMinor: 200}}},
		{"alice": {{ToUID: "", AmountMinor: 100}}},
		{"alice": {{ToUID: "alice", AmountMinor: 100}}},
		{"alice": {{ToUID: "bob", AmountMinor: 0}}},
		{"alice": {{ToUID: "bob", AmountMinor: -100}}},
	} {
		graph := models.DebtGraph{Currency: "EUR", Adjacency: adjacency}
		if !errors.Is(graph.Validate(), models.ErrInvalidDebtGraph) {
			t.Fatalf("accepted invalid adjacency: %+v", adjacency)
		}
	}
}

func debtAmount(adjacency map[string][]models.DebtNeighbor, from, to string) int64 {
	for _, debt := range adjacency[from] {
		if debt.ToUID == to {
			return debt.AmountMinor
		}
	}

	return 0
}

func TestDebtorHasMultipleCreditors(t *testing.T) {
	graph := models.DebtGraph{Currency: "EUR"}
	for _, to := range []string{"bob", "charlie"} {
		if err := graph.AdjustEdge("alice", to, 100); err != nil {
			t.Fatal(err)
		}
	}

	if err := graph.AdjustEdge("alice", "bob", -100); err != nil {
		t.Fatal(err)
	}

	if len(graph.Adjacency["alice"]) != 1 || graph.Adjacency["alice"][0].ToUID != "charlie" {
		t.Fatalf("unexpected remaining debts: %+v", graph.Adjacency)
	}
}
