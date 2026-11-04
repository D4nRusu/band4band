package httpapi

import (
	"encoding/json"
	"net/http"
)

// NewRouter registers public health and authenticated profile/group endpoints.
func NewRouter(verifier TokenVerifier, users UserStore, groups GroupStore) http.Handler {
	mux := http.NewServeMux()
	handler := userHandler{users: users}
	groupsHandler := groupHandler{groups: groups}

	mux.HandleFunc("GET /api/health", health)
	mux.Handle("GET /api/users/me", Authenticate(verifier, http.HandlerFunc(handler.currentUser)))
	mux.Handle("PUT /api/users/me/username", Authenticate(verifier, http.HandlerFunc(handler.setUsername)))

	mux.Handle("GET /api/groups", Authenticate(verifier, http.HandlerFunc(groupsHandler.listGroups)))
	mux.Handle("POST /api/groups", Authenticate(verifier, http.HandlerFunc(groupsHandler.createGroup)))
	mux.Handle("DELETE /api/groups/{groupID}", Authenticate(verifier, http.HandlerFunc(groupsHandler.deleteGroup)))

	return mux
}

func health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
		return
	}
}
