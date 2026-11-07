package httpapi

import "net/http"

func (h groupHandler) details(w http.ResponseWriter, r *http.Request) {
	uid, ok := UIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	group, err := h.groups.Details(r.Context(), uid, r.PathValue("groupID"))
	if err != nil {
		writeGroupError(w, err)
		return
	}

	writeGroupJSON(w, http.StatusOK, group)
}

func (h groupHandler) members(w http.ResponseWriter, r *http.Request) {
	uid, ok := UIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	members, err := h.groups.Members(r.Context(), uid, r.PathValue("groupID"))
	if err != nil {
		writeGroupError(w, err)
		return
	}

	writeGroupJSON(w, http.StatusOK, members)
}
