package httpapi

import (
	"encoding/json"
	"net/http"
)

// NewRouter registers the public health endpoint and authenticated profile endpoints.
func NewRouter(verifier TokenVerifier, users UserStore) http.Handler {
	mux := http.NewServeMux()
	handler := userHandler{users: users}

	mux.HandleFunc("GET /api/health", health)
	mux.Handle("GET /api/users/me", Authenticate(verifier, http.HandlerFunc(handler.currentUser)))
	mux.Handle("PUT /api/users/me/username", Authenticate(verifier, http.HandlerFunc(handler.setUsername)))

	return mux
}

func health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
		return
	}
}
