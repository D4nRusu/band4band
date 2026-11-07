package database_test

import (
	"errors"
	"testing"

	"cloud.google.com/go/firestore"
	"github.com/D4nRusu/band4band/backend/internal/database"
	"github.com/D4nRusu/band4band/backend/internal/models"
)

func TestGroupUpdateByMember(t *testing.T) {
	_, db, ctx := emulatorStore(t)
	groups := database.NewGroupStore(db)
	group, err := groups.Create(ctx, "owner", models.GroupInput{Name: "Original"})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := groups.Update(ctx, "outsider", group.ID, models.GroupInput{Name: "Forbidden"}); !errors.Is(err, models.ErrGroupAccessForbidden) {
		t.Fatalf("outsider update: %v", err)
	}

	if _, err := db.Collection("groups").Doc(group.ID).Update(ctx, []firestore.Update{{Path: "memberUIDs", Value: firestore.ArrayUnion("member")}}); err != nil {
		t.Fatal(err)
	}

	updated, err := groups.Update(ctx, "member", group.ID, models.GroupInput{Name: " Updated ", Description: " Details "})
	if err != nil {
		t.Fatal(err)
	}

	if updated.Name != "Updated" || updated.Description != "Details" || updated.CreatedBy != "owner" || len(updated.MemberUIDs) != 2 || !updated.CreatedAt.Equal(group.CreatedAt) {
		t.Fatalf("unexpected update: %+v", updated)
	}

	stored, err := groups.Details(ctx, "owner", group.ID)
	if err != nil || stored.Name != "Updated" {
		t.Fatalf("stored=%+v err=%v", stored, err)
	}

	if err := groups.Leave(ctx, "member", group.ID); err != nil {
		t.Fatal(err)
	}

	if _, err := groups.Update(ctx, "member", group.ID, models.GroupInput{Name: "Forbidden"}); !errors.Is(err, models.ErrGroupAccessForbidden) {
		t.Fatalf("former member update: %v", err)
	}
}
