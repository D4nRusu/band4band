package models

import (
	"errors"
	"math"
	"slices"
	"time"
)

var (
	ErrInvalidDebtGraph   = errors.New("invalid debt graph")
	ErrDebtOverflow       = errors.New("debt amount exceeds the supported integer range")
	ErrDebtUnderflow      = errors.New("expense reversal exceeds the recorded debt")
	ErrInvalidDebtExpense = errors.New("expense allocations are inconsistent with the debt graph")
)

type DebtEdge struct {
	FromUID     string `firestore:"fromUID"`
	ToUID       string `firestore:"toUID"`
	AmountMinor int64  `firestore:"amountMinor"`
}

type DebtGraph struct {
	Currency  string     `firestore:"currency"`
	Edges     []DebtEdge `firestore:"edges"`
	UpdatedAt time.Time  `firestore:"updatedAt"`
}

func (graph DebtGraph) Validate() error {
	if graph.Currency == "" {
		return ErrInvalidDebtGraph
	}

	seen := make(map[[2]string]bool, len(graph.Edges))
	for _, edge := range graph.Edges {
		pair := [2]string{edge.FromUID, edge.ToUID}
		if edge.FromUID == "" || edge.ToUID == "" || edge.FromUID == edge.ToUID || edge.AmountMinor <= 0 || seen[pair] {
			return ErrInvalidDebtGraph
		}

		seen[pair] = true
	}

	return nil
}

// AdjustEdge adds to or subtracts from one directed debt. Zero balances remove edges.
// The reverse direction is independent; this does not simplify or net debts.
func (graph *DebtGraph) AdjustEdge(fromUID, toUID string, delta int64) error {
	if fromUID == "" || toUID == "" || fromUID == toUID {
		return ErrInvalidDebtGraph
	}

	if delta == 0 {
		return nil
	}

	for i, edge := range graph.Edges {
		if edge.FromUID != fromUID || edge.ToUID != toUID {
			continue
		}

		if delta > 0 && delta > math.MaxInt64-edge.AmountMinor {
			return ErrDebtOverflow
		}

		if delta < -edge.AmountMinor {
			return ErrDebtUnderflow
		}

		balance := edge.AmountMinor + delta
		if balance == 0 {
			graph.Edges = slices.Delete(graph.Edges, i, i+1)
		} else {
			graph.Edges[i].AmountMinor = balance
		}

		return nil
	}

	if delta < 0 {
		return ErrDebtUnderflow
	}

	graph.Edges = append(graph.Edges, DebtEdge{FromUID: fromUID, ToUID: toUID, AmountMinor: delta})
	return nil
}

// ApplyExpense changes only the expense's contributor-to-payer edges. Failure leaves
// the graph unchanged; Firestore callers also commit the expense and graph atomically.
func (graph *DebtGraph) ApplyExpense(expense Expense, reverse bool) error {
	if !expense.DeletedAt.IsZero() || len(expense.Contributions) == 0 {
		return nil
	}

	if expense.Currency != graph.Currency || expense.PaidBy == "" || expense.AmountMinor <= 0 {
		return ErrInvalidDebtExpense
	}

	seen := make(map[string]bool, len(expense.Contributions))
	var total int64
	for _, share := range expense.Contributions {
		if share.UID == "" || seen[share.UID] || share.AmountMinor < 0 || share.AmountMinor > expense.AmountMinor-total {
			return ErrInvalidDebtExpense
		}

		seen[share.UID] = true
		total += share.AmountMinor
	}

	if total != expense.AmountMinor {
		return ErrInvalidDebtExpense
	}

	updated := *graph
	updated.Edges = slices.Clone(graph.Edges)
	for _, share := range expense.Contributions {
		if share.UID == expense.PaidBy || share.AmountMinor == 0 {
			continue
		}

		delta := share.AmountMinor
		if reverse {
			delta = -delta
		}

		if err := updated.AdjustEdge(share.UID, expense.PaidBy, delta); err != nil {
			return err
		}
	}

	graph.Edges = updated.Edges
	return nil
}
