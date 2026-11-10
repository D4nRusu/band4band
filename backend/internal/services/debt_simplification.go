package services

import (
	"cmp"
	"maps"
	"slices"

	"github.com/D4nRusu/band4band/backend/internal/models"
)

// SimplifyDebts cancels directed cycles on a copy, preserving each user's net balance.
// The raw graph must remain unchanged so expenses can still be reversed individually.
func SimplifyDebts(graph models.DebtGraph) (models.DebtGraph, error) {
	if err := graph.Validate(); err != nil {
		return models.DebtGraph{}, err
	}

	simplified := graph
	simplified.Adjacency = make(map[string][]models.DebtNeighbor, len(graph.Adjacency))
	for uid, debts := range graph.Adjacency {
		neighbors := slices.Clone(debts)
		slices.SortFunc(neighbors, func(a, b models.DebtNeighbor) int {
			return cmp.Compare(a.ToUID, b.ToUID)
		})
		simplified.Adjacency[uid] = neighbors
	}

	for {
		cycle := findDebtCycle(simplified.Adjacency)
		if len(cycle) == 0 {
			return simplified, nil
		}

		minimum := cycle[0].amountMinor
		for _, edge := range cycle[1:] {
			minimum = min(minimum, edge.amountMinor)
		}

		for _, edge := range cycle {
			if err := simplified.AdjustEdge(edge.fromUID, edge.toUID, -minimum); err != nil {
				return models.DebtGraph{}, err
			}
		}
	}
}

type cycleDebt struct {
	fromUID     string
	toUID       string
	amountMinor int64
}

// A link to an active DFS ancestor closes a cycle; parent links recover its edges.
func findDebtCycle(adjacency map[string][]models.DebtNeighbor) []cycleDebt {
	const (
		visiting = 1
		visited  = 2
	)

	state := make(map[string]int)
	parents := make(map[string]cycleDebt)
	var visit func(string) []cycleDebt
	visit = func(uid string) []cycleDebt {
		state[uid] = visiting
		for _, debt := range adjacency[uid] {
			edge := cycleDebt{fromUID: uid, toUID: debt.ToUID, amountMinor: debt.AmountMinor}
			if state[debt.ToUID] == visiting {
				cycle := []cycleDebt{edge}
				for ancestor := uid; ancestor != debt.ToUID; {
					parent := parents[ancestor]
					cycle = append(cycle, parent)
					ancestor = parent.fromUID
				}

				return cycle
			}

			if state[debt.ToUID] == 0 {
				parents[debt.ToUID] = edge
				if cycle := visit(debt.ToUID); len(cycle) > 0 {
					return cycle
				}
			}
		}

		state[uid] = visited
		return nil
	}

	// Stable traversal makes overlapping-cycle results independent of map order.
	for _, uid := range slices.Sorted(maps.Keys(adjacency)) {
		if state[uid] == 0 {
			if cycle := visit(uid); len(cycle) > 0 {
				return cycle
			}
		}
	}

	return nil
}
