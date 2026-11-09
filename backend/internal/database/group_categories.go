package database

import (
	"context"
	"crypto/sha256"
	"fmt"
	"sort"

	"cloud.google.com/go/firestore"
	"github.com/D4nRusu/band4band/backend/internal/models"
)

func (s *GroupStore) Categories(ctx context.Context, uid, groupID string) ([]models.Category, error) {
	group, err := s.Details(ctx, uid, groupID)
	if err != nil {
		return nil, err
	}

	custom := make([]models.Category, 0, len(group.CustomCategories))
	for _, category := range group.CustomCategories {
		custom = append(custom, category)
	}

	sort.Slice(custom, func(i, j int) bool {
		return models.CategoryNameKey(custom[i].Name) < models.CategoryNameKey(custom[j].Name)
	})

	return append(models.DefaultCategories(), custom...), nil
}

func (s *GroupStore) AddCategory(ctx context.Context, uid, groupID string, input models.CategoryInput) (models.Category, error) {
	if !validDocumentID(groupID) {
		return models.Category{}, models.ErrInvalidGroupID
	}

	input, err := input.Normalize()
	if err != nil {
		return models.Category{}, err
	}

	key := models.CategoryNameKey(input.Name)
	category := models.Category{
		ID:   fmt.Sprintf("custom_%x", sha256.Sum256([]byte(key))),
		Name: input.Name,
		Icon: input.Icon,
	}

	err = s.db.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		group, err := s.readMemberGroup(tx, uid, groupID)
		if err != nil {
			return err
		}

		for _, existing := range models.DefaultCategories() {
			if models.CategoryNameKey(existing.Name) == key {
				return models.ErrCategoryExists
			}
		}

		for _, existing := range group.CustomCategories {
			if models.CategoryNameKey(existing.Name) == key {
				return models.ErrCategoryExists
			}
		}

		return tx.Update(s.db.Collection("groups").Doc(groupID), []firestore.Update{{
			FieldPath: []string{"customCategories", category.ID}, Value: category,
		}})
	})

	return category, err
}

func (s *GroupStore) DeleteCategory(ctx context.Context, uid, groupID, categoryID string) error {
	if !validDocumentID(groupID) {
		return models.ErrInvalidGroupID
	}

	if !validDocumentID(categoryID) {
		return models.ErrInvalidCategoryID
	}

	return s.db.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		group, err := s.readMemberGroup(tx, uid, groupID)
		if err != nil {
			return err
		}

		for _, category := range models.DefaultCategories() {
			if category.ID == categoryID {
				return models.ErrDefaultCategory
			}
		}

		if _, exists := group.CustomCategories[categoryID]; !exists {
			return nil
		}

		return tx.Update(s.db.Collection("groups").Doc(groupID), []firestore.Update{{
			FieldPath: []string{"customCategories", categoryID}, Value: firestore.Delete,
		}})
	})
}
