package httpapi_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"firebase.google.com/go/v4/auth"

	"github.com/D4nRusu/band4band/backend/internal/httpapi"
	"github.com/D4nRusu/band4band/backend/internal/models"
)

type verifierFunc func(context.Context, string) (*auth.Token, error)

func (f verifierFunc) VerifyIDToken(ctx context.Context, token string) (*auth.Token, error) {
	return f(ctx, token)
}

func TestAuthenticateRejectsMalformedHeaders(t *testing.T) {
	for _, headers := range [][]string{
		nil, {""}, {"Bearer"}, {"Bearer "}, {"Basic token"},
		{"Bearer token extra"}, {"Bearer token", "Bearer other"},
	} {
		t.Run(associatedHeaderName(headers), func(t *testing.T) {
			verifier := verifierFunc(func(context.Context, string) (*auth.Token, error) {
				t.Fatal("verifier called for malformed header")
				return nil, nil
			})
			handler := httpapi.Authenticate(verifier, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
				t.Fatal("downstream handler called")
			}))
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			for _, header := range headers {
				r.Header.Add("Authorization", header)
			}
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)
			if w.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want 401", w.Code)
			}
		})
	}
}

func associatedHeaderName(headers []string) string {
	if len(headers) == 0 {
		return "missing"
	}
	return headers[0]
}

func TestAuthenticateRejectsUnverifiedTokens(t *testing.T) {
	for _, tc := range []struct {
		name  string
		token *auth.Token
		err   error
	}{
		{name: "invalid", err: errors.New("invalid token")},
		{name: "expired", err: errors.New("expired token")},
		{name: "nil token"},
		{name: "empty UID", token: &auth.Token{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			verifier := verifierFunc(func(context.Context, string) (*auth.Token, error) {
				return tc.token, tc.err
			})
			handler := httpapi.Authenticate(verifier, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
				t.Fatal("downstream handler called")
			}))
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.Header.Set("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)
			if w.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want 401", w.Code)
			}
		})
	}
}

func TestAuthenticateStoresUID(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization", "bearer test-token")
	verifier := verifierFunc(func(ctx context.Context, token string) (*auth.Token, error) {
		if ctx != r.Context() || token != "test-token" {
			t.Fatal("verifier did not receive request context and extracted token")
		}
		return &auth.Token{UID: "firebase-uid"}, nil
	})
	called := false
	handler := httpapi.Authenticate(verifier, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if uid, ok := httpapi.UIDFromContext(r.Context()); !ok || uid != "firebase-uid" {
			t.Fatalf("context UID = %q, %v", uid, ok)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if !called || w.Code != http.StatusNoContent {
		t.Fatalf("handler called = %v, status = %d", called, w.Code)
	}
	if _, ok := httpapi.UIDFromContext(r.Context()); ok {
		t.Fatal("original request context was modified")
	}
}

func TestRoutes(t *testing.T) {
	verifier := verifierFunc(func(context.Context, string) (*auth.Token, error) {
		return &auth.Token{UID: "firebase-uid"}, nil
	})
	for _, tc := range []struct {
		name, path, header, key, value string
		status                         int
	}{
		{"public health", "/api/health", "", "status", "ok", http.StatusOK},
		{"protected user", "/api/users/me", "", "", "", http.StatusUnauthorized},
		{"current user", "/api/users/me", "Bearer test-token", "uid", "firebase-uid", http.StatusOK},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, tc.path, nil)
			if tc.header != "" {
				r.Header.Set("Authorization", tc.header)
			}
			w := httptest.NewRecorder()
			httpapi.NewRouter(verifier, userStoreStub{
				get: func(_ context.Context, identity models.Identity) (models.User, error) {
					return models.User{UID: identity.UID}, nil
				},
			}, nil).ServeHTTP(w, r)
			if w.Code != tc.status {
				t.Fatalf("status = %d, want %d", w.Code, tc.status)
			}
			if tc.status == http.StatusOK {
				var body map[string]any
				if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
					t.Fatal(err)
				}
				if w.Header().Get("Content-Type") != "application/json" || body[tc.key] != tc.value {
					t.Fatalf("unexpected response: headers=%v body=%v", w.Header(), body)
				}
			}
		})
	}
}
