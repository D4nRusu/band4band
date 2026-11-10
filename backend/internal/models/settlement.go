package models

import "time"

type Settlement struct {
	ID          string    `json:"id" firestore:"-"`
	GroupID     string    `json:"groupID" firestore:"-"`
	FromUID     string    `json:"fromUID" firestore:"fromUID"`
	FromName    string    `json:"fromName" firestore:"fromName"`
	ToUID       string    `json:"toUID" firestore:"toUID"`
	ToName      string    `json:"toName" firestore:"toName"`
	AmountMinor int64     `json:"amountMinor" firestore:"amountMinor"`
	Currency    string    `json:"currency" firestore:"currency"`
	CreatedBy   string    `json:"createdBy" firestore:"createdBy"`
	CreatedAt   time.Time `json:"createdAt" firestore:"createdAt"`
}

// AsExpense expresses a payment as an expense the payer covered entirely for the recipient.
func (settlement Settlement) AsExpense() Expense {
	return Expense{
		PaidBy:      settlement.FromUID,
		AmountMinor: settlement.AmountMinor,
		Currency:    settlement.Currency,
		Contributions: []ExpenseContribution{
			{UID: settlement.ToUID, AmountMinor: settlement.AmountMinor},
			{UID: settlement.FromUID, AmountMinor: 0},
		},
	}
}
