package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/D4nRusu/band4band/backend/internal/models"
)

func (h groupHandler) categories(w http.ResponseWriter, r *http.Request) {
	uid, ok := UIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	categories, err := h.groups.Categories(r.Context(), uid, r.PathValue("groupID"))
	if err != nil {
		writeGroupError(w, err)
		return
	}

	writeGroupJSON(w, http.StatusOK, categories)
}

func (h groupHandler) addCategory(w http.ResponseWriter, r *http.Request) {
	uid, ok := UIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var input models.CategoryInput
	r.Body = http.MaxBytesReader(w, r.Body, 2048)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		writeGroupRequestError(w, "Expected a category name and icon.")
		return
	}

	if err := decoder.Decode(new(any)); err != io.EOF {
		writeGroupRequestError(w, "Expected a single JSON object.")
		return
	}

	input, err := input.Normalize()
	if err != nil {
		writeGroupRequestError(w, err.Error())
		return
	}

	category, err := h.groups.AddCategory(r.Context(), uid, r.PathValue("groupID"), input)
	if errors.Is(err, models.ErrCategoryExists) {
		writeGroupJSON(w, http.StatusConflict, map[string]string{"code": "category_exists", "message": err.Error()})
		return
	}

	if errors.Is(err, models.ErrInvalidCategoryName) || errors.Is(err, models.ErrInvalidCategoryIcon) {
		writeGroupRequestError(w, err.Error())
		return
	}

	if err != nil {
		writeGroupError(w, err)
		return
	}

	writeGroupJSON(w, http.StatusCreated, category)
}

func (h groupHandler) deleteCategory(w http.ResponseWriter, r *http.Request) {
	uid, ok := UIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	err := h.groups.DeleteCategory(r.Context(), uid, r.PathValue("groupID"), r.PathValue("categoryID"))
	if errors.Is(err, models.ErrDefaultCategory) || errors.Is(err, models.ErrInvalidCategoryID) {
		writeGroupRequestError(w, err.Error())
		return
	}

	if err != nil {
		writeGroupError(w, err)
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusNoContent)
}
