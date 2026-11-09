package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/D4nRusu/band4band/backend/internal/models"
)

type GroupStore interface {
	Categories(context.Context, string, string) ([]models.Category, error)
	AddCategory(context.Context, string, string, models.CategoryInput) (models.Category, error)
	Update(context.Context, string, string, models.GroupInput) (models.Group, error)
	ListForUser(context.Context, string) ([]models.Group, error)
	Create(context.Context, string, models.GroupInput) (models.Group, error)
	Delete(context.Context, string, string) error
	Leave(context.Context, string, string) error
	Details(context.Context, string, string) (models.Group, error)
	Members(context.Context, string, string) ([]models.GroupMember, error)
}

type groupHandler struct {
	groups GroupStore
}

func (h groupHandler) listGroups(w http.ResponseWriter, r *http.Request) {
	uid, ok := UIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	groups, err := h.groups.ListForUser(r.Context(), uid)
	if err != nil {
		writeGroupError(w, err)
		return
	}

	if groups == nil {
		groups = []models.Group{}
	}

	writeGroupJSON(w, http.StatusOK, groups)
}

func (h groupHandler) createGroup(w http.ResponseWriter, r *http.Request) {
	uid, ok := UIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var input models.GroupInput
	r.Body = http.MaxBytesReader(w, r.Body, 8192)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&input); err != nil {
		writeGroupRequestError(w, "Expected a JSON object containing name and description.")
		return
	}

	if err := decoder.Decode(new(any)); err != io.EOF {
		writeGroupRequestError(w, "Expected a single JSON object.")
		return
	}

	input, err := input.Normalize()
	if err != nil {
		writeGroupError(w, err)
		return
	}

	group, err := h.groups.Create(r.Context(), uid, input)
	if err != nil {
		writeGroupError(w, err)
		return
	}

	writeGroupJSON(w, http.StatusCreated, group)
}

func (h groupHandler) deleteGroup(w http.ResponseWriter, r *http.Request) {
	uid, ok := UIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := h.groups.Delete(r.Context(), uid, r.PathValue("groupID")); err != nil {
		writeGroupError(w, err)
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusNoContent)
}

func (h groupHandler) leaveGroup(w http.ResponseWriter, r *http.Request) {
	uid, ok := UIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := h.groups.Leave(r.Context(), uid, r.PathValue("groupID")); err != nil {
		writeGroupError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
