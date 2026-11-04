package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/D4nRusu/band4band/backend/internal/models"
)

type UserStore interface {
	GetOrCreate(context.Context, models.Identity) (models.User, error)
	SetUsername(context.Context, string, string) (models.User, error)
}

type userHandler struct {
	users UserStore
}

func (h userHandler) currentUser(w http.ResponseWriter, r *http.Request) {
	identity, ok := identityFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	user, err := h.users.GetOrCreate(r.Context(), identity)
	if err != nil {
		writeUserError(w, err)
		return
	}
	writeUser(w, user)
}

func (h userHandler) setUsername(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Username string `json:"username"`
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1024)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&input); err != nil {
		writeProfileError(w, "invalid_request", "Expected a JSON object containing only username.", http.StatusBadRequest)
		return
	}

	if err := decoder.Decode(new(any)); err != io.EOF {
		writeProfileError(w, "invalid_request", "Expected a single JSON object.", http.StatusBadRequest)
		return
	}

	username, err := models.NormalizeUsername(input.Username)
	if err != nil {
		writeUserError(w, err)
		return
	}

	identity, ok := identityFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// The setup request is also safe if the initial GET did not finish.
	if _, err := h.users.GetOrCreate(r.Context(), identity); err != nil {
		writeUserError(w, err)
		return
	}

	user, err := h.users.SetUsername(r.Context(), identity.UID, username)
	if err != nil {
		writeUserError(w, err)
		return
	}
	writeUser(w, user)
}
