package models

import (
	"errors"
	"slices"
	"strings"
	"time"
	"unicode/utf8"
)

const maxRotationMembers = 20

var (
	ErrInvalidListID        = errors.New("invalid list ID")
	ErrInvalidListItemID    = errors.New("invalid list item ID")
	ErrInvalidListName      = errors.New("list name must contain 1–50 characters")
	ErrInvalidListItemText  = errors.New("item text must contain 1–200 characters")
	ErrInvalidListAssignee  = errors.New("the assignee must be a current group member")
	ErrInvalidListRotation  = errors.New("a rotation needs 2–20 different current group members, starting with the assignee")
	ErrListItemCleared      = errors.New("cleared items are kept as history and cannot be changed")
	ErrListItemFollowUpDone = errors.New("the next turn of this chore is already done, so this one cannot be reopened")
	ErrListNotFound         = errors.New("this list no longer exists")
	ErrListItemNotFound     = errors.New("this item no longer exists")
	ErrListForbidden        = errors.New("only the list creator can delete this list")
	ErrListItemForbidden    = errors.New("only the item creator or assignee can delete this item")
)

type GroupList struct {
	ID        string    `json:"id" firestore:"-"`
	GroupID   string    `json:"groupID" firestore:"-"`
	Name      string    `json:"name" firestore:"name"`
	CreatedBy string    `json:"createdBy" firestore:"createdBy"`
	CreatedAt time.Time `json:"createdAt" firestore:"createdAt"`
}

type ListItem struct {
	ID           string     `json:"id" firestore:"-"`
	ListID       string     `json:"listID" firestore:"-"`
	Text         string     `json:"text" firestore:"text"`
	CreatedBy    string     `json:"createdBy" firestore:"createdBy"`
	CreatedAt    time.Time  `json:"createdAt" firestore:"createdAt"`
	AssigneeUID  string     `json:"assigneeUID" firestore:"assigneeUID"`
	AssigneeName string     `json:"assigneeName" firestore:"assigneeName"`
	RotationUIDs []string   `json:"rotationUIDs" firestore:"rotationUIDs"`
	Done         bool       `json:"done" firestore:"done"`
	DoneBy       string     `json:"doneBy" firestore:"doneBy"`
	DoneByName   string     `json:"doneByName" firestore:"doneByName"`
	DoneAt       *time.Time `json:"doneAt,omitempty" firestore:"doneAt,omitempty"`
	ClearedAt    *time.Time `json:"clearedAt,omitempty" firestore:"clearedAt,omitempty"`
	FollowUpID   string     `json:"followUpID" firestore:"followUpID"`
}

type ListInput struct {
	Name string `json:"name"`
}

type ListItemInput struct {
	Text         string   `json:"text"`
	AssigneeUID  string   `json:"assigneeUID"`
	RotationUIDs []string `json:"rotationUIDs"`
}

func (input ListInput) Normalize() (ListInput, error) {
	input.Name = strings.TrimSpace(input.Name)
	if length := utf8.RuneCountInString(input.Name); length < 1 || length > 50 {
		return ListInput{}, ErrInvalidListName
	}

	return input, nil
}

// Normalize checks the input shape; membership is checked against the group later.
// A rotation starts with the assignee so the first turn is explicit.
func (input ListItemInput) Normalize() (ListItemInput, error) {
	input.Text = strings.TrimSpace(input.Text)
	if length := utf8.RuneCountInString(input.Text); length < 1 || length > 200 {
		return ListItemInput{}, ErrInvalidListItemText
	}

	if len(input.RotationUIDs) == 0 {
		input.RotationUIDs = []string{}
		return input, nil
	}

	if len(input.RotationUIDs) < 2 || len(input.RotationUIDs) > maxRotationMembers {
		return ListItemInput{}, ErrInvalidListRotation
	}

	if input.AssigneeUID == "" || input.RotationUIDs[0] != input.AssigneeUID {
		return ListItemInput{}, ErrInvalidListRotation
	}

	seen := make(map[string]bool, len(input.RotationUIDs))
	for _, uid := range input.RotationUIDs {
		if uid == "" || seen[uid] {
			return ListItemInput{}, ErrInvalidListRotation
		}

		seen[uid] = true
	}

	return input, nil
}

// NextInRotation returns the member after current, skipping people who left the group.
// It returns "" when nobody else in the rotation is still a member.
func NextInRotation(rotation []string, current string, memberUIDs []string) string {
	start := slices.Index(rotation, current)
	for step := 1; step <= len(rotation); step++ {
		candidate := rotation[(start+step)%len(rotation)]
		if slices.Contains(memberUIDs, candidate) {
			return candidate
		}
	}

	return ""
}
