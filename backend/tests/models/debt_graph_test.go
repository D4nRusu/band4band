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

	if len(graph.Edges) != 2 || graph.Edges[0].AmountMinor != 1000 || graph.Edges[1].AmountMinor != 400 {
		t.Fatalf("opposite directions were netted: %+v", graph.Edges)
	}

	if err := graph.AdjustEdge("alice", "bob", -1000); err != nil {
		t.Fatal(err)
	}

	if len(graph.Edges) != 1 || graph.Edges[0].FromUID != "bob" {
		t.Fatalf("edges=%+v", graph.Edges)
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
			graph := models.DebtGraph{Currency: "EUR", Edges: []models.DebtEdge{{FromUID: "a", ToUID: "b", AmountMinor: math.MaxInt64}}}
			before := slices.Clone(graph.Edges)
			if err := graph.AdjustEdge(tc.from, tc.to, tc.delta); !errors.Is(err, tc.expected) {
				t.Fatalf("err=%v", err)
			}

			if !slices.Equal(before, graph.Edges) {
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

	if len(graph.Edges) != 2 || graph.Edges[0].AmountMinor != 1000 || graph.Edges[1].AmountMinor != 1000 {
		t.Fatalf("edges=%+v", graph.Edges)
	}

	if err := graph.ApplyExpense(expense, true); err != nil {
		t.Fatal(err)
	}

	if len(graph.Edges) != 2 || graph.Edges[0].AmountMinor != 500 {
		t.Fatalf("edges=%+v", graph.Edges)
	}

	if err := graph.ApplyExpense(expense, true); err != nil {
		t.Fatal(err)
	}

	if len(graph.Edges) != 0 {
		t.Fatalf("edges=%+v", graph.Edges)
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

	if len(graph.Edges) != 0 {
		t.Fatalf("unexpected edges=%+v", graph.Edges)
	}

	excludedPayer := models.Expense{PaidBy: "a", Currency: "EUR", AmountMinor: 100,
		Contributions: []models.ExpenseContribution{{UID: "b", AmountMinor: 100}},
	}
	if err := graph.ApplyExpense(excludedPayer, false); err != nil {
		t.Fatal(err)
	}

	if len(graph.Edges) != 1 || graph.Edges[0].AmountMinor != 100 {
		t.Fatalf("edges=%+v", graph.Edges)
	}
}

func TestExpenseGraphApplicationIsAtomic(t *testing.T) {
	graph := models.DebtGraph{Currency: "EUR", Edges: []models.DebtEdge{{FromUID: "bob", ToUID: "alice", AmountMinor: 100}}}
	before := slices.Clone(graph.Edges)
	expense := models.Expense{PaidBy: "alice", Currency: "EUR", AmountMinor: 200,
		Contributions: []models.ExpenseContribution{{UID: "bob", AmountMinor: 100}, {UID: "charlie", AmountMinor: 100}},
	}
	if err := graph.ApplyExpense(expense, true); !errors.Is(err, models.ErrDebtUnderflow) {
		t.Fatalf("err=%v", err)
	}

	if !reflect.DeepEqual(before, graph.Edges) {
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
