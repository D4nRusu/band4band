package services_test

import (
	"errors"
	"math"
	"math/big"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/D4nRusu/band4band/backend/internal/models"
	"github.com/D4nRusu/band4band/backend/internal/services"
)

func TestSimplifyDebtCycles(t *testing.T) {
	for _, tc := range []struct {
		name        string
		input, want map[string][]models.DebtNeighbor
	}{
		{
			name:  "empty graph",
			input: map[string][]models.DebtNeighbor{},
			want:  map[string][]models.DebtNeighbor{},
		},
		{
			name:  "acyclic chain collapses",
			input: map[string][]models.DebtNeighbor{"a": {{ToUID: "b", AmountMinor: 1000}}, "b": {{ToUID: "c", AmountMinor: 500}}},
			want:  map[string][]models.DebtNeighbor{"a": {{ToUID: "b", AmountMinor: 500}, {ToUID: "c", AmountMinor: 500}}},
		},
		{
			name:  "opposite debts form a two node cycle",
			input: map[string][]models.DebtNeighbor{"a": {{ToUID: "b", AmountMinor: 1000}}, "b": {{ToUID: "a", AmountMinor: 400}}},
			want:  map[string][]models.DebtNeighbor{"a": {{ToUID: "b", AmountMinor: 600}}},
		},
		{
			name: "three node cycle subtracts its smallest edge",
			input: map[string][]models.DebtNeighbor{
				"a": {{ToUID: "b", AmountMinor: 1000}},
				"b": {{ToUID: "c", AmountMinor: 700}},
				"c": {{ToUID: "a", AmountMinor: 400}},
			},
			want: map[string][]models.DebtNeighbor{"a": {{ToUID: "b", AmountMinor: 300}, {ToUID: "c", AmountMinor: 300}}},
		},
		{
			name: "equal edges all disappear",
			input: map[string][]models.DebtNeighbor{
				"a": {{ToUID: "b", AmountMinor: 100}},
				"b": {{ToUID: "c", AmountMinor: 100}},
				"c": {{ToUID: "a", AmountMinor: 100}},
			},
			want: map[string][]models.DebtNeighbor{},
		},
		{
			name: "overlapping cycles and disconnected debts",
			input: map[string][]models.DebtNeighbor{
				"a": {{ToUID: "b", AmountMinor: 1000}},
				"b": {{ToUID: "c", AmountMinor: 600}, {ToUID: "a", AmountMinor: 400}},
				"c": {{ToUID: "a", AmountMinor: 600}},
				"d": {{ToUID: "e", AmountMinor: 900}},
				"x": {{ToUID: "y", AmountMinor: 50}},
				"y": {{ToUID: "x", AmountMinor: 50}},
			},
			want: map[string][]models.DebtNeighbor{"d": {{ToUID: "e", AmountMinor: 900}}},
		},
		{
			name: "cycle below a noncyclic prefix",
			input: map[string][]models.DebtNeighbor{
				"a": {{ToUID: "b", AmountMinor: 900}},
				"b": {{ToUID: "c", AmountMinor: 700}},
				"c": {{ToUID: "d", AmountMinor: 500}},
				"d": {{ToUID: "b", AmountMinor: 300}},
			},
			want: map[string][]models.DebtNeighbor{
				"a": {{ToUID: "b", AmountMinor: 500}, {ToUID: "c", AmountMinor: 200}, {ToUID: "d", AmountMinor: 200}},
			},
		},
		{
			name: "large amounts do not overflow",
			input: map[string][]models.DebtNeighbor{
				"a": {{ToUID: "b", AmountMinor: math.MaxInt64}},
				"b": {{ToUID: "a", AmountMinor: math.MaxInt64 - 1}},
			},
			want: map[string][]models.DebtNeighbor{"a": {{ToUID: "b", AmountMinor: 1}}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			graph := models.DebtGraph{Currency: "EUR", Adjacency: tc.input, UpdatedAt: time.Unix(123, 0)}
			before := copyDebts(graph.Adjacency)
			simplified, err := services.SimplifyDebts(graph)
			if err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(simplified.Adjacency, tc.want) {
				t.Fatalf("got %+v, want %+v", simplified.Adjacency, tc.want)
			}

			if !reflect.DeepEqual(graph.Adjacency, before) {
				t.Fatal("simplification modified the raw graph")
			}

			if simplified.Currency != graph.Currency || !simplified.UpdatedAt.Equal(graph.UpdatedAt) {
				t.Fatal("simplification changed source metadata")
			}

			if err := simplified.Validate(); err != nil {
				t.Fatal(err)
			}

			assertNetBalances(t, graph, simplified)
			again, err := services.SimplifyDebts(simplified)
			if err != nil || !reflect.DeepEqual(again, simplified) {
				t.Fatalf("simplification is not idempotent: %+v, %v", again, err)
			}
		})
	}
}

func TestSimplificationCopiesNeighborLists(t *testing.T) {
	graph := models.DebtGraph{Currency: "EUR", Adjacency: map[string][]models.DebtNeighbor{
		"a": {{ToUID: "c", AmountMinor: 200}, {ToUID: "b", AmountMinor: 100}},
	}}
	before := copyDebts(graph.Adjacency)
	simplified, err := services.SimplifyDebts(graph)
	if err != nil {
		t.Fatal(err)
	}

	if err := simplified.AdjustEdge("a", "b", -50); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(graph.Adjacency, before) {
		t.Fatal("returned graph shares neighbor storage with input")
	}
}

func TestSimplificationRejectsInvalidGraphs(t *testing.T) {
	for _, graph := range []models.DebtGraph{
		{},
		{Currency: "EUR", Adjacency: map[string][]models.DebtNeighbor{"a": {{ToUID: "a", AmountMinor: 100}}}},
		{Currency: "EUR", Adjacency: map[string][]models.DebtNeighbor{"a": {{ToUID: "b", AmountMinor: -1}}}},
		{Currency: "EUR", Adjacency: map[string][]models.DebtNeighbor{"a": {{ToUID: "b", AmountMinor: 100}, {ToUID: "b", AmountMinor: 200}}}},
	} {
		if _, err := services.SimplifyDebts(graph); !errors.Is(err, models.ErrInvalidDebtGraph) {
			t.Fatalf("accepted invalid graph: %+v, err=%v", graph, err)
		}
	}
}

func copyDebts(adjacency map[string][]models.DebtNeighbor) map[string][]models.DebtNeighbor {
	copy := make(map[string][]models.DebtNeighbor, len(adjacency))
	for uid, debts := range adjacency {
		copy[uid] = slices.Clone(debts)
	}

	return copy
}

func assertNetBalances(t *testing.T, before, after models.DebtGraph) {
	t.Helper()
	balances := make(map[string]*big.Int)
	for i, graph := range []models.DebtGraph{before, after} {
		for from, debts := range graph.Adjacency {
			for _, debt := range debts {
				if balances[from] == nil {
					balances[from] = new(big.Int)
				}

				if balances[debt.ToUID] == nil {
					balances[debt.ToUID] = new(big.Int)
				}

				amount := big.NewInt(debt.AmountMinor)
				if i == 1 {
					amount.Neg(amount)
				}

				balances[from].Sub(balances[from], amount)
				balances[debt.ToUID].Add(balances[debt.ToUID], amount)
			}
		}
	}

	for uid, difference := range balances {
		if difference.Sign() != 0 {
			t.Fatalf("net balance changed for %s by %s cents", uid, difference)
		}
	}
}
