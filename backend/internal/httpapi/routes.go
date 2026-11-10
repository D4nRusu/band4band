package httpapi

import (
	"encoding/json"
	"net/http"
)

// NewRouter registers public health and authenticated profile/group endpoints.
func NewRouter(verifier TokenVerifier, users UserStore, groups GroupStore, invitations InvitationStore) http.Handler {
	mux := http.NewServeMux()
	handler := userHandler{users: users}
	groupsHandler := groupHandler{groups: groups}
	invitesHandler := invitationHandler{invitations: invitations}

	mux.HandleFunc("GET /api/health", health)
	mux.Handle("GET /api/users/me", Authenticate(verifier, http.HandlerFunc(handler.currentUser)))
	mux.Handle("PUT /api/users/me/username", Authenticate(verifier, http.HandlerFunc(handler.setUsername)))

	mux.Handle("GET /api/groups", Authenticate(verifier, http.HandlerFunc(groupsHandler.listGroups)))
	mux.Handle("POST /api/groups", Authenticate(verifier, http.HandlerFunc(groupsHandler.createGroup)))
	mux.Handle("GET /api/groups/{groupID}", Authenticate(verifier, http.HandlerFunc(groupsHandler.details)))
	mux.Handle("GET /api/groups/{groupID}/categories", Authenticate(verifier, http.HandlerFunc(groupsHandler.categories)))
	mux.Handle("DELETE /api/groups/{groupID}/categories/{categoryID}", Authenticate(verifier, http.HandlerFunc(groupsHandler.deleteCategory)))
	mux.Handle("POST /api/groups/{groupID}/categories", Authenticate(verifier, http.HandlerFunc(groupsHandler.addCategory)))
	mux.Handle("GET /api/groups/{groupID}/expenses", Authenticate(verifier, http.HandlerFunc(groupsHandler.expenses)))
	mux.Handle("DELETE /api/groups/{groupID}/expenses/{expenseID}", Authenticate(verifier, http.HandlerFunc(groupsHandler.deleteExpense)))
	mux.Handle("POST /api/groups/{groupID}/expenses", Authenticate(verifier, http.HandlerFunc(groupsHandler.createExpense)))
	mux.Handle("GET /api/groups/{groupID}/lists", Authenticate(verifier, http.HandlerFunc(groupsHandler.lists)))
	mux.Handle("POST /api/groups/{groupID}/lists", Authenticate(verifier, http.HandlerFunc(groupsHandler.createList)))
	mux.Handle("DELETE /api/groups/{groupID}/lists/{listID}", Authenticate(verifier, http.HandlerFunc(groupsHandler.deleteList)))
	mux.Handle("GET /api/groups/{groupID}/lists/{listID}/items", Authenticate(verifier, http.HandlerFunc(groupsHandler.listItems)))
	mux.Handle("POST /api/groups/{groupID}/lists/{listID}/items", Authenticate(verifier, http.HandlerFunc(groupsHandler.addListItem)))
	mux.Handle("POST /api/groups/{groupID}/lists/{listID}/clear-done", Authenticate(verifier, http.HandlerFunc(groupsHandler.clearDoneListItems)))
	mux.Handle("PATCH /api/groups/{groupID}/lists/{listID}/items/{itemID}", Authenticate(verifier, http.HandlerFunc(groupsHandler.setListItemDone)))
	mux.Handle("DELETE /api/groups/{groupID}/lists/{listID}/items/{itemID}", Authenticate(verifier, http.HandlerFunc(groupsHandler.deleteListItem)))
	mux.Handle("GET /api/groups/{groupID}/stats", Authenticate(verifier, http.HandlerFunc(groupsHandler.stats)))
	mux.Handle("GET /api/groups/{groupID}/balances", Authenticate(verifier, http.HandlerFunc(groupsHandler.memberBalances)))
	mux.Handle("GET /api/groups/{groupID}/settlements", Authenticate(verifier, http.HandlerFunc(groupsHandler.settlements)))
	mux.Handle("POST /api/groups/{groupID}/settlements", Authenticate(verifier, http.HandlerFunc(groupsHandler.settleUp)))
	mux.Handle("GET /api/groups/{groupID}/users/{userID}/owed", Authenticate(verifier, http.HandlerFunc(groupsHandler.userOwedSummary)))
	mux.Handle("GET /api/groups/{groupID}/users/{userID}/receivable", Authenticate(verifier, http.HandlerFunc(groupsHandler.userReceivableSummary)))
	mux.Handle("GET /api/groups/{groupID}/members", Authenticate(verifier, http.HandlerFunc(groupsHandler.members)))
	mux.Handle("PUT /api/groups/{groupID}", Authenticate(verifier, http.HandlerFunc(groupsHandler.updateGroup)))
	mux.Handle("DELETE /api/groups/{groupID}", Authenticate(verifier, http.HandlerFunc(groupsHandler.deleteGroup)))
	mux.Handle("POST /api/groups/{groupID}/leave", Authenticate(verifier, http.HandlerFunc(groupsHandler.leaveGroup)))

	mux.Handle("GET /api/invitations", Authenticate(verifier, http.HandlerFunc(invitesHandler.list)))
	mux.Handle("POST /api/groups/{groupID}/invitations", Authenticate(verifier, http.HandlerFunc(invitesHandler.invite)))
	mux.Handle("POST /api/invitations/{invitationID}/accept", Authenticate(verifier, http.HandlerFunc(invitesHandler.accept)))
	mux.Handle("POST /api/invitations/{invitationID}/decline", Authenticate(verifier, http.HandlerFunc(invitesHandler.decline)))

	return mux
}

func health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
		return
	}
}
