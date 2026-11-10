package httpapi_test

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"firebase.google.com/go/v4/auth"
	"github.com/D4nRusu/band4band/backend/internal/httpapi"
	"github.com/D4nRusu/band4band/backend/internal/models"
)

func TestUpdateGroup(t *testing.T) {
	for _, tc := range []struct {
		name, body    string
		authenticated bool
		storeError    error
		status        int
	}{
		{"member", `{"name":" Trip ","description":" Shared "}`, true, nil, 200},
		{"non-member", `{"name":"Trip"}`, true, models.ErrGroupAccessForbidden, 403},
		{"missing group", `{"name":"Trip"}`, true, models.ErrGroupNotFound, 404},
		{"unauthenticated", `{"name":"Trip"}`, false, nil, 401},
		{"empty name", `{"name":" "}`, true, nil, 400},
		{"injected owner", `{"name":"Trip","createdBy":"attacker"}`, true, nil, 400},
		{"oversized description", `{"name":"Trip","description":"` + strings.Repeat("a", 1001) + `"}`, true, nil, 400},
		{"trailing JSON", `{"name":"Trip"} {}`, true, nil, 400},
	} {
		t.Run(tc.name, func(t *testing.T) {
			verifier := verifierFunc(func(context.Context, string) (*auth.Token, error) { return &auth.Token{UID: "member"}, nil })
			called := false
			store := groupStoreStub{update: func(_ context.Context, uid, id string, input models.GroupInput) (models.Group, error) {
				called = true
				if uid != "member" || id != "group-id" || input.Name != "Trip" {
					t.Fatalf("uid=%s id=%s input=%+v", uid, id, input)
				}

				if tc.name == "member" && input.Description != "Shared" {
					t.Fatalf("description=%q", input.Description)
				}

				return models.Group{ID: id, Name: input.Name, Description: input.Description}, tc.storeError
			}}

			r := httptest.NewRequest("PUT", "/api/groups/group-id?uid=attacker", strings.NewReader(tc.body))
			if tc.authenticated {
				r.Header.Set("Authorization", "Bearer token")
			}

			w := httptest.NewRecorder()
			httpapi.NewRouter(verifier, nil, store, nil, nil).ServeHTTP(w, r)
			if w.Code != tc.status {
				t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
			}

			if (tc.status == 400 || tc.status == 401) && called {
				t.Fatal("invalid request reached storage")
			}
		})
	}
}
