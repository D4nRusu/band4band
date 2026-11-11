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

type invitationStoreStub struct {
	list    func(context.Context, string) ([]models.GroupInvitation, error)
	invite  func(context.Context, models.Identity, string, string) error
	respond func(context.Context, string, string, bool) error
}

func (s invitationStoreStub) ListForUser(ctx context.Context, uid string) ([]models.GroupInvitation, error) {
	return s.list(ctx, uid)
}

func (s invitationStoreStub) Invite(ctx context.Context, identity models.Identity, groupID, username string) error {
	return s.invite(ctx, identity, groupID, username)
}

func (s invitationStoreStub) Respond(ctx context.Context, uid, id string, accept bool) error {
	return s.respond(ctx, uid, id, accept)
}

func TestInvitationRoutesRequireAuthentication(t *testing.T) {
	for _, route := range []struct{ method, path string }{
		{"GET", "/api/invitations"},
		{"POST", "/api/groups/group-id/invitations"},
		{"POST", "/api/invitations/invite-id/accept"},
		{"POST", "/api/invitations/invite-id/decline"},
		{"POST", "/api/groups/group-id/leave"},
	} {
		t.Run(route.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			httpapi.NewRouter(nil, nil, nil, nil, nil).ServeHTTP(w, httptest.NewRequest(route.method, route.path, nil))

			if w.Code != 401 {
				t.Fatalf("status=%d", w.Code)
			}
		})
	}
}

func TestInviteUsesVerifiedSenderAndNormalizesUsername(t *testing.T) {
	verifier := verifierFunc(func(context.Context, string) (*auth.Token, error) {
		return &auth.Token{UID: "member", Claims: map[string]any{"name": "Verified Name"}}, nil
	})

	for _, tc := range []struct {
		name, body string
		err        error
		status     int
	}{
		{"success", `{"username":"ALICE"}`, nil, 204},
		{"not a member", `{"username":"ALICE"}`, models.ErrGroupMembershipRequired, 403},
		{"already member", `{"username":"ALICE"}`, models.ErrAlreadyGroupMember, 409},
		{"unknown user", `{"username":"ALICE"}`, models.ErrInviteUserNotFound, 404},
		{"spoof sender", `{"username":"ALICE","inviterUID":"other"}`, nil, 400},
		{"invalid username", `{"username":"a/b"}`, nil, 400},
	} {
		t.Run(tc.name, func(t *testing.T) {
			called := false
			store := invitationStoreStub{invite: func(_ context.Context, sender models.Identity, groupID, username string) error {
				called = true
				if sender.UID != "member" || sender.Name != "Verified Name" || groupID != "group-id" || username != "alice" {
					t.Fatalf("unexpected invite: %+v %s %s", sender, groupID, username)
				}

				return tc.err
			}}

			r := httptest.NewRequest("POST", "/api/groups/group-id/invitations", strings.NewReader(tc.body))
			r.Header.Set("Authorization", "Bearer token")
			w := httptest.NewRecorder()
			httpapi.NewRouter(verifier, nil, nil, store, nil).ServeHTTP(w, r)

			if w.Code != tc.status || (tc.status == 400 && called) {
				t.Fatalf("status=%d storeCalled=%v body=%s", w.Code, called, w.Body.String())
			}
		})
	}
}

func TestInvitationResponsesUseAuthenticatedRecipient(t *testing.T) {
	verifier := verifierFunc(func(context.Context, string) (*auth.Token, error) {
		return &auth.Token{UID: "recipient"}, nil
	})

	for _, action := range []string{"accept", "decline"} {
		t.Run(action, func(t *testing.T) {
			store := invitationStoreStub{respond: func(_ context.Context, uid, id string, accept bool) error {
				if uid != "recipient" || id != "invite-id" || accept != (action == "accept") {
					t.Fatalf("unexpected response: %s %s %v", uid, id, accept)
				}

				return models.ErrInvitationForbidden
			}}

			r := httptest.NewRequest("POST", "/api/invitations/invite-id/"+action+"?uid=other", nil)
			r.Header.Set("Authorization", "Bearer token")
			w := httptest.NewRecorder()
			httpapi.NewRouter(verifier, nil, nil, store, nil).ServeHTTP(w, r)

			if w.Code != 403 {
				t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
			}
		})
	}
}

func TestInvitationInboxUsesAuthenticatedRecipient(t *testing.T) {
	verifier := verifierFunc(func(context.Context, string) (*auth.Token, error) {
		return &auth.Token{UID: "recipient"}, nil
	})
	store := invitationStoreStub{list: func(_ context.Context, uid string) ([]models.GroupInvitation, error) {
		if uid != "recipient" {
			t.Fatalf("unexpected UID %q", uid)
		}

		return nil, nil
	}}

	r := httptest.NewRequest("GET", "/api/invitations?uid=other", nil)
	r.Header.Set("Authorization", "Bearer token")
	w := httptest.NewRecorder()
	httpapi.NewRouter(verifier, nil, nil, store, nil).ServeHTTP(w, r)

	if w.Code != 200 || strings.TrimSpace(w.Body.String()) != "[]" {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestOwnerCannotUseLeaveEndpoint(t *testing.T) {
	verifier := verifierFunc(func(context.Context, string) (*auth.Token, error) {
		return &auth.Token{UID: "owner"}, nil
	})
	store := groupStoreStub{leave: func(_ context.Context, uid, groupID string) error {
		if uid != "owner" || groupID != "group-id" {
			t.Fatalf("unexpected leave: %s %s", uid, groupID)
		}

		return models.ErrOwnerCannotLeave
	}}

	r := httptest.NewRequest("POST", "/api/groups/group-id/leave", nil)
	r.Header.Set("Authorization", "Bearer token")
	w := httptest.NewRecorder()
	httpapi.NewRouter(verifier, nil, store, nil, nil).ServeHTTP(w, r)

	if w.Code != 403 {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}
