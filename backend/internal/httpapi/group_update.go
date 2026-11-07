package httpapi

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/D4nRusu/band4band/backend/internal/models"
)

func (h groupHandler) updateGroup(w http.ResponseWriter, r *http.Request) {
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

	group, err := h.groups.Update(r.Context(), uid, r.PathValue("groupID"), input)
	if err != nil {
		writeGroupError(w, err)
		return
	}

	writeGroupJSON(w, http.StatusOK, group)
}
