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
		simplified.Adjacency[uid] = slices.Clone(debts)
	}
	sortAdjacency(simplified.Adjacency)

	for {
		cycle := findDebtCycle(simplified.Adjacency)
		if len(cycle) > 0 {
			minimum := cycle[0].amountMinor
			for _, edge := range cycle[1:] {
				minimum = min(minimum, edge.amountMinor)
			}

			for _, edge := range cycle {
				if err := simplified.AdjustEdge(edge.fromUID, edge.toUID, -minimum); err != nil {
					return models.DebtGraph{}, err
				}
			}

			sortAdjacency(simplified.Adjacency)
			continue
		}

		chain, ok := findChainTransfer(simplified.Adjacency)
		if !ok {
			return simplified, nil
		}

		if err := simplified.AdjustEdge(chain.fromUID, chain.viaUID, -chain.amountMinor); err != nil {
			return models.DebtGraph{}, err
		}

		if err := simplified.AdjustEdge(chain.viaUID, chain.toUID, -chain.amountMinor); err != nil {
			return models.DebtGraph{}, err
		}

		if err := simplified.AdjustEdge(chain.fromUID, chain.toUID, chain.amountMinor); err != nil {
			return models.DebtGraph{}, err
		}

		sortAdjacency(simplified.Adjacency)
	}
}

type cycleDebt struct {
	fromUID     string
	toUID       string
	amountMinor int64
}

type chainDebt struct {
	fromUID     string
	viaUID      string
	toUID       string
	amountMinor int64
}

type incomingDebt struct {
	fromUID     string
	amountMinor int64
}

func sortAdjacency(adjacency map[string][]models.DebtNeighbor) {
	for uid := range adjacency {
		slices.SortFunc(adjacency[uid], func(a, b models.DebtNeighbor) int {
			return cmp.Compare(a.ToUID, b.ToUID)
		})
	}
}

// If A owes B and B owes C, reroute min(A->B, B->C) directly from A to C.
func findChainTransfer(adjacency map[string][]models.DebtNeighbor) (chainDebt, bool) {
	incoming := make(map[string][]incomingDebt, len(adjacency))
	for _, fromUID := range slices.Sorted(maps.Keys(adjacency)) {
		for _, debt := range adjacency[fromUID] {
			incoming[debt.ToUID] = append(incoming[debt.ToUID], incomingDebt{fromUID: fromUID, amountMinor: debt.AmountMinor})
		}
	}

	for _, viaUID := range slices.Sorted(maps.Keys(adjacency)) {
		inEdges := incoming[viaUID]
		if len(inEdges) == 0 || len(adjacency[viaUID]) == 0 {
			continue
		}

		slices.SortFunc(inEdges, func(a, b incomingDebt) int {
			return cmp.Compare(a.fromUID, b.fromUID)
		})

		for _, inEdge := range inEdges {
			for _, outEdge := range adjacency[viaUID] {
				if inEdge.fromUID == outEdge.ToUID {
					continue
				}

				return chainDebt{
					fromUID:     inEdge.fromUID,
					viaUID:      viaUID,
					toUID:       outEdge.ToUID,
					amountMinor: min(inEdge.amountMinor, outEdge.AmountMinor),
				}, true
			}
		}
	}

	return chainDebt{}, false
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
