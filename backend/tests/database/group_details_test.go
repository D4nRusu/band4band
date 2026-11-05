package database_test

import (
	"errors"
	"testing"

	"cloud.google.com/go/firestore"

	"github.com/D4nRusu/band4band/backend/internal/database"
	"github.com/D4nRusu/band4band/backend/internal/models"
)

func TestGroupDetailsRequireCurrentMembership(t *testing.T) {
	users, db, ctx := emulatorStore(t)
	groups := database.NewGroupStore(db)
	owner := models.Identity{UID: "owner", Name: "Alice", Email: "alice@example.com", PhotoURL: "https://example.com/photo.png"}
	if _, err := users.GetOrCreate(ctx, owner); err != nil {
		t.Fatal(err)
	}

	group, err := groups.Create(ctx, owner.UID, models.GroupInput{Name: "Trip"})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := groups.Details(ctx, "outsider", group.ID); !errors.Is(err, models.ErrGroupAccessForbidden) {
		t.Fatalf("outsider read group: %v", err)
	}

	if _, err := groups.Members(ctx, "outsider", group.ID); !errors.Is(err, models.ErrGroupAccessForbidden) {
		t.Fatalf("outsider read profiles: %v", err)
	}

	if _, err := db.Collection("groups").Doc(group.ID).Update(ctx, []firestore.Update{
		{Path: "memberUIDs", Value: firestore.ArrayUnion("missing-profile")},
	}); err != nil {
		t.Fatal(err)
	}

	members, err := groups.Members(ctx, owner.UID, group.ID)
	if err != nil || len(members) != 2 {
		t.Fatalf("members=%+v err=%v", members, err)
	}

	if members[0].UID != owner.UID || members[0].Email != owner.Email || members[0].PhotoURL != owner.PhotoURL {
		t.Fatalf("profile=%+v", members[0])
	}

	if members[1].UID != "missing-profile" || members[1].Name != "" {
		t.Fatalf("missing profile fallback=%+v", members[1])
	}

	if err := groups.Leave(ctx, "missing-profile", group.ID); err != nil {
		t.Fatal(err)
	}

	if _, err := groups.Members(ctx, "missing-profile", group.ID); !errors.Is(err, models.ErrGroupAccessForbidden) {
		t.Fatalf("former member read profiles: %v", err)
	}

	if _, err := groups.Details(ctx, owner.UID, "invalid/id"); !errors.Is(err, models.ErrInvalidGroupID) {
		t.Fatalf("invalid ID: %v", err)
	}

	if _, err := groups.Details(ctx, owner.UID, "missing-group"); !errors.Is(err, models.ErrGroupNotFound) {
		t.Fatalf("missing group: %v", err)
	}
}
