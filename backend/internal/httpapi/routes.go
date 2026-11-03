package httpapi

import (
	"encoding/json"
	"net/http"
)

// NewRouter registers the public health endpoint and authenticated user endpoint.
func NewRouter(verifier TokenVerifier) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
			return
		}
	})
	mux.Handle("GET /api/users/me", Authenticate(verifier, http.HandlerFunc(currentUser)))
	return mux
}

func currentUser(w http.ResponseWriter, r *http.Request) {
	uid, ok := UIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{"uid": uid}); err != nil {
		return
	}
}
