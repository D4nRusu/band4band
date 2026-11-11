package models

import (
	"cmp"
	"fmt"
	"slices"
)

type ReceiptUnitAssignment struct {
	UID   string `json:"uid"`
	Units int64  `json:"units"`
}

type ReceiptExpenseItem struct {
	AmountMinor int64                   `json:"amountMinor"`
	Units       int64                   `json:"units"`
	Assignments []ReceiptUnitAssignment `json:"assignments"`
}

type ReceiptExpenseInput struct {
	Name            string               `json:"name"`
	CategoryID      string               `json:"categoryID"`
	PaidBy          string               `json:"paidBy"`
	Currency        string               `json:"currency"`
	ConfirmCurrency bool                 `json:"confirmCurrency"`
	TotalMinor      int64                `json:"totalMinor"`
	Items           []ReceiptExpenseItem `json:"items"`
}

// Expense combines reviewed line assignments into the existing unequal split.
func (input ReceiptExpenseInput) Expense(groupCurrency string) (ExpenseInput, error) {
	if input.Currency == "" || (input.Currency != groupCurrency && !input.ConfirmCurrency) {
		return ExpenseInput{}, fmt.Errorf("confirm using the receipt amounts in the group currency")
	}

	if len(input.Items) == 0 || len(input.Items) > 200 {
		return ExpenseInput{}, fmt.Errorf("assign between 1 and 200 receipt items")
	}

	total := int64(0)
	shares := make(map[string]int64)
	for i, item := range input.Items {
		lineShares, err := item.contributions()
		if err != nil {
			return ExpenseInput{}, fmt.Errorf("item %d: %w", i+1, err)
		}

		total += item.AmountMinor
		if total < -MaxAmountMinor || total > MaxAmountMinor {
			return ExpenseInput{}, ErrInvalidExpenseAmount
		}

		for _, share := range lineShares {
			shares[share.UID] += share.AmountMinor
			if shares[share.UID] < -MaxAmountMinor || shares[share.UID] > MaxAmountMinor {
				return ExpenseInput{}, ErrInvalidExpenseShares
			}
		}
	}

	if total != input.TotalMinor {
		return ExpenseInput{}, fmt.Errorf("the reviewed item prices must add up exactly to the receipt total")
	}

	expense := ExpenseInput{
		Name: input.Name, CategoryID: input.CategoryID, PaidBy: input.PaidBy,
		AmountMinor: total, SplitType: "unequal",
	}
	for uid, amount := range shares {
		expense.Contributions = append(expense.Contributions, ExpenseContributionInput{UID: uid, AmountMinor: amount})
	}
	slices.SortFunc(expense.Contributions, func(a, b ExpenseContributionInput) int { return cmp.Compare(a.UID, b.UID) })

	return expense.Normalize()
}

func (item ReceiptExpenseItem) contributions() ([]ExpenseContributionInput, error) {
	if item.AmountMinor < -MaxAmountMinor || item.AmountMinor > MaxAmountMinor {
		return nil, ErrInvalidExpenseAmount
	}

	if item.Units < 1 || item.Units > MaxAmountMinor || len(item.Assignments) == 0 {
		return nil, fmt.Errorf("assign every unit to a group member")
	}

	assignments := slices.Clone(item.Assignments)
	slices.SortFunc(assignments, func(a, b ReceiptUnitAssignment) int { return cmp.Compare(a.UID, b.UID) })
	remaining := item.Units
	for i, assignment := range assignments {
		if assignment.UID == "" || (i > 0 && assignment.UID == assignments[i-1].UID) {
			return nil, fmt.Errorf("choose a different group member for each assignment")
		}

		if assignment.Units < 1 || assignment.Units > remaining {
			return nil, fmt.Errorf("assigned units must match the item quantity")
		}

		remaining -= assignment.Units
	}

	if remaining != 0 {
		return nil, fmt.Errorf("assign all %d units", item.Units)
	}

	amount := item.AmountMinor
	if amount < 0 {
		amount = -amount
	}

	base, extra := amount/item.Units, amount%item.Units
	shares := make([]ExpenseContributionInput, 0, len(assignments))
	for _, assignment := range assignments {
		bonus := min(extra, assignment.Units)
		share := base*assignment.Units + bonus
		extra -= bonus
		if item.AmountMinor < 0 {
			share = -share
		}

		shares = append(shares, ExpenseContributionInput{UID: assignment.UID, AmountMinor: share})
	}

	return shares, nil
}
