package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"

	"github.com/D4nRusu/band4band/backend/internal/models"
)

type UserStore interface {
	GetOrCreate(context.Context, models.Identity) (models.User, error)
	SetUsername(context.Context, string, string) (models.User, error)
}

// NewRouter registers the public health endpoint and authenticated user endpoint.
func NewRouter(verifier TokenVerifier, users UserStore) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
			return
		}
	})

	mux.Handle("GET /api/users/me", Authenticate(verifier, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		identity, ok := identityFromContext(r.Context())
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		user, err := users.GetOrCreate(r.Context(), identity)
		writeUser(w, user, err)
	})))

	mux.Handle("PUT /api/users/me/username", Authenticate(verifier, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
			writeUser(w, models.User{}, err)
			return
		}

		identity, ok := identityFromContext(r.Context())
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// The setup request is also safe if the initial GET did not finish.
		if _, err := users.GetOrCreate(r.Context(), identity); err != nil {
			writeUser(w, models.User{}, err)
			return
		}

		user, err := users.SetUsername(r.Context(), identity.UID, username)
		writeUser(w, user, err)
	})))
	return mux
}

func writeUser(w http.ResponseWriter, user models.User, err error) {
	if err != nil {
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
		return
	}
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
