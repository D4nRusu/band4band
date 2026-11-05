package database

import (
	"context"

	"cloud.google.com/go/firestore"

	"github.com/D4nRusu/band4band/backend/internal/models"
)

func (s *GroupStore) Update(ctx context.Context, uid, groupID string, input models.GroupInput) (models.Group, error) {
	if !validDocumentID(groupID) {
		return models.Group{}, models.ErrInvalidGroupID
	}

	input, err := input.Normalize()
	if err != nil {
		return models.Group{}, err
	}

	var group models.Group
	err = s.db.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		current, err := s.readMemberGroup(tx, uid, groupID)
		if err != nil {
			return err
		}

		err = tx.Update(s.db.Collection("groups").Doc(groupID), []firestore.Update{
			{Path: "name", Value: input.Name},
			{Path: "description", Value: input.Description},
		})
		if err != nil {
			return err
		}

		current.Name = input.Name
		current.Description = input.Description
		group = current
		return nil
	})

	return group, err
}
