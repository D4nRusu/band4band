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

func TestGroupDetailsEndpoints(t *testing.T) {
	for _, suffix := range []string{"", "/members"} {
		for _, tc := range []struct {
			name          string
			authenticated bool
			storeError    error
			status        int
		}{
			{"member", true, nil, 200},
			{"non-member", true, models.ErrGroupAccessForbidden, 403},
			{"missing", true, models.ErrGroupNotFound, 404},
			{"invalid", true, models.ErrInvalidGroupID, 400},
			{"unauthenticated", false, nil, 401},
		} {
			t.Run(suffix+"/"+tc.name, func(t *testing.T) {
				verifier := verifierFunc(func(context.Context, string) (*auth.Token, error) {
					return &auth.Token{UID: "verified-user"}, nil
				})

				called := false
				checkCaller := func(uid, id string) {
					called = true
					if uid != "verified-user" || id != "group-id" {
						t.Fatalf("uid=%q groupID=%q", uid, id)
					}
				}

				store := groupStoreStub{
					details: func(_ context.Context, uid, id string) (models.Group, error) {
						checkCaller(uid, id)
						return models.Group{ID: id, Name: "Trip", PendingInvitations: map[string]string{"private": "invitation"}}, tc.storeError
					},
					members: func(_ context.Context, uid, id string) ([]models.GroupMember, error) {
						checkCaller(uid, id)
						return []models.GroupMember{{UID: "member", Name: "Alice", Email: "alice@example.com"}}, tc.storeError
					},
				}

				request := httptest.NewRequest("GET", "/api/groups/group-id"+suffix+"?uid=spoofed", nil)
				if tc.authenticated {
					request.Header.Set("Authorization", "Bearer token")
				}

				response := httptest.NewRecorder()
				httpapi.NewRouter(verifier, nil, store, nil).ServeHTTP(response, request)

				if response.Code != tc.status || called != tc.authenticated {
					t.Fatalf("status=%d called=%v body=%s", response.Code, called, response.Body.String())
				}

				if tc.status == 200 {
					if response.Header().Get("Cache-Control") != "no-store" {
						t.Fatal("missing no-store header")
					}

					if strings.Contains(response.Body.String(), "private") {
						t.Fatal("internal invitation data exposed")
					}

					if suffix == "/members" && !strings.Contains(response.Body.String(), "alice@example.com") {
						t.Fatal("member profile missing")
					}
				}
			})
		}
	}
}
