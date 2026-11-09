package models_test

import (
	"strings"
	"testing"

	"github.com/D4nRusu/band4band/backend/internal/models"
)

func TestCategoryValidation(t *testing.T) {
	for _, input := range []models.CategoryInput{
		{Name: " ", Icon: "other"},
		{Name: strings.Repeat("a", 41), Icon: "other"},
		{Name: "Pets", Icon: "untrusted-icon"},
	} {
		if _, err := input.Normalize(); err == nil {
			t.Fatalf("accepted %+v", input)
		}
	}

	input, err := (models.CategoryInput{Name: "  Pet   Care ", Icon: "other"}).Normalize()
	if err != nil || input.Name != "Pet Care" || models.CategoryNameKey(input.Name) != "pet care" {
		t.Fatalf("normalized=%+v err=%v", input, err)
	}

	defaults := models.DefaultCategories()
	ids := map[string]bool{}
	for _, category := range defaults {
		if ids[category.ID] || !category.IsDefault {
			t.Fatalf("invalid default=%+v", category)
		}
		ids[category.ID] = true
	}

	if len(defaults) != 10 {
		t.Fatalf("defaults=%d", len(defaults))
	}
}
