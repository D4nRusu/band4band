package models

import (
	"errors"
	"regexp"
	"strings"
	"time"
)

var (
	ErrInvalidUsername = errors.New("username must contain 3-30 letters, numbers, or underscores")
	ErrUsernameTaken   = errors.New("that username is already taken; choose another")
	ErrProfileComplete = errors.New("the profile already has a username")
)

var usernamePattern = regexp.MustCompile(`^[a-zA-Z0-9_]{3,30}$`)

// NormalizeUsername defines the canonical handle used for uniqueness and lookup.
func NormalizeUsername(username string) (string, error) {
	if !usernamePattern.MatchString(username) {
		return "", ErrInvalidUsername
	}
	return strings.ToLower(username), nil
}

// Identity contains only claims from a verified Firebase ID token.
type Identity struct {
	UID      string
	Email    string
	Name     string
	PhotoURL string
}

type User struct {
	UID             string    `json:"uid" firestore:"uid"`
	Email           string    `json:"email" firestore:"email"`
	Name            string    `json:"name" firestore:"name"`
	Username        string    `json:"username" firestore:"username"`
	PhotoURL        string    `json:"photoURL" firestore:"photoURL"`
	CreatedAt       time.Time `json:"createdAt" firestore:"createdAt"`
	ProfileComplete bool      `json:"profileComplete" firestore:"profileComplete"`
}
