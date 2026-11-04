package httpapi_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"firebase.google.com/go/v4/auth"

	"github.com/D4nRusu/band4band/backend/internal/httpapi"
	"github.com/D4nRusu/band4band/backend/internal/models"
)

type userStoreStub struct {
	get func(context.Context, models.Identity) (models.User, error)
	set func(context.Context, string, string) (models.User, error)
}

func (s userStoreStub) GetOrCreate(ctx context.Context, identity models.Identity) (models.User, error) {
	return s.get(ctx, identity)
}

func (s userStoreStub) SetUsername(ctx context.Context, uid, username string) (models.User, error) {
	return s.set(ctx, uid, username)
}

func TestCurrentProfileUsesVerifiedClaims(t *testing.T) {
	want := models.User{
		UID: "verified", Email: "trusted@example.test", Name: "Original Name",
		Username: "saved_handle", PhotoURL: "https://example.test/photo",
		CreatedAt: time.Date(2026, 11, 4, 12, 0, 0, 0, time.UTC), ProfileComplete: true,
	}
	verifier := verifierFunc(func(context.Context, string) (*auth.Token, error) {
		return &auth.Token{UID: "verified", Claims: map[string]any{
			"email": "trusted@example.test", "name": "Google Name", "picture": "https://example.test/google",
		}}, nil
	})
	store := userStoreStub{get: func(_ context.Context, identity models.Identity) (models.User, error) {
		if identity != (models.Identity{UID: "verified", Email: "trusted@example.test", Name: "Google Name", PhotoURL: "https://example.test/google"}) {
			t.Fatalf("unexpected identity: %+v", identity)
		}
		return want, nil
	}}
	r := httptest.NewRequest("GET", "/api/users/me?uid=attacker", nil)
	r.Header.Set("Authorization", "Bearer token")
	w := httptest.NewRecorder()
	httpapi.NewRouter(verifier, store).ServeHTTP(w, r)
	var got models.User
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if w.Code != 200 || got != want || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("status=%d profile=%+v", w.Code, got)
	}
}

func TestUsernameEndpoint(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		err        error
		status     int
		code       string
	}{
		{"valid normalized", `{"username":"Alice_123"}`, nil, 200, ""},
		{"taken", `{"username":"Alice_123"}`, models.ErrUsernameTaken, 409, "username_taken"},
		{"already complete", `{"username":"Alice_123"}`, models.ErrProfileComplete, 409, "profile_complete"},
		{"database failure", `{"username":"Alice_123"}`, errors.New("private database details"), 500, "profile_unavailable"},
		{"too short", `{"username":"ab"}`, nil, 400, "invalid_username"},
		{"spaces", `{"username":"alice bob"}`, nil, 400, "invalid_username"},
		{"missing", `{}`, nil, 400, "invalid_username"},
		{"null", `null`, nil, 400, "invalid_username"},
		{"wrong type", `{"username":123}`, nil, 400, "invalid_request"},
		{"spoof UID", `{"username":"alice","uid":"other"}`, nil, 400, "invalid_request"},
		{"spoof email", `{"username":"alice","email":"other"}`, nil, 400, "invalid_request"},
		{"trailing JSON", `{"username":"alice"} {}`, nil, 400, "invalid_request"},
		{"oversize", `{"username":"` + strings.Repeat("a", 1100) + `"}`, nil, 400, "invalid_request"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			called := false
			store := userStoreStub{
				get: func(context.Context, models.Identity) (models.User, error) { return models.User{}, nil },
				set: func(_ context.Context, uid, username string) (models.User, error) {
					called = true
					if uid != "verified" || username != "alice_123" {
						t.Fatalf("uid=%s username=%s", uid, username)
					}
					return models.User{UID: uid, Username: username, ProfileComplete: true}, tc.err
				},
			}
			verifier := verifierFunc(func(context.Context, string) (*auth.Token, error) { return &auth.Token{UID: "verified"}, nil })
			r := httptest.NewRequest("PUT", "/api/users/me/username", strings.NewReader(tc.body))
			r.Header.Set("Authorization", "Bearer token")
			w := httptest.NewRecorder()
			httpapi.NewRouter(verifier, store).ServeHTTP(w, r)
			if w.Code != tc.status {
				t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
			}
			if tc.status == 400 && called {
				t.Fatal("invalid username reached store")
			}
			if tc.code != "" {
				var body map[string]string
				if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
					t.Fatal(err)
				}
				if body["code"] != tc.code || strings.Contains(w.Body.String(), "private database") {
					t.Fatalf("unexpected error: %s", w.Body.String())
				}
			}
		})
	}
}

func TestProfilesRequireAuthentication(t *testing.T) {
	for _, tc := range []struct{ method, path string }{{"GET", "/api/users/me"}, {"PUT", "/api/users/me/username"}} {
		t.Run(tc.method, func(t *testing.T) {
			verifier := verifierFunc(func(context.Context, string) (*auth.Token, error) {
				t.Fatal("verifier should not be called")
				return nil, nil
			})
			w := httptest.NewRecorder()
			httpapi.NewRouter(verifier, userStoreStub{}).ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, nil))
			if w.Code != http.StatusUnauthorized {
				t.Fatalf("status=%d", w.Code)
			}
		})
	}
}

func TestProfileReadFailureIsNotAnEmptyProfile(t *testing.T) {
	verifier := verifierFunc(func(context.Context, string) (*auth.Token, error) { return &auth.Token{UID: "verified"}, nil })
	store := userStoreStub{get: func(context.Context, models.Identity) (models.User, error) {
		return models.User{}, errors.New("database unavailable")
	}}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/users/me", nil)
	r.Header.Set("Authorization", "Bearer token")
	httpapi.NewRouter(verifier, store).ServeHTTP(w, r)
	if w.Code != 500 {
		t.Fatalf("status=%d", w.Code)
	}
}
