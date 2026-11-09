package models

type ExpensePosition struct {
	Kind        string `json:"kind"`
	AmountMinor int64  `json:"amountMinor"`
}

// PositionFor describes only this expense, before settlements or cross-expense netting.
func (expense Expense) PositionFor(uid string) ExpensePosition {
	if len(expense.Contributions) == 0 {
		return ExpensePosition{Kind: "unallocated"}
	}

	var ownShare int64
	participates := false
	for _, contribution := range expense.Contributions {
		if contribution.UID == uid {
			ownShare = contribution.AmountMinor
			participates = true
			break
		}
	}

	if expense.PaidBy == uid {
		receivable := expense.AmountMinor - ownShare
		if receivable > 0 {
			return ExpensePosition{Kind: "lent", AmountMinor: receivable}
		}

		return ExpensePosition{Kind: "none"}
	}

	if participates {
		if ownShare > 0 {
			return ExpensePosition{Kind: "borrowed", AmountMinor: ownShare}
		}

		return ExpensePosition{Kind: "none"}
	}

	return ExpensePosition{Kind: "not_involved"}
}
