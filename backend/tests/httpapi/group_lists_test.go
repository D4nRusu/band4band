package httpapi_test

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"firebase.google.com/go/v4/auth"
	"github.com/D4nRusu/band4band/backend/internal/httpapi"
	"github.com/D4nRusu/band4band/backend/internal/models"
)

func serveLists(t *testing.T, store groupStoreStub, method, path, body string, authenticated bool) *httptest.ResponseRecorder {
	t.Helper()
	verifier := verifierFunc(func(context.Context, string) (*auth.Token, error) { return &auth.Token{UID: "member"}, nil })
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	if authenticated {
		r.Header.Set("Authorization", "Bearer token")
	}

	w := httptest.NewRecorder()
	httpapi.NewRouter(verifier, nil, store, nil).ServeHTTP(w, r)
	return w
}

func TestCreateListEndpoint(t *testing.T) {
	for _, tc := range []struct {
		name, body    string
		authenticated bool
		storeError    error
		status        int
	}{
		{"member creates list", `{"name":"Shopping"}`, true, nil, 201},
		{"unauthenticated", `{"name":"Shopping"}`, false, nil, 401},
		{"unknown field", `{"name":"Shopping","owner":"x"}`, true, nil, 400},
		{"invalid name", `{"name":""}`, true, models.ErrInvalidListName, 400},
		{"non-member", `{"name":"Shopping"}`, true, models.ErrGroupAccessForbidden, 403},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := groupStoreStub{createList: func(_ context.Context, uid, groupID string, input models.ListInput) (models.GroupList, error) {
				if uid != "member" || groupID != "group-id" {
					t.Fatalf("uid=%s group=%s", uid, groupID)
				}

				return models.GroupList{ID: "list-id", Name: input.Name, CreatedBy: uid}, tc.storeError
			}}

			w := serveLists(t, store, "POST", "/api/groups/group-id/lists", tc.body, tc.authenticated)
			if w.Code != tc.status {
				t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
			}
		})
	}
}

func TestListItemsEndpointDistinguishesMissingList(t *testing.T) {
	for _, tc := range []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{"items", nil, 200, ""},
		{"missing list", models.ErrListNotFound, 404, "list_not_found"},
		{"missing group", models.ErrGroupNotFound, 404, "group_not_found"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := groupStoreStub{listItems: func(_ context.Context, uid, groupID, listID string) ([]models.ListItem, error) {
				if uid != "member" || groupID != "group-id" || listID != "list-id" {
					t.Fatalf("uid=%s group=%s list=%s", uid, groupID, listID)
				}

				if tc.err != nil {
					return nil, tc.err
				}

				return []models.ListItem{{ID: "item-id", Text: "Milk", RotationUIDs: []string{}}}, nil
			}}

			w := serveLists(t, store, "GET", "/api/groups/group-id/lists/list-id/items", "", true)
			if w.Code != tc.status {
				t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
			}

			if tc.code == "" {
				return
			}

			var body map[string]string
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}

			if body["code"] != tc.code {
				t.Fatalf("body=%v", body)
			}
		})
	}
}

func TestSetListItemDoneEndpoint(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		storeError error
		status     int
		wantDone   bool
	}{
		{"tick", `{"done":true}`, nil, 204, true},
		{"untick", `{"done":false}`, nil, 204, false},
		{"missing done", `{}`, nil, 400, false},
		{"cleared item", `{"done":false}`, models.ErrListItemCleared, 400, false},
		{"follow-up already done", `{"done":false}`, models.ErrListItemFollowUpDone, 400, false},
		{"missing item", `{"done":true}`, models.ErrListItemNotFound, 404, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			called := false
			store := groupStoreStub{setItemDone: func(_ context.Context, uid, groupID, listID, itemID string, done bool) error {
				called = true
				if uid != "member" || groupID != "group-id" || listID != "list-id" || itemID != "item-id" || done != tc.wantDone {
					t.Fatalf("uid=%s group=%s list=%s item=%s done=%v", uid, groupID, listID, itemID, done)
				}

				return tc.storeError
			}}

			w := serveLists(t, store, "PATCH", "/api/groups/group-id/lists/list-id/items/item-id", tc.body, true)
			if w.Code != tc.status {
				t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
			}

			if tc.body == `{}` && called {
				t.Fatal("request without done reached storage")
			}
		})
	}
}

func TestDeleteListItemEndpointForbidden(t *testing.T) {
	store := groupStoreStub{deleteListItem: func(context.Context, string, string, string, string) error {
		return models.ErrListItemForbidden
	}}

	w := serveLists(t, store, "DELETE", "/api/groups/group-id/lists/list-id/items/item-id", "", true)
	if w.Code != 403 || !strings.Contains(w.Body.String(), "list_forbidden") {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestClearDoneEndpoint(t *testing.T) {
	called := false
	store := groupStoreStub{clearDone: func(_ context.Context, uid, groupID, listID string) error {
		called = uid == "member" && groupID == "group-id" && listID == "list-id"
		return nil
	}}

	w := serveLists(t, store, "POST", "/api/groups/group-id/lists/list-id/clear-done", "", true)
	if w.Code != 204 || !called {
		t.Fatalf("status=%d called=%v", w.Code, called)
	}
}
