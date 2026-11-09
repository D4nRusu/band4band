package models_test

import (
	"slices"
	"testing"

	"github.com/D4nRusu/band4band/backend/internal/models"
)

func TestEqualContributions(t *testing.T) {
	for _, tc := range []struct {
		name         string
		amount       int64
		participants []string
		expected     []int64
	}{
		{"all three", 1500, []string{"usr3", "usr1", "usr2"}, []int64{500, 500, 500}},
		{"payer excluded", 5000, []string{"usr3", "usr2"}, []int64{2500, 2500}},
		{"one contributor", 5000, []string{"usr2"}, []int64{5000}},
		{"one remainder cent", 1000, []string{"usr3", "usr2", "usr1"}, []int64{334, 333, 333}},
		{"two remainder cents", 1001, []string{"usr3", "usr2", "usr1"}, []int64{334, 334, 333}},
		{"less than one cent per person", 1, []string{"usr3", "usr2", "usr1"}, []int64{1, 0, 0}},
		{"largest exact JSON amount", models.MaxAmountMinor, []string{"usr2", "usr1"}, []int64{4503599627370496, 4503599627370495}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			original := slices.Clone(tc.participants)
			shares, err := models.EqualContributions(tc.amount, tc.participants)
			if err != nil {
				t.Fatal(err)
			}

			var total int64
			for i, share := range shares {
				if share.AmountMinor != tc.expected[i] {
					t.Fatalf("shares=%+v", shares)
				}

				if i > 0 && shares[i-1].UID >= share.UID {
					t.Fatal("UIDs not ordered")
				}

				total += share.AmountMinor
			}

			if total != tc.amount {
				t.Fatalf("sum=%d want=%d", total, tc.amount)
			}

			if !slices.Equal(original, tc.participants) {
				t.Fatal("input mutated")
			}
		})
	}
}

func TestEqualContributionsRejectsInvalidSelections(t *testing.T) {
	for _, uids := range [][]string{nil, {}, {""}, {"usr1", "usr1"}} {
		if _, err := models.EqualContributions(100, uids); err == nil {
			t.Fatalf("accepted %v", uids)
		}
	}
}

func FuzzEqualContributionsPreservesTotal(f *testing.F) {
	f.Add(int64(1000), uint8(3))
	f.Add(models.MaxAmountMinor, uint8(2))
	f.Fuzz(func(t *testing.T, amount int64, count uint8) {
		if amount < 1 || amount > models.MaxAmountMinor || count == 0 {
			return
		}

		uids := make([]string, int(count))
		for i := range uids {
			uids[i] = string(rune('a' + i))
		}

		shares, err := models.EqualContributions(amount, uids)
		if err != nil {
			t.Fatal(err)
		}

		var total int64
		for _, share := range shares {
			total += share.AmountMinor
			if share.AmountMinor < 0 || share.AmountMinor < amount/int64(count) || share.AmountMinor > amount/int64(count)+1 {
				t.Fatalf("invalid share=%+v", share)
			}
		}

		if total != amount {
			t.Fatalf("total=%d amount=%d", total, amount)
		}
	})
}
