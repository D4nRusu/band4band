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

func TestCategoryEndpoints(t *testing.T) {
	for _, tc := range []struct {
		name, method, body string
		authenticated      bool
		storeError         error
		status             int
	}{
		{"list", "GET", "", true, nil, 200},
		{"create", "POST", `{"name":" Pets ","icon":"other"}`, true, nil, 201},
		{"duplicate", "POST", `{"name":"Pets","icon":"other"}`, true, models.ErrCategoryExists, 409},
		{"non-member list", "GET", "", true, models.ErrGroupAccessForbidden, 403},
		{"non-member create", "POST", `{"name":"Pets","icon":"other"}`, true, models.ErrGroupAccessForbidden, 403},
		{"missing group", "GET", "", true, models.ErrGroupNotFound, 404},
		{"unauthenticated list", "GET", "", false, nil, 401},
		{"unauthenticated create", "POST", `{"name":"Pets","icon":"other"}`, false, nil, 401},
		{"empty", "POST", `{"name":" ","icon":"other"}`, true, nil, 400},
		{"invalid icon", "POST", `{"name":"Pets","icon":"invalid"}`, true, nil, 400},
		{"injected defaults", "POST", `{"name":"Pets","icon":"other","isDefault":true}`, true, nil, 400},
		{"trailing JSON", "POST", `{"name":"Pets","icon":"other"} {}`, true, nil, 400},
	} {
		t.Run(tc.name, func(t *testing.T) {
			called := false
			check := func(uid, id string) {
				called = true
				if uid != "member" || id != "group-id" {
					t.Fatalf("uid=%s id=%s", uid, id)
				}
			}
			store := groupStoreStub{
				categories: func(_ context.Context, uid, id string) ([]models.Category, error) {
					check(uid, id)
					return models.DefaultCategories(), tc.storeError
				},
				addCategory: func(_ context.Context, uid, id string, input models.CategoryInput) (models.Category, error) {
					check(uid, id)
					if input.Name != "Pets" {
						t.Fatalf("input=%+v", input)
					}
					return models.Category{ID: "custom", Name: input.Name, Icon: input.Icon}, tc.storeError
				},
			}
			verifier := verifierFunc(func(context.Context, string) (*auth.Token, error) { return &auth.Token{UID: "member"}, nil })
			r := httptest.NewRequest(tc.method, "/api/groups/group-id/categories?uid=outsider", strings.NewReader(tc.body))
			if tc.authenticated {
				r.Header.Set("Authorization", "Bearer token")
			}

			w := httptest.NewRecorder()
			httpapi.NewRouter(verifier, nil, store, nil).ServeHTTP(w, r)
			if w.Code != tc.status {
				t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
			}

			if (tc.status == 400 || tc.status == 401) && called {
				t.Fatal("invalid request reached storage")
			}
		})
	}
}

func TestDeleteCategoryEndpoint(t *testing.T) {
	for _, tc := range []struct {
		name          string
		authenticated bool
		err           error
		status        int
	}{
		{"member", true, nil, 204},
		{"default", true, models.ErrDefaultCategory, 400},
		{"invalid ID", true, models.ErrInvalidCategoryID, 400},
		{"outsider", true, models.ErrGroupAccessForbidden, 403},
		{"missing group", true, models.ErrGroupNotFound, 404},
		{"unauthenticated", false, nil, 401},
	} {
		t.Run(tc.name, func(t *testing.T) {
			called := false
			store := groupStoreStub{deleteCategory: func(_ context.Context, uid, groupID, categoryID string) error {
				called = true
				if uid != "member" || groupID != "group-id" || categoryID != "custom-id" {
					t.Fatalf("uid=%s group=%s category=%s", uid, groupID, categoryID)
				}

				return tc.err
			}}
			verifier := verifierFunc(func(context.Context, string) (*auth.Token, error) { return &auth.Token{UID: "member"}, nil })
			r := httptest.NewRequest("DELETE", "/api/groups/group-id/categories/custom-id?uid=attacker", nil)
			if tc.authenticated {
				r.Header.Set("Authorization", "Bearer token")
			}

			w := httptest.NewRecorder()
			httpapi.NewRouter(verifier, nil, store, nil).ServeHTTP(w, r)
			if w.Code != tc.status || called != tc.authenticated {
				t.Fatalf("status=%d called=%v", w.Code, called)
			}

			if tc.status == 204 && w.Body.Len() != 0 {
				t.Fatal("204 has a body")
			}
		})
	}
}
