package database

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/D4nRusu/band4band/backend/internal/models"
)

type UserStore struct {
	db *firestore.Client
}

func NewUserStore(db *firestore.Client) *UserStore {
	return &UserStore{db: db}
}

func (s *UserStore) GetOrCreate(ctx context.Context, identity models.Identity) (models.User, error) {
	if identity.UID == "" || strings.Contains(identity.UID, "/") {
		return models.User{}, fmt.Errorf("invalid user document ID")
	}
	ref := s.db.Collection("users").Doc(identity.UID)
	snapshot, err := ref.Get(ctx)
	if status.Code(err) == codes.NotFound {
		// Create, rather than Set, preserves an existing profile during concurrent logins.
		_, err = ref.Create(ctx, map[string]any{
			"uid":             identity.UID,
			"email":           identity.Email,
			"name":            identity.Name,
			"username":        "",
			"photoURL":        identity.PhotoURL,
			"createdAt":       firestore.ServerTimestamp,
			"profileComplete": false,
		})
		if err != nil && status.Code(err) != codes.AlreadyExists {
			return models.User{}, err
		}
		snapshot, err = ref.Get(ctx)
	}
	if err != nil {
		return models.User{}, err
	}
	var user models.User
	if err := snapshot.DataTo(&user); err != nil {
		return models.User{}, err
	}
	return user, nil
}

// SetUsername completes onboarding; username changes are outside this milestone.
func (s *UserStore) SetUsername(ctx context.Context, uid, username string) (models.User, error) {
	username, err := models.NormalizeUsername(username)
	if err != nil {
		return models.User{}, err
	}
	if uid == "" || strings.Contains(uid, "/") {
		return models.User{}, fmt.Errorf("invalid user document ID")
	}
	userRef := s.db.Collection("users").Doc(uid)
	nameRef := s.db.Collection("usernames").Doc(username)
	var user models.User
	err = s.db.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		snapshot, err := tx.Get(userRef)
		if err != nil {
			return err
		}
		if err := snapshot.DataTo(&user); err != nil {
			return err
		}
		if user.Username != "" {
			if user.Username == username {
				return nil // Safe retry if the first response was lost.
			}
			return models.ErrProfileComplete
		}
		_, err = tx.Get(nameRef)
		if err == nil {
			return models.ErrUsernameTaken
		}
		if status.Code(err) != codes.NotFound {
			return err
		}
		if err := tx.Create(nameRef, map[string]any{"uid": uid}); err != nil {
			return err
		}
		if err := tx.Update(userRef, []firestore.Update{
			{Path: "username", Value: username},
			{Path: "profileComplete", Value: true},
		}); err != nil {
			return err
		}
		user.Username = username
		user.ProfileComplete = true
		return nil
	})
	return user, err
}
