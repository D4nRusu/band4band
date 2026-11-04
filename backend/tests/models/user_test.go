package models_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/D4nRusu/band4band/backend/internal/models"
)

func TestNormalizeUsername(t *testing.T) {
	for _, input := range []string{"Alice_123", "ALICE_123", "alice_123"} {
		got, err := models.NormalizeUsername(input)
		if err != nil || got != "alice_123" {
			t.Fatalf("%q: got %q, %v", input, got, err)
		}
	}
	for _, input := range []string{"", "ab", strings.Repeat("a", 31), "alice bob", " alice", "alice ", "alice/bob", "ali.ce", "álîce", "Kelvin"} {
		if _, err := models.NormalizeUsername(input); !errors.Is(err, models.ErrInvalidUsername) {
			t.Fatalf("accepted %q: %v", input, err)
		}
	}
}
