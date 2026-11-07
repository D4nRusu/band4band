package database

import (
	"context"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/D4nRusu/band4band/backend/internal/models"
)

func (s *InvitationStore) Respond(ctx context.Context, uid, invitationID string, accept bool) error {
	if !validDocumentID(invitationID) {
		return models.ErrInvalidInvitationID
	}

	ref := s.db.Collection("groupInvitations").Doc(invitationID)

	return s.db.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		snapshot, err := tx.Get(ref)
		if status.Code(err) == codes.NotFound {
			return models.ErrInvitationNotFound
		}

		if err != nil {
			return err
		}

		var invitation models.GroupInvitation
		if err := snapshot.DataTo(&invitation); err != nil {
			return err
		}

		if invitation.RecipientUID != uid {
			return models.ErrInvitationForbidden
		}

		if !validDocumentID(invitation.GroupID) {
			return models.ErrGroupNotFound
		}

		groupRef := s.db.Collection("groups").Doc(invitation.GroupID)
		group, err := readGroup(tx, groupRef)
		if err != nil {
			return err
		}

		if group.PendingInvitations[uid] != invitationID {
			return models.ErrInvitationNotFound
		}

		updates := []firestore.Update{{
			FieldPath: []string{"pendingInvitations", uid},
			Value:     firestore.Delete,
		}}

		if accept {
			updates = append(updates, firestore.Update{Path: "memberUIDs", Value: firestore.ArrayUnion(uid)})
		}

		if err := tx.Update(groupRef, updates); err != nil {
			return err
		}

		return tx.Delete(ref)
	})
}
