package database

import (
	"context"
	"slices"
	"sort"

	"cloud.google.com/go/firestore"
	"github.com/D4nRusu/band4band/backend/internal/models"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *GroupStore) Expenses(ctx context.Context, uid, groupID string) ([]models.Expense, error) {
	if !validDocumentID(groupID) {
		return nil, models.ErrInvalidGroupID
	}

	var expenses []models.Expense
	err := s.db.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		if _, err := s.readMemberGroup(tx, uid, groupID); err != nil {
			return err
		}

		snapshots, err := tx.Documents(s.db.Collection("groups").Doc(groupID).Collection("expenses").Query).GetAll()
		if err != nil {
			return err
		}

		expenses = make([]models.Expense, 0, len(snapshots))
		for _, snapshot := range snapshots {
			var expense models.Expense
			if err := snapshot.DataTo(&expense); err != nil {
				return err
			}

			expense.ID = snapshot.Ref.ID
			expenses = append(expenses, expense)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(expenses, func(i, j int) bool {
		if expenses[i].CreatedAt.Equal(expenses[j].CreatedAt) {
			return expenses[i].ID < expenses[j].ID
		}

		return expenses[i].CreatedAt.After(expenses[j].CreatedAt)
	})

	return expenses, nil
}

func (s *GroupStore) CreateExpense(ctx context.Context, uid, groupID string, input models.ExpenseInput) (models.Expense, error) {
	if !validDocumentID(groupID) {
		return models.Expense{}, models.ErrInvalidGroupID
	}

	input, err := input.Normalize()
	if err != nil {
		return models.Expense{}, err
	}

	if !validDocumentID(input.PaidBy) {
		return models.Expense{}, models.ErrInvalidExpensePayer
	}

	groupRef := s.db.Collection("groups").Doc(groupID)
	expenseRef := groupRef.Collection("expenses").NewDoc()
	err = s.db.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		group, err := s.readMemberGroup(tx, uid, groupID)
		if err != nil {
			return err
		}

		if !slices.Contains(group.MemberUIDs, input.PaidBy) {
			return models.ErrInvalidExpensePayer
		}

		if group.Currency != models.DefaultCurrency {
			return models.ErrUnsupportedCurrency
		}

		category, ok := expenseCategory(group, input.CategoryID)
		if !ok {
			return models.ErrInvalidExpenseCategory
		}

		payer, err := tx.Get(s.db.Collection("users").Doc(input.PaidBy))
		if err != nil && status.Code(err) != codes.NotFound {
			return err
		}

		payerName := input.PaidBy
		if err == nil {
			var profile models.User
			if err := payer.DataTo(&profile); err != nil {
				return err
			}

			if profile.Name != "" {
				payerName = profile.Name
			} else if profile.Username != "" {
				payerName = profile.Username
			}
		}

		if err := tx.Create(expenseRef, map[string]any{
			"groupID":      groupID,
			"name":         input.Name,
			"categoryID":   category.ID,
			"categoryName": category.Name,
			"categoryIcon": category.Icon,
			"paidBy":       input.PaidBy,
			"payerName":    payerName,
			"amountMinor":  input.AmountMinor,
			"currency":     group.Currency,
			"createdBy":    uid,
			"createdAt":    firestore.ServerTimestamp,
		}); err != nil {
			return err
		}

		return tx.Update(groupRef, []firestore.Update{
			{Path: "currency", Value: group.Currency},
			{Path: "hasExpenses", Value: true},
		})
	})
	if err != nil {
		return models.Expense{}, err
	}

	// Resolve the server timestamp after committing. A failed read is an uncertain result,
	// so the client must refresh rather than automatically repeating the write.
	snapshot, err := expenseRef.Get(ctx)
	if err != nil {
		return models.Expense{}, err
	}

	var expense models.Expense
	if err := snapshot.DataTo(&expense); err != nil {
		return models.Expense{}, err
	}

	expense.ID = expenseRef.ID
	return expense, nil
}

func expenseCategory(group models.Group, id string) (models.Category, bool) {
	for _, category := range models.DefaultCategories() {
		if category.ID == id {
			return category, true
		}
	}

	category, ok := group.CustomCategories[id]
	return category, ok
}
