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
	ErrInvalidGroupID          = errors.New("invalid group ID")
	ErrGroupNotFound           = errors.New("group not found")
	ErrGroupForbidden          = errors.New("only the group owner can delete this group")
	ErrGroupMembershipRequired = errors.New("you must be a group member to invite people")
	ErrOwnerCannotLeave        = errors.New("the owner cannot leave; delete the group instead")
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
	ID                 string            `json:"id" firestore:"-"`
	Name               string            `json:"name" firestore:"name"`
	Description        string            `json:"description" firestore:"description"`
	CreatedBy          string            `json:"createdBy" firestore:"createdBy"`
	MemberUIDs         []string          `json:"memberUIDs" firestore:"memberUIDs"`
	CreatedAt          time.Time         `json:"createdAt" firestore:"createdAt"`
	PendingInvitations map[string]string `json:"-" firestore:"pendingInvitations,omitempty"`
}
