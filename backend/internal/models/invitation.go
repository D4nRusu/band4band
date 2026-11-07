package models

import (
	"errors"
	"time"
)

var (
	ErrInviteUserNotFound  = errors.New("no user has that username")
	ErrAlreadyGroupMember  = errors.New("that user is already a group member")
	ErrInvitationNotFound  = errors.New("this invitation is no longer available")
	ErrInvitationForbidden = errors.New("only the invited user can respond")
	ErrInvalidInvitationID = errors.New("invalid invitation ID")
)

type GroupInvitation struct {
	ID              string    `json:"id" firestore:"-"`
	GroupID         string    `json:"groupID" firestore:"groupID"`
	GroupName       string    `json:"groupName" firestore:"groupName"`
	InviterUID      string    `json:"inviterUID" firestore:"inviterUID"`
	InviterUsername string    `json:"inviterUsername" firestore:"inviterUsername"`
	InviterName     string    `json:"inviterName" firestore:"inviterName"`
	RecipientUID    string    `json:"-" firestore:"recipientUID"`
	CreatedAt       time.Time `json:"createdAt" firestore:"createdAt"`
}
