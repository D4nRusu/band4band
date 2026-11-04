package models

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"
)

var (
	ErrInvalidGroupName        = errors.New("group name must contain 1–100 characters")
	ErrInvalidGroupDescription = errors.New("group description must contain at most 1000 characters")
)

type GroupInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (input GroupInput) Normalize() (GroupInput, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.Description = strings.TrimSpace(input.Description)

	if length := utf8.RuneCountInString(input.Name); length == 0 || length > 100 {
		return GroupInput{}, ErrInvalidGroupName
	}

	if utf8.RuneCountInString(input.Description) > 1000 {
		return GroupInput{}, ErrInvalidGroupDescription
	}

	return input, nil
}

type Group struct {
	ID          string    `json:"id" firestore:"-"`
	Name        string    `json:"name" firestore:"name"`
	Description string    `json:"description" firestore:"description"`
	CreatedBy   string    `json:"createdBy" firestore:"createdBy"`
	MemberUIDs  []string  `json:"memberUIDs" firestore:"memberUIDs"`
	CreatedAt   time.Time `json:"createdAt" firestore:"createdAt"`
}
