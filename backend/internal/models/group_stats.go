package models

import (
	"math"
	"sort"
)

type CategoryTotal struct {
	CategoryID   string `json:"categoryID"`
	CategoryName string `json:"categoryName"`
	CategoryIcon string `json:"categoryIcon"`
	AmountMinor  int64  `json:"amountMinor"`
}

type MemberShareTotal struct {
	UID         string `json:"uid"`
	Name        string `json:"name"`
	AmountMinor int64  `json:"amountMinor"`
}

type GroupStats struct {
	TotalAmountMinor         int64              `json:"totalAmountMinor"`
	ByCategory               []CategoryTotal    `json:"byCategory"`
	ByMember                 []MemberShareTotal `json:"byMember"`
	PersonalTotalAmountMinor int64              `json:"personalTotalAmountMinor"`
	PersonalByCategory       []CategoryTotal    `json:"personalByCategory"`
}

// BuildGroupStats totals active expenses. Member and personal figures use each person's
// share of an expense, so expenses without recorded contributions only count in group totals.
func BuildGroupStats(expenses []Expense, viewerUID string) (GroupStats, error) {
	var stats GroupStats
	groupCategories := map[string]*CategoryTotal{}
	personalCategories := map[string]*CategoryTotal{}
	members := map[string]*MemberShareTotal{}

	for _, expense := range expenses {
		if !expense.DeletedAt.IsZero() {
			continue
		}

		if err := addAmount(&stats.TotalAmountMinor, expense.AmountMinor); err != nil {
			return GroupStats{}, err
		}

		if err := addAmount(&categoryTotal(groupCategories, expense).AmountMinor, expense.AmountMinor); err != nil {
			return GroupStats{}, err
		}

		for _, share := range expense.Contributions {
			if share.AmountMinor <= 0 {
				continue
			}

			member, ok := members[share.UID]
			if !ok {
				member = &MemberShareTotal{UID: share.UID, Name: share.Name}
				members[share.UID] = member
			}

			if err := addAmount(&member.AmountMinor, share.AmountMinor); err != nil {
				return GroupStats{}, err
			}

			if share.UID != viewerUID {
				continue
			}

			if err := addAmount(&stats.PersonalTotalAmountMinor, share.AmountMinor); err != nil {
				return GroupStats{}, err
			}

			if err := addAmount(&categoryTotal(personalCategories, expense).AmountMinor, share.AmountMinor); err != nil {
				return GroupStats{}, err
			}
		}
	}

	stats.ByCategory = sortedCategories(groupCategories)
	stats.PersonalByCategory = sortedCategories(personalCategories)

	stats.ByMember = make([]MemberShareTotal, 0, len(members))
	for _, member := range members {
		stats.ByMember = append(stats.ByMember, *member)
	}

	sort.Slice(stats.ByMember, func(i, j int) bool {
		if stats.ByMember[i].AmountMinor == stats.ByMember[j].AmountMinor {
			return stats.ByMember[i].UID < stats.ByMember[j].UID
		}

		return stats.ByMember[i].AmountMinor > stats.ByMember[j].AmountMinor
	})

	return stats, nil
}

func categoryTotal(totals map[string]*CategoryTotal, expense Expense) *CategoryTotal {
	total, ok := totals[expense.CategoryID]
	if !ok {
		total = &CategoryTotal{CategoryID: expense.CategoryID, CategoryName: expense.CategoryName, CategoryIcon: expense.CategoryIcon}
		totals[expense.CategoryID] = total
	}

	return total
}

func sortedCategories(totals map[string]*CategoryTotal) []CategoryTotal {
	sorted := make([]CategoryTotal, 0, len(totals))
	for _, total := range totals {
		sorted = append(sorted, *total)
	}

	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].AmountMinor == sorted[j].AmountMinor {
			return sorted[i].CategoryID < sorted[j].CategoryID
		}

		return sorted[i].AmountMinor > sorted[j].AmountMinor
	})

	return sorted
}

func addAmount(total *int64, amount int64) error {
	if amount > math.MaxInt64-*total {
		return ErrDebtOverflow
	}

	*total += amount
	return nil
}
