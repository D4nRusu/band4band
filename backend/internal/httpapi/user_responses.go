package httpapi

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/D4nRusu/band4band/backend/internal/models"
)

func writeUserError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, models.ErrInvalidUsername):
		writeProfileError(w, "invalid_username", models.ErrInvalidUsername.Error(), http.StatusBadRequest)

	case errors.Is(err, models.ErrUsernameTaken):
		writeProfileError(w, "username_taken", models.ErrUsernameTaken.Error(), http.StatusConflict)

	case errors.Is(err, models.ErrProfileComplete):
		writeProfileError(w, "profile_complete", models.ErrProfileComplete.Error(), http.StatusConflict)

	default:
		// Do not log SDK errors or request data, which may include private information.
		log.Print("User profile operation failed")
		writeProfileError(w, "profile_unavailable", "Could not load or save your profile. Please retry.", http.StatusInternalServerError)
	}
}

func writeUser(w http.ResponseWriter, user models.User) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	if err := json.NewEncoder(w).Encode(user); err != nil {
		return
	}
}

func writeProfileError(w http.ResponseWriter, code, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(map[string]string{"code": code, "message": message}); err != nil {
		return
	}
}
