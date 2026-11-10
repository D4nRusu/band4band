package models_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/D4nRusu/band4band/backend/internal/models"
)

func TestListInputNormalize(t *testing.T) {
	input, err := models.ListInput{Name: "  Shopping  "}.Normalize()
	if err != nil || input.Name != "Shopping" {
		t.Fatalf("input=%+v err=%v", input, err)
	}

	for _, name := range []string{"", "   ", strings.Repeat("a", 51)} {
		if _, err := (models.ListInput{Name: name}).Normalize(); !errors.Is(err, models.ErrInvalidListName) {
			t.Fatalf("name=%q err=%v", name, err)
		}
	}
}

func TestListItemInputNormalize(t *testing.T) {
	input, err := models.ListItemInput{Text: " Milk "}.Normalize()
	if err != nil || input.Text != "Milk" || input.RotationUIDs == nil {
		t.Fatalf("input=%+v err=%v", input, err)
	}

	valid := models.ListItemInput{Text: "Dishes", AssigneeUID: "alice", RotationUIDs: []string{"alice", "bob"}}
	if _, err := valid.Normalize(); err != nil {
		t.Fatal(err)
	}

	for name, tc := range map[string]struct {
		input models.ListItemInput
		want  error
	}{
		"empty text":            {models.ListItemInput{Text: " "}, models.ErrInvalidListItemText},
		"long text":             {models.ListItemInput{Text: strings.Repeat("a", 201)}, models.ErrInvalidListItemText},
		"single member":         {models.ListItemInput{Text: "x", AssigneeUID: "alice", RotationUIDs: []string{"alice"}}, models.ErrInvalidListRotation},
		"no assignee":           {models.ListItemInput{Text: "x", RotationUIDs: []string{"alice", "bob"}}, models.ErrInvalidListRotation},
		"assignee not first":    {models.ListItemInput{Text: "x", AssigneeUID: "bob", RotationUIDs: []string{"alice", "bob"}}, models.ErrInvalidListRotation},
		"duplicate member":      {models.ListItemInput{Text: "x", AssigneeUID: "alice", RotationUIDs: []string{"alice", "alice"}}, models.ErrInvalidListRotation},
		"empty rotation member": {models.ListItemInput{Text: "x", AssigneeUID: "alice", RotationUIDs: []string{"alice", ""}}, models.ErrInvalidListRotation},
	} {
		if _, err := tc.input.Normalize(); !errors.Is(err, tc.want) {
			t.Fatalf("%s: err=%v", name, err)
		}
	}
}

func TestNextInRotation(t *testing.T) {
	rotation := []string{"alice", "bob", "charlie"}
	members := []string{"alice", "bob", "charlie"}

	for current, want := range map[string]string{"alice": "bob", "bob": "charlie", "charlie": "alice"} {
		if got := models.NextInRotation(rotation, current, members); got != want {
			t.Fatalf("after %s got %s, want %s", current, got, want)
		}
	}

	if got := models.NextInRotation(rotation, "alice", []string{"alice", "charlie"}); got != "charlie" {
		t.Fatalf("skipping former member got %s", got)
	}

	if got := models.NextInRotation(rotation, "alice", []string{"alice"}); got != "alice" {
		t.Fatalf("only current member left got %s", got)
	}

	if got := models.NextInRotation(rotation, "alice", []string{"dana"}); got != "" {
		t.Fatalf("no members left got %s", got)
	}
}
