package database

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/D4nRusu/band4band/backend/internal/models"
)

func validDocumentID(id string) bool {
	return id != "" && len(id) <= 1500 && id != "." && id != ".." && !strings.Contains(id, "/")
}

func readGroup(tx *firestore.Transaction, ref *firestore.DocumentRef) (models.Group, error) {
	snapshot, err := tx.Get(ref)
	if status.Code(err) == codes.NotFound {
		return models.Group{}, models.ErrGroupNotFound
	}

	if err != nil {
		return models.Group{}, err
	}

	var group models.Group
	if err := snapshot.DataTo(&group); err != nil {
		return models.Group{}, err
	}

	group.ID = ref.ID
	return group, nil
}

func (s *GroupStore) Leave(ctx context.Context, uid, groupID string) error {
	if uid == "" {
		return fmt.Errorf("missing authenticated UID")
	}

	if !validDocumentID(groupID) {
		return models.ErrInvalidGroupID
	}

	ref := s.db.Collection("groups").Doc(groupID)

	return s.db.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		group, err := readGroup(tx, ref)
		if err != nil {
			return err
		}

		if group.CreatedBy == uid {
			return models.ErrOwnerCannotLeave
		}

		if !slices.Contains(group.MemberUIDs, uid) {
			return nil // Safe retry if a previous leave already succeeded.
		}

		return tx.Update(ref, []firestore.Update{{Path: "memberUIDs", Value: firestore.ArrayRemove(uid)}})
	})
}
