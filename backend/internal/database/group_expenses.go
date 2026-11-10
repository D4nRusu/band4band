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
		group, err := s.readMemberGroup(tx, uid, groupID)
		if err != nil {
			return err
		}

		snapshots, err := tx.Documents(s.db.Collection("groups").Doc(groupID).Collection("expenses").Query).GetAll()
		if err != nil {
			return err
		}

		graph, needsWrite, err := s.loadDebtGraph(tx, groupID, group.Currency)
		if err != nil {
			return err
		}

		expenses = make([]models.Expense, 0, len(snapshots))
		for _, snapshot := range snapshots {
			var expense models.Expense
			if err := snapshot.DataTo(&expense); err != nil {
				return err
			}

			if !expense.DeletedAt.IsZero() {
				continue
			}

			expense.ID = snapshot.Ref.ID
			expense.GroupID = groupID
			expenses = append(expenses, expense)
		}

		if needsWrite && (len(snapshots) > 0 || len(graph.Adjacency) > 0) {
			return s.writeDebtGraph(tx, groupID, graph)
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

		payerName, contributions, err := s.expensePeople(tx, group, input)
		if err != nil {
			return err
		}

		graph, _, err := s.loadDebtGraph(tx, groupID, group.Currency)
		if err != nil {
			return err
		}

		if err := graph.ApplyExpense(models.Expense{
			PaidBy: input.PaidBy, AmountMinor: input.AmountMinor,
			Currency: group.Currency, Contributions: contributions,
		}, false); err != nil {
			return err
		}

		if err := tx.Create(expenseRef, map[string]any{
			"splitType":     "equal",
			"contributions": contributions,
			"name":          input.Name,
			"categoryID":    category.ID,
			"categoryName":  category.Name,
			"categoryIcon":  category.Icon,
			"paidBy":        input.PaidBy,
			"payerName":     payerName,
			"amountMinor":   input.AmountMinor,
			"currency":      group.Currency,
			"createdBy":     uid,
			"createdAt":     firestore.ServerTimestamp,
		}); err != nil {
			return err
		}

		if err := s.writeDebtGraph(tx, groupID, graph); err != nil {
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
	expense.GroupID = groupID
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

func (s *GroupStore) DeleteExpense(ctx context.Context, uid, groupID, expenseID string) error {
	if !validDocumentID(groupID) {
		return models.ErrInvalidGroupID
	}

	if !validDocumentID(expenseID) {
		return models.ErrInvalidExpenseID
	}

	return s.db.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		group, err := s.readMemberGroup(tx, uid, groupID)
		if err != nil {
			return err
		}

		ref := s.db.Collection("groups").Doc(groupID).Collection("expenses").Doc(expenseID)
		snapshot, err := tx.Get(ref)
		if status.Code(err) == codes.NotFound {
			return nil
		}

		if err != nil {
			return err
		}

		var expense models.Expense
		if err := snapshot.DataTo(&expense); err != nil {
			return err
		}

		if !expense.DeletedAt.IsZero() {
			return nil
		}

		graph, _, err := s.loadDebtGraph(tx, groupID, group.Currency)
		if err != nil {
			return err
		}

		if err := graph.ApplyExpense(expense, true); err != nil {
			return err
		}

		if err := s.writeDebtGraph(tx, groupID, graph); err != nil {
			return err
		}

		return tx.Update(ref, []firestore.Update{
			{Path: "deletedAt", Value: firestore.ServerTimestamp},
			{Path: "deletedBy", Value: uid},
		})
	})
}
