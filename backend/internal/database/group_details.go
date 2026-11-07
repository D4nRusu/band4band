package database

import (
	"context"
	"fmt"
	"slices"

	"cloud.google.com/go/firestore"

	"github.com/D4nRusu/band4band/backend/internal/models"
)

func (s *GroupStore) Details(ctx context.Context, uid, groupID string) (models.Group, error) {
	if !validDocumentID(groupID) {
		return models.Group{}, models.ErrInvalidGroupID
	}

	var group models.Group
	err := s.db.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		var err error
		group, err = s.readMemberGroup(tx, uid, groupID)
		return err
	})

	return group, err
}

func (s *GroupStore) Members(ctx context.Context, uid, groupID string) ([]models.GroupMember, error) {
	if !validDocumentID(groupID) {
		return nil, models.ErrInvalidGroupID
	}

	var members []models.GroupMember
	err := s.db.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		group, err := s.readMemberGroup(tx, uid, groupID)
		if err != nil {
			return err
		}

		refs := make([]*firestore.DocumentRef, 0, len(group.MemberUIDs))

		for _, memberUID := range group.MemberUIDs {
			if !validDocumentID(memberUID) {
				return fmt.Errorf("invalid stored member UID")
			}

			refs = append(refs, s.db.Collection("users").Doc(memberUID))
		}

		snapshots, err := tx.GetAll(refs)
		if err != nil {
			return err
		}

		members = make([]models.GroupMember, 0, len(snapshots))

		for i, snapshot := range snapshots {
			var profile models.User
			if snapshot.Exists() {
				if err := snapshot.DataTo(&profile); err != nil {
					return err
				}
			}

			members = append(members, models.GroupMember{
				UID:      group.MemberUIDs[i],
				Name:     profile.Name,
				Email:    profile.Email,
				Username: profile.Username,
				PhotoURL: profile.PhotoURL,
			})
		}

		return nil
	})

	return members, err
}

func (s *GroupStore) readMemberGroup(tx *firestore.Transaction, uid, groupID string) (models.Group, error) {
	group, err := readGroup(tx, s.db.Collection("groups").Doc(groupID))
	if err != nil {
		return models.Group{}, err
	}

	if uid == "" || !slices.Contains(group.MemberUIDs, uid) {
		return models.Group{}, models.ErrGroupAccessForbidden
	}

	return group, nil
}
