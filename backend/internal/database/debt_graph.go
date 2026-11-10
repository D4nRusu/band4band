package database

import (
	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/D4nRusu/band4band/backend/internal/models"
)

func (s *GroupStore) debtGraphRef(groupID string) *firestore.DocumentRef {
	return s.db.Collection("groups").Doc(groupID).Collection("debts").Doc("current")
}

// loadDebtGraph initializes only when absent. Existing graphs are never rebuilt here.
// All reads happen before the caller writes anything in the transaction.
func (s *GroupStore) loadDebtGraph(tx *firestore.Transaction, groupID, currency string) (models.DebtGraph, bool, error) {
	snapshot, err := tx.Get(s.debtGraphRef(groupID))
	if status.Code(err) == codes.NotFound {
		graph, err := s.initializeDebtGraph(tx, groupID, currency)
		return graph, true, err
	}

	if err != nil {
		return models.DebtGraph{}, false, err
	}

	var graph models.DebtGraph
	if err := snapshot.DataTo(&graph); err != nil {
		return models.DebtGraph{}, false, err
	}

	if graph.Currency != currency {
		return models.DebtGraph{}, false, models.ErrInvalidDebtGraph
	}

	if err := graph.Validate(); err != nil {
		return models.DebtGraph{}, false, err
	}

	return graph, false, nil
}

// initializeDebtGraph replays existing active allocations without rewriting expenses.
// New expense creation calls this before creating its own document, avoiding double counting.
func (s *GroupStore) initializeDebtGraph(tx *firestore.Transaction, groupID, currency string) (models.DebtGraph, error) {
	graph := models.DebtGraph{Currency: currency, Edges: []models.DebtEdge{}}
	snapshots, err := tx.Documents(s.db.Collection("groups").Doc(groupID).Collection("expenses").Query).GetAll()
	if err != nil {
		return models.DebtGraph{}, err
	}

	for _, snapshot := range snapshots {
		var expense models.Expense
		if err := snapshot.DataTo(&expense); err != nil {
			return models.DebtGraph{}, err
		}

		if err := graph.ApplyExpense(expense, false); err != nil {
			return models.DebtGraph{}, err
		}
	}

	return graph, nil
}

func (s *GroupStore) writeDebtGraph(tx *firestore.Transaction, groupID string, graph models.DebtGraph) error {
	if graph.Edges == nil {
		graph.Edges = []models.DebtEdge{}
	}

	return tx.Set(s.debtGraphRef(groupID), map[string]any{
		"currency":  graph.Currency,
		"edges":     graph.Edges,
		"updatedAt": firestore.ServerTimestamp,
	})
}
