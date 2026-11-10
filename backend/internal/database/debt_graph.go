package database

import (
	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"time"

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

	graph, converted, err := decodeDebtGraph(snapshot)
	if err != nil {
		return models.DebtGraph{}, false, err
	}

	if graph.Currency != currency {
		return models.DebtGraph{}, false, models.ErrInvalidDebtGraph
	}

	if err := graph.Validate(); err != nil {
		return models.DebtGraph{}, false, err
	}

	return graph, converted, nil
}

// decodeDebtGraph preserves stored debts when converting earlier representations.
func decodeDebtGraph(snapshot *firestore.DocumentSnapshot) (models.DebtGraph, bool, error) {
	var metadata struct {
		Currency  string    `firestore:"currency"`
		UpdatedAt time.Time `firestore:"updatedAt"`
	}

	if err := snapshot.DataTo(&metadata); err != nil {
		return models.DebtGraph{}, false, err
	}

	graph := models.DebtGraph{Currency: metadata.Currency, UpdatedAt: metadata.UpdatedAt,
		Adjacency: make(map[string][]models.DebtNeighbor)}
	data := snapshot.Data()
	if raw, exists := data["adjacency"]; exists {
		if neighbors, ok := raw.(map[string]any); ok {
			for _, value := range neighbors {
				if _, isMap := value.(map[string]any); isMap {
					return convertDebtMap(snapshot, graph)
				}
			}
		}

		err := snapshot.DataTo(&graph)
		return graph, false, err
	}

	if _, exists := data["edges"]; !exists {
		return models.DebtGraph{}, false, models.ErrInvalidDebtGraph
	}

	var previous struct {
		Edges []struct {
			FromUID     string `firestore:"fromUID"`
			ToUID       string `firestore:"toUID"`
			AmountMinor int64  `firestore:"amountMinor"`
		} `firestore:"edges"`
	}

	if err := snapshot.DataTo(&previous); err != nil {
		return models.DebtGraph{}, false, err
	}

	for _, edge := range previous.Edges {
		graph.Adjacency[edge.FromUID] = append(graph.Adjacency[edge.FromUID], models.DebtNeighbor{
			ToUID: edge.ToUID, AmountMinor: edge.AmountMinor,
		})
	}

	return graph, true, nil
}

func convertDebtMap(snapshot *firestore.DocumentSnapshot, graph models.DebtGraph) (models.DebtGraph, bool, error) {
	var previous struct {
		Adjacency map[string]map[string]int64 `firestore:"adjacency"`
	}

	if err := snapshot.DataTo(&previous); err != nil {
		return models.DebtGraph{}, false, err
	}

	for uid, creditors := range previous.Adjacency {
		graph.Adjacency[uid] = make([]models.DebtNeighbor, 0, len(creditors))
		for toUID, amount := range creditors {
			graph.Adjacency[uid] = append(graph.Adjacency[uid], models.DebtNeighbor{ToUID: toUID, AmountMinor: amount})
		}
	}

	return graph, true, nil
}

// initializeDebtGraph replays existing active allocations without rewriting expenses.
// New expense creation calls this before creating its own document, avoiding double counting.
func (s *GroupStore) initializeDebtGraph(tx *firestore.Transaction, groupID, currency string) (models.DebtGraph, error) {
	graph := models.DebtGraph{Currency: currency, Adjacency: map[string][]models.DebtNeighbor{}}
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
	if graph.Adjacency == nil {
		graph.Adjacency = map[string][]models.DebtNeighbor{}
	}

	return tx.Set(s.debtGraphRef(groupID), map[string]any{
		"currency":  graph.Currency,
		"adjacency": graph.Adjacency,
		"updatedAt": firestore.ServerTimestamp,
	})
}
