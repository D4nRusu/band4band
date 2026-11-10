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
	ErrInvalidDebtUser    = errors.New("the selected user must be a current group member")
	ErrInvalidDebtAmount  = errors.New("amount must be a positive integer in cents within the supported range")
	ErrSettlementNotParty = errors.New("you can only record a settlement you paid or received")
)

type DebtNeighbor struct {
	ToUID       string `firestore:"toUID"`
	AmountMinor int64  `firestore:"amountMinor"`
}

type DebtGraph struct {
	Currency  string                    `firestore:"currency"`
	Adjacency map[string][]DebtNeighbor `firestore:"adjacency"`
	UpdatedAt time.Time                 `firestore:"updatedAt"`
}

type UserOwedDebt struct {
	ToUID       string `json:"toUID"`
	AmountMinor int64  `json:"amountMinor"`
}

type UserOwedSummary struct {
	UserUID          string         `json:"userUID"`
	Currency         string         `json:"currency"`
	TotalAmountMinor int64          `json:"totalAmountMinor"`
	Debts            []UserOwedDebt `json:"debts"`
}

type UserReceivableDebt struct {
	FromUID     string `json:"fromUID"`
	AmountMinor int64  `json:"amountMinor"`
}

type UserReceivableSummary struct {
	UserUID          string               `json:"userUID"`
	Currency         string               `json:"currency"`
	TotalAmountMinor int64                `json:"totalAmountMinor"`
	Debts            []UserReceivableDebt `json:"debts"`
}

func (graph DebtGraph) Validate() error {
	if graph.Currency == "" {
		return ErrInvalidDebtGraph
	}

	for fromUID, creditors := range graph.Adjacency {
		if fromUID == "" || len(creditors) == 0 {
			return ErrInvalidDebtGraph
		}

		seen := make(map[string]bool, len(creditors))
		for _, debt := range creditors {
			if debt.ToUID == "" || fromUID == debt.ToUID || debt.AmountMinor <= 0 || seen[debt.ToUID] {
				return ErrInvalidDebtGraph
			}

			seen[debt.ToUID] = true
		}
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

	creditors := graph.Adjacency[fromUID]
	index := slices.IndexFunc(creditors, func(debt DebtNeighbor) bool {
		return debt.ToUID == toUID
	})

	var amount int64
	if index >= 0 {
		amount = creditors[index].AmountMinor
	}

	if delta > 0 && delta > math.MaxInt64-amount {
		return ErrDebtOverflow
	}

	if delta < -amount {
		return ErrDebtUnderflow
	}

	balance := amount + delta
	if balance == 0 {
		creditors = slices.Delete(creditors, index, index+1)
		if len(creditors) == 0 {
			delete(graph.Adjacency, fromUID)
		} else {
			graph.Adjacency[fromUID] = creditors
		}

		return nil
	}

	if graph.Adjacency == nil {
		graph.Adjacency = make(map[string][]DebtNeighbor)
	}

	if index >= 0 {
		creditors[index].AmountMinor = balance
	} else {
		creditors = append(creditors, DebtNeighbor{ToUID: toUID, AmountMinor: balance})
	}

	graph.Adjacency[fromUID] = creditors
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
	updated.Adjacency = make(map[string][]DebtNeighbor, len(graph.Adjacency))
	for uid, creditors := range graph.Adjacency {
		updated.Adjacency[uid] = slices.Clone(creditors)
	}

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

	graph.Adjacency = updated.Adjacency
	return nil
}
