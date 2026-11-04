package httpapi

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/D4nRusu/band4band/backend/internal/models"
)

func writeGroupError(w http.ResponseWriter, err error) {
	if errors.Is(err, models.ErrInvalidGroupName) || errors.Is(err, models.ErrInvalidGroupDescription) {
		writeGroupRequestError(w, err.Error())
		return
	}

	log.Print("Group operation failed")
	writeGroupJSON(w, http.StatusInternalServerError, map[string]string{
		"code":    "groups_unavailable",
		"message": "Could not load or create the group. Please retry.",
	})
}

func writeGroupRequestError(w http.ResponseWriter, message string) {
	writeGroupJSON(w, http.StatusBadRequest, map[string]string{
		"code":    "invalid_group",
		"message": message,
	})
}

func writeGroupJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(value); err != nil {
		return
	}
}
