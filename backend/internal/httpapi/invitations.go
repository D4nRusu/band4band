package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/D4nRusu/band4band/backend/internal/models"
)

type InvitationStore interface {
	ListForUser(context.Context, string) ([]models.GroupInvitation, error)
	Invite(context.Context, models.Identity, string, string) error
	Respond(context.Context, string, string, bool) error
}

type invitationHandler struct {
	invitations InvitationStore
}

func (h invitationHandler) list(w http.ResponseWriter, r *http.Request) {
	uid, ok := UIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	invitations, err := h.invitations.ListForUser(r.Context(), uid)
	if err != nil {
		writeInvitationError(w, err)
		return
	}

	if invitations == nil {
		invitations = []models.GroupInvitation{}
	}

	writeGroupJSON(w, http.StatusOK, invitations)
}

func (h invitationHandler) invite(w http.ResponseWriter, r *http.Request) {
	identity, ok := identityFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var input struct {
		Username string `json:"username"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1024)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&input); err != nil {
		writeGroupRequestError(w, "Expected a JSON object containing only username.")
		return
	}

	if err := decoder.Decode(new(any)); err != io.EOF {
		writeGroupRequestError(w, "Expected a single JSON object.")
		return
	}

	username, err := models.NormalizeUsername(input.Username)
	if err != nil {
		writeInvitationError(w, err)
		return
	}

	if err := h.invitations.Invite(r.Context(), identity, r.PathValue("groupID"), username); err != nil {
		writeInvitationError(w, err)
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusNoContent)
}

func (h invitationHandler) accept(w http.ResponseWriter, r *http.Request) {
	h.respond(w, r, true)
}

func (h invitationHandler) decline(w http.ResponseWriter, r *http.Request) {
	h.respond(w, r, false)
}

func (h invitationHandler) respond(w http.ResponseWriter, r *http.Request, accept bool) {
	uid, ok := UIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := h.invitations.Respond(r.Context(), uid, r.PathValue("invitationID"), accept); err != nil {
		writeInvitationError(w, err)
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusNoContent)
}
