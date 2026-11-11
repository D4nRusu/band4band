package models

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"
)

const DefaultCurrency = "EUR"

// JSON numbers must remain exact in the mobile JavaScript runtime.
const MaxAmountMinor int64 = 9007199254740991

var (
	ErrInvalidExpenseParticipants = errors.New("select at least one contributor, without duplicates; all contributors must be current group members")
	ErrInvalidExpenseID           = errors.New("invalid expense ID")
	ErrInvalidExpenseName         = errors.New("expense name must contain 1–100 characters")
	ErrInvalidExpenseAmount       = errors.New("amount must be a positive integer in cents within the supported range")
	ErrInvalidExpenseCategory     = errors.New("choose an existing category from this group")
	ErrInvalidExpensePayer        = errors.New("the payer must be a current group member")
	ErrUnsupportedCurrency        = errors.New("this group currency is not supported yet")
	ErrInvalidExpenseSplit        = errors.New("choose an equal or unequal split and provide only that mode's contributor fields")
	ErrInvalidExpenseShares       = errors.New("enter an amount of zero or more for every contributor; the amounts must add up exactly to the expense total")
)

type ExpenseInput struct {
	SplitType       string                     `json:"splitType"`
	Contributions   []ExpenseContributionInput `json:"contributions"`
	ParticipantUIDs []string                   `json:"participantUIDs"`
	Name            string                     `json:"name"`
	CategoryID      string                     `json:"categoryID"`
	PaidBy          string                     `json:"paidBy"`
	AmountMinor     int64                      `json:"amountMinor"`
}

type Expense struct {
	ViewerPosition ExpensePosition       `json:"viewerPosition" firestore:"-"`
	SplitType      string                `json:"splitType,omitempty" firestore:"splitType,omitempty"`
	Contributions  []ExpenseContribution `json:"contributions,omitempty" firestore:"contributions,omitempty"`
	DeletedAt      time.Time             `json:"-" firestore:"deletedAt,omitempty"`
	DeletedBy      string                `json:"-" firestore:"deletedBy,omitempty"`
	ID             string                `json:"id" firestore:"-"`
	GroupID        string                `json:"groupID" firestore:"-"`
	Name           string                `json:"name" firestore:"name"`
	CategoryID     string                `json:"categoryID" firestore:"categoryID"`
	CategoryName   string                `json:"categoryName" firestore:"categoryName"`
	CategoryIcon   string                `json:"categoryIcon" firestore:"categoryIcon"`
	PaidBy         string                `json:"paidBy" firestore:"paidBy"`
	PayerName      string                `json:"payerName" firestore:"payerName"`
	AmountMinor    int64                 `json:"amountMinor" firestore:"amountMinor"`
	Currency       string                `json:"currency" firestore:"currency"`
	CreatedBy      string                `json:"createdBy" firestore:"createdBy"`
	CreatedAt      time.Time             `json:"createdAt" firestore:"createdAt"`
}

func (input ExpenseInput) Normalize() (ExpenseInput, error) {
	input.Name = strings.TrimSpace(input.Name)
	if length := utf8.RuneCountInString(input.Name); length < 1 || length > 100 {
		return ExpenseInput{}, ErrInvalidExpenseName
	}

	if input.AmountMinor < 1 || input.AmountMinor > MaxAmountMinor {
		return ExpenseInput{}, ErrInvalidExpenseAmount
	}

	if input.CategoryID == "" {
		return ExpenseInput{}, ErrInvalidExpenseCategory
	}

	if input.PaidBy == "" {
		return ExpenseInput{}, ErrInvalidExpensePayer
	}

	if input.SplitType == "" {
		input.SplitType = "equal"
	}

	if _, err := input.Allocations(); err != nil {
		return ExpenseInput{}, err
	}

	return input, nil
}
