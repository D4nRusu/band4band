package database_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/D4nRusu/band4band/backend/internal/database"
	"github.com/D4nRusu/band4band/backend/internal/models"
)

func emulatorStore(t *testing.T) (*database.UserStore, *firestore.Client, context.Context) {
	t.Helper()
	if os.Getenv("FIRESTORE_EMULATOR_HOST") == "" {
		t.Skip("requires FIRESTORE_EMULATOR_HOST; never runs against production")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	// A separate project per test avoids interference without clearing existing data.
	project := fmt.Sprintf("demo-profile-%x", time.Now().UnixNano())
	db, err := firestore.NewClient(ctx, project)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return database.NewUserStore(db), db, ctx
}

func TestProfileCreationPreservesSavedInformation(t *testing.T) {
	store, db, ctx := emulatorStore(t)
	identity := models.Identity{UID: "user-1", Email: "original@example.test", Name: "Original Name", PhotoURL: "https://example.test/photo"}
	before := time.Now().Add(-time.Second)
	profile, err := store.GetOrCreate(ctx, identity)
	if err != nil {
		t.Fatal(err)
	}
	if profile.UID != identity.UID || profile.Email != identity.Email || profile.Name != identity.Name || profile.PhotoURL != identity.PhotoURL || profile.Username != "" || profile.ProfileComplete || profile.CreatedAt.Before(before) || profile.CreatedAt.After(time.Now().Add(time.Second)) {
		t.Fatalf("unexpected new profile: %+v", profile)
	}
	completed, err := store.SetUsername(ctx, identity.UID, "Alice_123")
	if err != nil {
		t.Fatal(err)
	}
	if completed.Username != "alice_123" || !completed.ProfileComplete || completed.CreatedAt != profile.CreatedAt {
		t.Fatalf("unexpected completed profile: %+v", completed)
	}
	identity.Email, identity.Name, identity.PhotoURL = "changed@example.test", "Changed Name", ""
	again, err := store.GetOrCreate(ctx, identity)
	if err != nil {
		t.Fatal(err)
	}
	if again != completed {
		t.Fatalf("subsequent login overwrote profile: %+v", again)
	}
	retried, err := store.SetUsername(ctx, identity.UID, "ALICE_123")
	if err != nil || retried != completed {
		t.Fatalf("retry: %+v, %v", retried, err)
	}
	if _, err := store.SetUsername(ctx, identity.UID, "another"); !errors.Is(err, models.ErrProfileComplete) {
		t.Fatalf("unexpected rename result: %v", err)
	}
	if _, err := db.Collection("usernames").Doc("another").Get(ctx); status.Code(err) != codes.NotFound {
		t.Fatalf("rename reserved a handle: %v", err)
	}
}

func TestConcurrentFirstLoginsCreateOneProfile(t *testing.T) {
	store, _, ctx := emulatorStore(t)
	var wg sync.WaitGroup
	results := make(chan models.User, 8)
	errs := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			user, err := store.GetOrCreate(ctx, models.Identity{UID: "shared-uid", Name: fmt.Sprintf("Name %d", i)})
			if err != nil {
				errs <- err
				return
			}
			results <- user
		}(i)
	}
	wg.Wait()
	close(results)
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
	var first *models.User
	for user := range results {
		if first == nil {
			copy := user
			first = &copy
		}
		if user != *first {
			t.Fatalf("inconsistent profiles: %+v vs %+v", user, *first)
		}
	}
}

func TestConcurrentUsernameClaimsHaveOneWinner(t *testing.T) {
	store, db, ctx := emulatorStore(t)
	for _, uid := range []string{"first", "second"} {
		if _, err := store.GetOrCreate(ctx, models.Identity{UID: uid}); err != nil {
			t.Fatal(err)
		}
	}
	type result struct {
		uid string
		err error
	}
	results := make(chan result, 2)
	start := make(chan struct{})
	for i, uid := range []string{"first", "second"} {
		go func(uid, name string) {
			<-start
			_, err := store.SetUsername(ctx, uid, name)
			results <- result{uid, err}
		}(uid, []string{"Alice", "ALICE"}[i])
	}
	close(start)
	winner, loser := "", ""
	for i := 0; i < 2; i++ {
		r := <-results
		if r.err == nil {
			if winner != "" {
				t.Fatal("both users claimed the same username")
			}
			winner = r.uid
		} else if errors.Is(r.err, models.ErrUsernameTaken) {
			loser = r.uid
		} else {
			t.Fatal(r.err)
		}
	}
	if winner == "" || loser == "" {
		t.Fatalf("winner=%q loser=%q", winner, loser)
	}
	reservation, err := db.Collection("usernames").Doc("alice").Get(ctx)
	if err != nil {
		t.Fatal(err)
	}
	owner, err := reservation.DataAt("uid")
	if err != nil || owner != winner {
		t.Fatalf("reservation owner=%v, %v", owner, err)
	}
	loserProfile, err := store.GetOrCreate(ctx, models.Identity{UID: loser})
	if err != nil || loserProfile.ProfileComplete || loserProfile.Username != "" {
		t.Fatalf("loser's profile changed: %+v, %v", loserProfile, err)
	}
	if _, err := store.SetUsername(ctx, loser, "available"); err != nil {
		t.Fatalf("loser cannot choose another handle: %v", err)
	}
}
