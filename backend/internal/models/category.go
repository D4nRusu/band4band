package models

import (
	"errors"
	"strings"
	"unicode/utf8"
)

var (
	ErrInvalidCategoryID   = errors.New("invalid category ID")
	ErrDefaultCategory     = errors.New("default categories cannot be deleted")
	ErrInvalidCategoryName = errors.New("category name must contain 1-40 characters")
	ErrInvalidCategoryIcon = errors.New("choose a supported category icon")
	ErrCategoryExists      = errors.New("a category with this name already exists")
)

type Category struct {
	ID        string `json:"id" firestore:"id"`
	Name      string `json:"name" firestore:"name"`
	Icon      string `json:"icon" firestore:"icon"`
	IsDefault bool   `json:"isDefault" firestore:"isDefault"`
}

type CategoryInput struct {
	Name string `json:"name"`
	Icon string `json:"icon"`
}

func DefaultCategories() []Category {
	return []Category{
		{"default_food", "Food & drinks", "food", true},
		{"default_groceries", "Groceries", "groceries", true},
		{"default_transport", "Transport", "transport", true},
		{"default_accommodation", "Accommodation", "accommodation", true},
		{"default_utilities", "Utilities", "utilities", true},
		{"default_entertainment", "Entertainment", "entertainment", true},
		{"default_shopping", "Shopping", "shopping", true},
		{"default_health", "Health", "health", true},
		{"default_travel", "Travel", "travel", true},
		{"default_other", "Other", "other", true},
	}
}

func CategoryNameKey(name string) string {
	return strings.ToLower(strings.Join(strings.Fields(name), " "))
}

func (input CategoryInput) Normalize() (CategoryInput, error) {
	input.Name = strings.Join(strings.Fields(input.Name), " ")
	if length := utf8.RuneCountInString(input.Name); length < 1 || length > 40 {
		return CategoryInput{}, ErrInvalidCategoryName
	}

	for _, category := range DefaultCategories() {
		if input.Icon == category.Icon {
			return input, nil
		}
	}

	return CategoryInput{}, ErrInvalidCategoryIcon
}
