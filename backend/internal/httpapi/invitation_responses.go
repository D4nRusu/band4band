package httpapi

import (
	"errors"
	"net/http"

	"github.com/D4nRusu/band4band/backend/internal/models"
)

func writeInvitationError(w http.ResponseWriter, err error) {
	status := http.StatusBadRequest
	code := "invalid_invitation"
	message := ""

	switch {
	case errors.Is(err, models.ErrInvalidUsername):
		message = models.ErrInvalidUsername.Error()

	case errors.Is(err, models.ErrInvalidInvitationID):
		message = models.ErrInvalidInvitationID.Error()

	case errors.Is(err, models.ErrInviteUserNotFound):
		status, code, message = http.StatusNotFound, "user_not_found", models.ErrInviteUserNotFound.Error()

	case errors.Is(err, models.ErrAlreadyGroupMember):
		status, code, message = http.StatusConflict, "already_member", models.ErrAlreadyGroupMember.Error()

	case errors.Is(err, models.ErrInvitationNotFound):
		status, code, message = http.StatusNotFound, "invitation_not_found", models.ErrInvitationNotFound.Error()

	case errors.Is(err, models.ErrInvitationForbidden):
		status, code, message = http.StatusForbidden, "invitation_forbidden", models.ErrInvitationForbidden.Error()

	default:
		writeGroupError(w, err)
		return
	}

	writeGroupJSON(w, status, map[string]string{"code": code, "message": message})
}
