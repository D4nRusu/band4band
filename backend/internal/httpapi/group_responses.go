package httpapi

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/D4nRusu/band4band/backend/internal/models"
)

func writeGroupError(w http.ResponseWriter, err error) {
	if errors.Is(err, models.ErrGroupAccessForbidden) {
		writeGroupJSON(w, http.StatusForbidden, map[string]string{
			"code":    "group_access_forbidden",
			"message": models.ErrGroupAccessForbidden.Error(),
		})
		return
	}

	if errors.Is(err, models.ErrGroupMembershipRequired) || errors.Is(err, models.ErrOwnerCannotLeave) {
		writeGroupJSON(w, http.StatusForbidden, map[string]string{
			"code":    "group_membership_forbidden",
			"message": err.Error(),
		})
		return
	}

	if errors.Is(err, models.ErrGroupForbidden) {
		writeGroupJSON(w, http.StatusForbidden, map[string]string{
			"code":    "group_forbidden",
			"message": models.ErrGroupForbidden.Error(),
		})
		return
	}

	if errors.Is(err, models.ErrGroupNotFound) {
		writeGroupJSON(w, http.StatusNotFound, map[string]string{
			"code":    "group_not_found",
			"message": models.ErrGroupNotFound.Error(),
		})
		return
	}

	if errors.Is(err, models.ErrInvalidGroupID) {
		writeGroupRequestError(w, err.Error())
		return
	}

	if errors.Is(err, models.ErrInvalidGroupName) || errors.Is(err, models.ErrInvalidGroupDescription) {
		writeGroupRequestError(w, err.Error())
		return
	}

	log.Print("Group operation failed")
	writeGroupJSON(w, http.StatusInternalServerError, map[string]string{
		"code":    "groups_unavailable",
		"message": "Could not complete the group operation. Please retry.",
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
