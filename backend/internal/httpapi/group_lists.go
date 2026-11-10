package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/D4nRusu/band4band/backend/internal/models"
)

type listItemDoneInput struct {
	Done *bool `json:"done"`
}

func (h groupHandler) lists(w http.ResponseWriter, r *http.Request) {
	uid, ok := UIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	lists, err := h.groups.Lists(r.Context(), uid, r.PathValue("groupID"))
	if err != nil {
		writeListError(w, err)
		return
	}

	if lists == nil {
		lists = []models.GroupList{}
	}

	writeGroupJSON(w, http.StatusOK, lists)
}

func (h groupHandler) createList(w http.ResponseWriter, r *http.Request) {
	uid, ok := UIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var input models.ListInput
	if !decodeSingleJSON(w, r, &input, "Expected a list name.") {
		return
	}

	list, err := h.groups.CreateList(r.Context(), uid, r.PathValue("groupID"), input)
	if err != nil {
		writeListError(w, err)
		return
	}

	writeGroupJSON(w, http.StatusCreated, list)
}

func (h groupHandler) deleteList(w http.ResponseWriter, r *http.Request) {
	uid, ok := UIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := h.groups.DeleteList(r.Context(), uid, r.PathValue("groupID"), r.PathValue("listID")); err != nil {
		writeListError(w, err)
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusNoContent)
}

func (h groupHandler) listItems(w http.ResponseWriter, r *http.Request) {
	uid, ok := UIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	items, err := h.groups.ListItems(r.Context(), uid, r.PathValue("groupID"), r.PathValue("listID"))
	if err != nil {
		writeListError(w, err)
		return
	}

	if items == nil {
		items = []models.ListItem{}
	}

	writeGroupJSON(w, http.StatusOK, items)
}

func (h groupHandler) addListItem(w http.ResponseWriter, r *http.Request) {
	uid, ok := UIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var input models.ListItemInput
	if !decodeSingleJSON(w, r, &input, "Expected text, and optionally assigneeUID and rotationUIDs.") {
		return
	}

	item, err := h.groups.AddListItem(r.Context(), uid, r.PathValue("groupID"), r.PathValue("listID"), input)
	if err != nil {
		writeListError(w, err)
		return
	}

	writeGroupJSON(w, http.StatusCreated, item)
}

func (h groupHandler) setListItemDone(w http.ResponseWriter, r *http.Request) {
	uid, ok := UIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var input listItemDoneInput
	if !decodeSingleJSON(w, r, &input, "Expected done as true or false.") {
		return
	}

	if input.Done == nil {
		writeGroupRequestError(w, "Expected done as true or false.")
		return
	}

	err := h.groups.SetListItemDone(r.Context(), uid, r.PathValue("groupID"), r.PathValue("listID"), r.PathValue("itemID"), *input.Done)
	if err != nil {
		writeListError(w, err)
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusNoContent)
}

func (h groupHandler) deleteListItem(w http.ResponseWriter, r *http.Request) {
	uid, ok := UIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	err := h.groups.DeleteListItem(r.Context(), uid, r.PathValue("groupID"), r.PathValue("listID"), r.PathValue("itemID"))
	if err != nil {
		writeListError(w, err)
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusNoContent)
}

func (h groupHandler) clearDoneListItems(w http.ResponseWriter, r *http.Request) {
	uid, ok := UIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := h.groups.ClearDoneListItems(r.Context(), uid, r.PathValue("groupID"), r.PathValue("listID")); err != nil {
		writeListError(w, err)
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusNoContent)
}

func decodeSingleJSON(w http.ResponseWriter, r *http.Request, dst any, message string) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 8192)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		writeGroupRequestError(w, message)
		return false
	}

	if err := decoder.Decode(new(any)); err != io.EOF {
		writeGroupRequestError(w, "Expected a single JSON object.")
		return false
	}

	return true
}

// List errors use their own codes so the app can tell a missing list apart from a missing group.
func writeListError(w http.ResponseWriter, err error) {
	for _, validation := range []error{
		models.ErrInvalidListID, models.ErrInvalidListItemID, models.ErrInvalidListName, models.ErrInvalidListItemText,
		models.ErrInvalidListAssignee, models.ErrInvalidListRotation, models.ErrListItemCleared, models.ErrListItemFollowUpDone,
	} {
		if errors.Is(err, validation) {
			writeGroupJSON(w, http.StatusBadRequest, map[string]string{"code": "invalid_list", "message": validation.Error()})
			return
		}
	}

	for _, missing := range []error{models.ErrListNotFound, models.ErrListItemNotFound} {
		if errors.Is(err, missing) {
			writeGroupJSON(w, http.StatusNotFound, map[string]string{"code": "list_not_found", "message": missing.Error()})
			return
		}
	}

	for _, forbidden := range []error{models.ErrListForbidden, models.ErrListItemForbidden} {
		if errors.Is(err, forbidden) {
			writeGroupJSON(w, http.StatusForbidden, map[string]string{"code": "list_forbidden", "message": forbidden.Error()})
			return
		}
	}

	writeGroupError(w, err)
}
