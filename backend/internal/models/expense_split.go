package models

import (
	"cmp"
	"slices"
)

type ExpenseContributionInput struct {
	UID         string `json:"uid"`
	AmountMinor int64  `json:"amountMinor"`
}

type ExpenseContribution struct {
	UID         string `json:"uid" firestore:"uid"`
	Name        string `json:"name" firestore:"name"`
	AmountMinor int64  `json:"amountMinor" firestore:"amountMinor"`
}

// Allocations validates the selected split mode and produces the shares to save.
// Names are resolved from stored profiles, never accepted from the client.
func (input ExpenseInput) Allocations() ([]ExpenseContribution, error) {
	switch input.SplitType {
	case "", "equal":
		if len(input.Contributions) != 0 {
			return nil, ErrInvalidExpenseSplit
		}

		return EqualContributions(input.AmountMinor, input.ParticipantUIDs)
	case "unequal":
		if len(input.ParticipantUIDs) != 0 {
			return nil, ErrInvalidExpenseSplit
		}

		return UnequalContributions(input.AmountMinor, input.Contributions)
	default:
		return nil, ErrInvalidExpenseSplit
	}
}

func UnequalContributions(amountMinor int64, shares []ExpenseContributionInput) ([]ExpenseContribution, error) {
	if amountMinor < 1 || amountMinor > MaxAmountMinor {
		return nil, ErrInvalidExpenseAmount
	}

	if len(shares) == 0 {
		return nil, ErrInvalidExpenseParticipants
	}

	seen := make(map[string]bool, len(shares))
	remaining := amountMinor
	contributions := make([]ExpenseContribution, 0, len(shares))
	for _, share := range shares {
		if share.UID == "" || seen[share.UID] {
			return nil, ErrInvalidExpenseParticipants
		}

		if share.AmountMinor < 0 || share.AmountMinor > remaining {
			return nil, ErrInvalidExpenseShares
		}

		seen[share.UID] = true
		remaining -= share.AmountMinor
		contributions = append(contributions, ExpenseContribution{UID: share.UID, AmountMinor: share.AmountMinor})
	}

	if remaining != 0 {
		return nil, ErrInvalidExpenseShares
	}

	slices.SortFunc(contributions, func(a, b ExpenseContribution) int {
		return cmp.Compare(a.UID, b.UID)
	})
	return contributions, nil
}

// EqualContributions uses UID order so reordering a selection cannot change rounding.
func EqualContributions(amountMinor int64, participantUIDs []string) ([]ExpenseContribution, error) {
	if amountMinor < 1 || amountMinor > MaxAmountMinor {
		return nil, ErrInvalidExpenseAmount
	}

	if len(participantUIDs) == 0 {
		return nil, ErrInvalidExpenseParticipants
	}

	uids := slices.Clone(participantUIDs)
	slices.Sort(uids)
	for i, uid := range uids {
		if uid == "" || (i > 0 && uid == uids[i-1]) {
			return nil, ErrInvalidExpenseParticipants
		}
	}

	base := amountMinor / int64(len(uids))
	remainder := amountMinor % int64(len(uids))
	contributions := make([]ExpenseContribution, 0, len(uids))
	for i, uid := range uids {
		share := base
		if int64(i) < remainder {
			share++
		}

		contributions = append(contributions, ExpenseContribution{UID: uid, AmountMinor: share})
	}

	return contributions, nil
}
