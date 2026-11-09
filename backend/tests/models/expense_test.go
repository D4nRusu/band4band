package models_test

import (
	"strings"
	"testing"

	"github.com/D4nRusu/band4band/backend/internal/models"
)

func TestExpenseValidation(t *testing.T) {
	valid := models.ExpenseInput{Name: " Lunch ", CategoryID: "default_food", PaidBy: "payer", AmountMinor: 1250}
	normalized, err := valid.Normalize()
	if err != nil || normalized.Name != "Lunch" || normalized.AmountMinor != 1250 {
		t.Fatalf("normalized=%+v err=%v", normalized, err)
	}

	for _, tc := range []struct {
		name   string
		change func(*models.ExpenseInput)
	}{
		{"empty name", func(input *models.ExpenseInput) { input.Name = " " }},
		{"long name", func(input *models.ExpenseInput) { input.Name = strings.Repeat("a", 101) }},
		{"zero amount", func(input *models.ExpenseInput) { input.AmountMinor = 0 }},
		{"negative amount", func(input *models.ExpenseInput) { input.AmountMinor = -1 }},
		{"unsafe JSON integer", func(input *models.ExpenseInput) { input.AmountMinor = models.MaxAmountMinor + 1 }},
		{"missing payer", func(input *models.ExpenseInput) { input.PaidBy = "" }},
		{"missing category", func(input *models.ExpenseInput) { input.CategoryID = "" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := valid
			tc.change(&input)
			if _, err := input.Normalize(); err == nil {
				t.Fatalf("accepted %+v", input)
			}
		})
	}

	for _, amount := range []int64{1, models.MaxAmountMinor} {
		input := valid
		input.AmountMinor = amount
		if _, err := input.Normalize(); err != nil {
			t.Fatalf("amount=%d err=%v", amount, err)
		}
	}
}
