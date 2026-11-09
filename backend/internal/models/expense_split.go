package models

import "slices"

type ExpenseContribution struct {
	UID         string `json:"uid" firestore:"uid"`
	Name        string `json:"name" firestore:"name"`
	AmountMinor int64  `json:"amountMinor" firestore:"amountMinor"`
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
