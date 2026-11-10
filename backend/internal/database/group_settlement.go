package database

import (
	"context"
	"slices"
	"sort"

	"cloud.google.com/go/firestore"
	"github.com/D4nRusu/band4band/backend/internal/models"
	"github.com/D4nRusu/band4band/backend/internal/services"
)

func (s *GroupStore) SettleUp(ctx context.Context, uid, groupID, fromUID, toUID string, amountMinor int64) error {
	if !validDocumentID(groupID) {
		return models.ErrInvalidGroupID
	}

	if !validDocumentID(fromUID) || !validDocumentID(toUID) || fromUID == toUID {
		return models.ErrInvalidDebtUser
	}

	if uid != fromUID && uid != toUID {
		return models.ErrSettlementNotParty
	}

	if amountMinor < 1 || amountMinor > models.MaxAmountMinor {
		return models.ErrInvalidDebtAmount
	}

	return s.db.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		group, err := s.readMemberGroup(tx, uid, groupID)
		if err != nil {
			return err
		}

		if !slices.Contains(group.MemberUIDs, fromUID) || !slices.Contains(group.MemberUIDs, toUID) {
			return models.ErrInvalidDebtUser
		}

		graph, _, err := s.loadDebtGraph(tx, groupID, group.Currency)
		if err != nil {
			return err
		}

		// A payment is recorded as a new opposite debt rather than reducing existing edges,
		// so expenses stay individually reversible and simplification nets the balances.
		err = graph.ApplyExpense(models.Expense{
			PaidBy:      fromUID,
			AmountMinor: amountMinor,
			Currency:    group.Currency,
			Contributions: []models.ExpenseContribution{
				{UID: toUID, AmountMinor: amountMinor},
				{UID: fromUID, AmountMinor: 0},
			},
		}, false)
		if err != nil {
			return err
		}

		return s.writeDebtGraph(tx, groupID, graph)
	})
}

func (s *GroupStore) UserOwedSummary(ctx context.Context, uid, groupID, userUID string) (models.UserOwedSummary, error) {
	if !validDocumentID(groupID) {
		return models.UserOwedSummary{}, models.ErrInvalidGroupID
	}

	if !validDocumentID(userUID) {
		return models.UserOwedSummary{}, models.ErrInvalidDebtUser
	}

	var summary models.UserOwedSummary
	err := s.db.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		group, err := s.readMemberGroup(tx, uid, groupID)
		if err != nil {
			return err
		}

		if !slices.Contains(group.MemberUIDs, userUID) {
			return models.ErrInvalidDebtUser
		}

		graph, _, err := s.loadDebtGraph(tx, groupID, group.Currency)
		if err != nil {
			return err
		}

		simplified, err := services.SimplifyDebts(graph)
		if err != nil {
			return err
		}

		summary = models.UserOwedSummary{
			UserUID:  userUID,
			Currency: simplified.Currency,
			Debts:    make([]models.UserOwedDebt, 0, len(simplified.Adjacency[userUID])),
		}

		for _, debt := range simplified.Adjacency[userUID] {
			summary.TotalAmountMinor += debt.AmountMinor
			summary.Debts = append(summary.Debts, models.UserOwedDebt{
				ToUID:       debt.ToUID,
				AmountMinor: debt.AmountMinor,
			})
		}

		return nil
	})
	if err != nil {
		return models.UserOwedSummary{}, err
	}

	sort.Slice(summary.Debts, func(i, j int) bool {
		if summary.Debts[i].AmountMinor == summary.Debts[j].AmountMinor {
			return summary.Debts[i].ToUID < summary.Debts[j].ToUID
		}

		return summary.Debts[i].AmountMinor > summary.Debts[j].AmountMinor
	})

	return summary, nil
}

func (s *GroupStore) UserReceivableSummary(ctx context.Context, uid, groupID, userUID string) (models.UserReceivableSummary, error) {
	if !validDocumentID(groupID) {
		return models.UserReceivableSummary{}, models.ErrInvalidGroupID
	}

	if !validDocumentID(userUID) {
		return models.UserReceivableSummary{}, models.ErrInvalidDebtUser
	}

	var summary models.UserReceivableSummary
	err := s.db.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		group, err := s.readMemberGroup(tx, uid, groupID)
		if err != nil {
			return err
		}

		if !slices.Contains(group.MemberUIDs, userUID) {
			return models.ErrInvalidDebtUser
		}

		graph, _, err := s.loadDebtGraph(tx, groupID, group.Currency)
		if err != nil {
			return err
		}

		simplified, err := services.SimplifyDebts(graph)
		if err != nil {
			return err
		}

		summary = models.UserReceivableSummary{
			UserUID:  userUID,
			Currency: simplified.Currency,
			Debts:    []models.UserReceivableDebt{},
		}

		for fromUID, creditors := range simplified.Adjacency {
			if fromUID == userUID {
				continue
			}

			for _, debt := range creditors {
				if debt.ToUID != userUID {
					continue
				}

				summary.TotalAmountMinor += debt.AmountMinor
				summary.Debts = append(summary.Debts, models.UserReceivableDebt{
					FromUID:     fromUID,
					AmountMinor: debt.AmountMinor,
				})
			}
		}

		return nil
	})
	if err != nil {
		return models.UserReceivableSummary{}, err
	}

	sort.Slice(summary.Debts, func(i, j int) bool {
		if summary.Debts[i].AmountMinor == summary.Debts[j].AmountMinor {
			return summary.Debts[i].FromUID < summary.Debts[j].FromUID
		}

		return summary.Debts[i].AmountMinor > summary.Debts[j].AmountMinor
	})

	return summary, nil
}
