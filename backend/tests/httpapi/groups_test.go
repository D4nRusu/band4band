package httpapi_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"firebase.google.com/go/v4/auth"

	"github.com/D4nRusu/band4band/backend/internal/httpapi"
	"github.com/D4nRusu/band4band/backend/internal/models"
)

type groupStoreStub struct {
	leave  func(context.Context, string, string) error
	remove func(context.Context, string, string) error
	list   func(context.Context, string) ([]models.Group, error)
	create func(context.Context, string, models.GroupInput) (models.Group, error)
}

func (s groupStoreStub) ListForUser(ctx context.Context, uid string) ([]models.Group, error) {
	return s.list(ctx, uid)
}

func (s groupStoreStub) Create(ctx context.Context, uid string, input models.GroupInput) (models.Group, error) {
	return s.create(ctx, uid, input)
}

func (s groupStoreStub) Delete(ctx context.Context, uid, groupID string) error {
	return s.remove(ctx, uid, groupID)
}

func (s groupStoreStub) Leave(ctx context.Context, uid, groupID string) error {
	return s.leave(ctx, uid, groupID)
}

func TestGroupsRequireAuthentication(t *testing.T) {
	for _, method := range []string{"GET", "POST"} {
		t.Run(method, func(t *testing.T) {
			verifier := verifierFunc(func(context.Context, string) (*auth.Token, error) {
				t.Fatal("verifier called without a token")
				return nil, nil
			})

			w := httptest.NewRecorder()
			httpapi.NewRouter(verifier, nil, groupStoreStub{}, nil).ServeHTTP(w, httptest.NewRequest(method, "/api/groups", nil))

			if w.Code != 401 {
				t.Fatalf("status = %d", w.Code)
			}
		})
	}
}

func TestGroupListUsesAuthenticatedMembership(t *testing.T) {
	verifier := verifierFunc(func(context.Context, string) (*auth.Token, error) {
		return &auth.Token{UID: "verified-user"}, nil
	})

	store := groupStoreStub{list: func(_ context.Context, uid string) ([]models.Group, error) {
		if uid != "verified-user" {
			t.Fatalf("queried groups for %q", uid)
		}

		return nil, nil
	}}

	r := httptest.NewRequest("GET", "/api/groups?uid=another-user", nil)
	r.Header.Set("Authorization", "Bearer token")
	w := httptest.NewRecorder()
	httpapi.NewRouter(verifier, nil, store, nil).ServeHTTP(w, r)

	if w.Code != 200 || strings.TrimSpace(w.Body.String()) != "[]" {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestGroupCreationValidatesClientInput(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		status     int
	}{
		{"valid", `{"name":"  Trip  ","description":"  Shared costs  "}`, 201},
		{"empty name", `{"name":"   "}`, 400},
		{"oversized name", `{"name":"` + strings.Repeat("a", 101) + `"}`, 400},
		{"oversized description", `{"name":"Trip","description":"` + strings.Repeat("a", 1001) + `"}`, 400},
		{"injected membership", `{"name":"Trip","memberUIDs":["other"]}`, 400},
		{"injected creator", `{"name":"Trip","createdBy":"other"}`, 400},
		{"trailing JSON", `{"name":"Trip"} {}`, 400},
		{"null", `null`, 400},
	} {
		t.Run(tc.name, func(t *testing.T) {
			verifier := verifierFunc(func(context.Context, string) (*auth.Token, error) {
				return &auth.Token{UID: "creator"}, nil
			})

			called := false
			store := groupStoreStub{create: func(_ context.Context, uid string, input models.GroupInput) (models.Group, error) {
				called = true

				if uid != "creator" || input.Name != "Trip" || input.Description != "Shared costs" {
					t.Fatalf("unexpected creator/input: %q, %+v", uid, input)
				}

				return models.Group{ID: "group-id", Name: input.Name, CreatedBy: uid, MemberUIDs: []string{uid}}, nil
			}}

			r := httptest.NewRequest("POST", "/api/groups", strings.NewReader(tc.body))
			r.Header.Set("Authorization", "Bearer token")
			w := httptest.NewRecorder()
			httpapi.NewRouter(verifier, nil, store, nil).ServeHTTP(w, r)

			if w.Code != tc.status {
				t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
			}

			if tc.status == 400 && called {
				t.Fatal("invalid request reached the group store")
			}

			if tc.status == 201 {
				var group models.Group
				if err := json.Unmarshal(w.Body.Bytes(), &group); err != nil {
					t.Fatal(err)
				}

				if group.ID != "group-id" || len(group.MemberUIDs) != 1 || group.MemberUIDs[0] != "creator" {
					t.Fatalf("unexpected group: %+v", group)
				}
			}
		})
	}
}

func TestDeleteGroupEndpoint(t *testing.T) {
	for _, tc := range []struct {
		name          string
		authenticated bool
		err           error
		status        int
	}{
		{"owner", true, nil, 204},
		{"non-owner", true, models.ErrGroupForbidden, 403},
		{"missing", true, models.ErrGroupNotFound, 404},
		{"invalid ID", true, models.ErrInvalidGroupID, 400},
		{"database failure", true, errors.New("private details"), 500},
		{"unauthenticated", false, nil, 401},
	} {
		t.Run(tc.name, func(t *testing.T) {
			verifier := verifierFunc(func(context.Context, string) (*auth.Token, error) {
				return &auth.Token{UID: "verified-owner"}, nil
			})

			called := false
			store := groupStoreStub{remove: func(_ context.Context, uid, groupID string) error {
				called = true

				if uid != "verified-owner" || groupID != "group-id" {
					t.Fatalf("uid=%q groupID=%q", uid, groupID)
				}

				return tc.err
			}}

			r := httptest.NewRequest("DELETE", "/api/groups/group-id?uid=spoofed", nil)
			if tc.authenticated {
				r.Header.Set("Authorization", "Bearer token")
			}

			w := httptest.NewRecorder()
			httpapi.NewRouter(verifier, nil, store, nil).ServeHTTP(w, r)

			if w.Code != tc.status {
				t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
			}

			if called != tc.authenticated {
				t.Fatalf("store called=%v", called)
			}

			if tc.status == 204 && w.Body.Len() != 0 {
				t.Fatal("204 response has a body")
			}

			if strings.Contains(w.Body.String(), "private details") {
				t.Fatal("private error leaked")
			}
		})
	}
}
