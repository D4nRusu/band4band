package database

import (
	"context"
	"fmt"
	"slices"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/D4nRusu/band4band/backend/internal/models"
)

type InvitationStore struct {
	db *firestore.Client
}

func NewInvitationStore(db *firestore.Client) *InvitationStore {
	return &InvitationStore{db: db}
}

func (s *InvitationStore) Invite(ctx context.Context, sender models.Identity, groupID, username string) error {
	if !validDocumentID(groupID) {
		return models.ErrInvalidGroupID
	}

	if !validDocumentID(sender.UID) {
		return fmt.Errorf("invalid authenticated UID")
	}

	username, err := models.NormalizeUsername(username)
	if err != nil {
		return err
	}

	groupRef := s.db.Collection("groups").Doc(groupID)
	invitationRef := s.db.Collection("groupInvitations").NewDoc()

	return s.db.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		group, err := readGroup(tx, groupRef)
		if err != nil {
			return err
		}

		if !slices.Contains(group.MemberUIDs, sender.UID) {
			return models.ErrGroupMembershipRequired
		}

		recipientUID, err := s.recipientForUsername(tx, username)
		if err != nil {
			return err
		}

		if slices.Contains(group.MemberUIDs, recipientUID) {
			return models.ErrAlreadyGroupMember
		}

		if pendingID := group.PendingInvitations[recipientUID]; pendingID != "" {
			_, err := tx.Get(s.db.Collection("groupInvitations").Doc(pendingID))
			if err == nil {
				return nil // Preserve the original sender and avoid duplicate cards.
			}

			if status.Code(err) != codes.NotFound {
				return err
			}
		}

		inviterUsername, err := s.inviterUsername(tx, sender.UID)
		if err != nil {
			return err
		}

		if err := tx.Create(invitationRef, map[string]any{
			"groupID":         groupID,
			"groupName":       group.Name,
			"recipientUID":    recipientUID,
			"inviterUID":      sender.UID,
			"inviterUsername": inviterUsername,
			"inviterName":     sender.Name,
			"createdAt":       firestore.ServerTimestamp,
		}); err != nil {
			return err
		}

		// FieldPath safely handles any dots in a Firebase UID.
		return tx.Update(groupRef, []firestore.Update{{
			FieldPath: []string{"pendingInvitations", recipientUID},
			Value:     invitationRef.ID,
		}})
	})
}

func (s *InvitationStore) recipientForUsername(tx *firestore.Transaction, username string) (string, error) {
	reservation, err := tx.Get(s.db.Collection("usernames").Doc(username))
	if status.Code(err) == codes.NotFound {
		return "", models.ErrInviteUserNotFound
	}

	if err != nil {
		return "", err
	}

	var owner struct {
		UID string `firestore:"uid"`
	}
	if err := reservation.DataTo(&owner); err != nil {
		return "", err
	}

	if !validDocumentID(owner.UID) {
		return "", models.ErrInviteUserNotFound
	}

	profile, err := tx.Get(s.db.Collection("users").Doc(owner.UID))
	if status.Code(err) == codes.NotFound {
		return "", models.ErrInviteUserNotFound
	}

	if err != nil {
		return "", err
	}

	var user models.User
	if err := profile.DataTo(&user); err != nil {
		return "", err
	}

	if user.Username != username || !user.ProfileComplete {
		return "", models.ErrInviteUserNotFound
	}

	return owner.UID, nil
}

func (s *InvitationStore) inviterUsername(tx *firestore.Transaction, uid string) (string, error) {
	snapshot, err := tx.Get(s.db.Collection("users").Doc(uid))
	if status.Code(err) == codes.NotFound {
		return "", nil // The verified Google name is available as a display fallback.
	}

	if err != nil {
		return "", err
	}

	var user models.User
	if err := snapshot.DataTo(&user); err != nil {
		return "", err
	}

	return user.Username, nil
}
