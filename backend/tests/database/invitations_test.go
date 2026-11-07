package database_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/D4nRusu/band4band/backend/internal/database"
	"github.com/D4nRusu/band4band/backend/internal/models"
)

func seedInviteUser(t *testing.T, users *database.UserStore, ctx context.Context, uid, username string) models.Identity {
	t.Helper()
	identity := models.Identity{UID: uid, Name: username}
	if _, err := users.GetOrCreate(ctx, identity); err != nil {
		t.Fatal(err)
	}

	if _, err := users.SetUsername(ctx, uid, username); err != nil {
		t.Fatal(err)
	}

	return identity
}

func TestInvitationLifecycleAndMembership(t *testing.T) {
	users, db, ctx := emulatorStore(t)
	groups := database.NewGroupStore(db)
	invitations := database.NewInvitationStore(db)
	owner := seedInviteUser(t, users, ctx, "owner", "owner_handle")
	recipient := seedInviteUser(t, users, ctx, "recipient", "alice")
	seedInviteUser(t, users, ctx, "third", "charlie")

	group, err := groups.Create(ctx, owner.UID, models.GroupInput{Name: "Trip"})
	if err != nil {
		t.Fatal(err)
	}

	if err := invitations.Invite(ctx, recipient, group.ID, "charlie"); !errors.Is(err, models.ErrGroupMembershipRequired) {
		t.Fatalf("non-member invited: %v", err)
	}

	if err := invitations.Invite(ctx, owner, group.ID, "ALICE"); err != nil {
		t.Fatal(err)
	}

	inbox, err := invitations.ListForUser(ctx, recipient.UID)
	if err != nil || len(inbox) != 1 || inbox[0].InviterUsername != "owner_handle" {
		t.Fatalf("unexpected inbox: %+v, %v", inbox, err)
	}
	firstID := inbox[0].ID

	if err := invitations.Respond(ctx, "third", firstID, true); !errors.Is(err, models.ErrInvitationForbidden) {
		t.Fatalf("non-recipient responded: %v", err)
	}

	if err := invitations.Respond(ctx, recipient.UID, firstID, false); err != nil {
		t.Fatal(err)
	}

	if err := invitations.Invite(ctx, owner, group.ID, "alice"); err != nil {
		t.Fatal(err)
	}

	inbox, err = invitations.ListForUser(ctx, recipient.UID)
	if err != nil || len(inbox) != 1 || inbox[0].ID == firstID {
		t.Fatalf("re-invite did not get a new ID: %+v, %v", inbox, err)
	}

	if err := invitations.Respond(ctx, recipient.UID, firstID, true); !errors.Is(err, models.ErrInvitationNotFound) {
		t.Fatalf("old response affected new invite: %v", err)
	}

	if err := invitations.Respond(ctx, recipient.UID, inbox[0].ID, true); err != nil {
		t.Fatal(err)
	}

	joined, err := groups.ListForUser(ctx, recipient.UID)
	if err != nil || len(joined) != 1 || joined[0].ID != group.ID {
		t.Fatalf("accept did not add membership: %+v, %v", joined, err)
	}

	if err := invitations.Invite(ctx, owner, group.ID, "alice"); !errors.Is(err, models.ErrAlreadyGroupMember) {
		t.Fatalf("existing member invited: %v", err)
	}

	if err := invitations.Invite(ctx, recipient, group.ID, "charlie"); err != nil {
		t.Fatalf("regular member could not invite: %v", err)
	}

	if err := groups.Leave(ctx, owner.UID, group.ID); !errors.Is(err, models.ErrOwnerCannotLeave) {
		t.Fatalf("owner could leave: %v", err)
	}

	if err := groups.Leave(ctx, recipient.UID, group.ID); err != nil {
		t.Fatal(err)
	}

	joined, err = groups.ListForUser(ctx, recipient.UID)
	if err != nil || len(joined) != 0 {
		t.Fatalf("leave did not remove membership: %+v, %v", joined, err)
	}

	if err := invitations.Invite(ctx, recipient, group.ID, "owner_handle"); !errors.Is(err, models.ErrGroupMembershipRequired) {
		t.Fatalf("departed member could invite: %v", err)
	}

	if err := groups.Delete(ctx, owner.UID, group.ID); err != nil {
		t.Fatal(err)
	}

	remaining, err := db.Collection("groupInvitations").Where("groupID", "==", group.ID).Documents(ctx).GetAll()
	if err != nil || len(remaining) != 0 {
		t.Fatalf("deletion left pending invites: %d, %v", len(remaining), err)
	}
}

func TestConcurrentInvitesProduceOneCard(t *testing.T) {
	users, db, ctx := emulatorStore(t)
	groups := database.NewGroupStore(db)
	invitations := database.NewInvitationStore(db)
	owner := seedInviteUser(t, users, ctx, "owner", "owner_handle")
	seedInviteUser(t, users, ctx, "recipient", "alice")

	group, err := groups.Create(ctx, owner.UID, models.GroupInput{Name: "Trip"})
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	errorsFound := make(chan error, 2)
	start := make(chan struct{})

	for _, username := range []string{"alice", "ALICE"} {
		wg.Add(1)
		go func(username string) {
			defer wg.Done()
			<-start
			errorsFound <- invitations.Invite(ctx, owner, group.ID, username)
		}(username)
	}

	close(start)
	wg.Wait()
	close(errorsFound)

	for err := range errorsFound {
		if err != nil {
			t.Fatal(err)
		}
	}

	inbox, err := invitations.ListForUser(ctx, "recipient")
	if err != nil || len(inbox) != 1 {
		t.Fatalf("duplicate invitation cards: %+v, %v", inbox, err)
	}
}
