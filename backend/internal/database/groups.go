package database

import (
	"context"
	"fmt"
	"sort"

	"cloud.google.com/go/firestore"

	"github.com/D4nRusu/band4band/backend/internal/models"
)

type GroupStore struct {
	db *firestore.Client
}

func NewGroupStore(db *firestore.Client) *GroupStore {
	return &GroupStore{db: db}
}

func (s *GroupStore) ListForUser(ctx context.Context, uid string) ([]models.Group, error) {
	if uid == "" {
		return nil, fmt.Errorf("missing authenticated UID")
	}

	snapshots, err := s.db.Collection("groups").Where("memberUIDs", "array-contains", uid).Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	groups := make([]models.Group, 0, len(snapshots))

	for _, snapshot := range snapshots {
		var group models.Group
		if err := snapshot.DataTo(&group); err != nil {
			return nil, err
		}

		group.ID = snapshot.Ref.ID
		groups = append(groups, group)
	}

	// Sorting here keeps the membership query independent of a composite index.
	sort.Slice(groups, func(i, j int) bool {
		if groups[i].CreatedAt.Equal(groups[j].CreatedAt) {
			return groups[i].ID < groups[j].ID
		}

		return groups[i].CreatedAt.After(groups[j].CreatedAt)
	})

	return groups, nil
}

func (s *GroupStore) Create(ctx context.Context, uid string, input models.GroupInput) (models.Group, error) {
	if uid == "" {
		return models.Group{}, fmt.Errorf("missing authenticated UID")
	}

	input, err := input.Normalize()
	if err != nil {
		return models.Group{}, err
	}

	ref := s.db.Collection("groups").NewDoc()
	group := models.Group{
		ID:          ref.ID,
		Name:        input.Name,
		Description: input.Description,
		CreatedBy:   uid,
		MemberUIDs:  []string{uid},
	}

	// The group and creator's membership are committed in one document write.
	result, err := ref.Create(ctx, map[string]any{
		"name":        group.Name,
		"description": group.Description,
		"createdBy":   group.CreatedBy,
		"memberUIDs":  group.MemberUIDs,
		"createdAt":   firestore.ServerTimestamp,
	})
	if err != nil {
		return models.Group{}, err
	}

	group.CreatedAt = result.UpdateTime

	return group, nil
}

func (s *GroupStore) Delete(ctx context.Context, uid, groupID string) error {
	if uid == "" {
		return fmt.Errorf("missing authenticated UID")
	}

	if groupID == "" {
		return fmt.Errorf("missing group ID")
	}

	ref := s.db.Collection("groups").Doc(groupID)

	snapshot, err := ref.Get(ctx)
	if err != nil {
		return err
	}

	var group models.Group
	if err := snapshot.DataTo(&group); err != nil {
		return err
	}

	if group.CreatedBy != uid {
		return fmt.Errorf("user %s is not the creator of group %s", uid, groupID)
	}

	_, err = ref.Delete(ctx)
	if err != nil {
		return err
	}

	return nil
}
