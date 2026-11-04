package httpapi

import (
	"context"
	"net/http"
	"strings"

	"firebase.google.com/go/v4/auth"

	"github.com/D4nRusu/band4band/backend/internal/models"
)

// TokenVerifier verifies Firebase ID tokens.
type TokenVerifier interface {
	VerifyIDToken(context.Context, string) (*auth.Token, error)
}

type uidContextKey struct{}
type identityContextKey struct{}

func identityFromContext(ctx context.Context) (models.Identity, bool) {
	identity, ok := ctx.Value(identityContextKey{}).(models.Identity)
	return identity, ok && identity.UID != ""
}

// UIDFromContext returns the authenticated Firebase UID, if present.
func UIDFromContext(ctx context.Context) (string, bool) {
	uid, ok := ctx.Value(uidContextKey{}).(string)
	return uid, ok && uid != ""
}

// Authenticate requires a valid Firebase Bearer token before calling next.
func Authenticate(verifier TokenVerifier, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		headers := r.Header.Values("Authorization")
		if len(headers) != 1 {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		parts := strings.Fields(headers[0])
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		token, err := verifier.VerifyIDToken(r.Context(), parts[1])
		if err != nil || token == nil || token.UID == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), uidContextKey{}, token.UID)
		email, _ := token.Claims["email"].(string)
		name, _ := token.Claims["name"].(string)
		photoURL, _ := token.Claims["picture"].(string)
		ctx = context.WithValue(ctx, identityContextKey{}, models.Identity{
			UID: token.UID, Email: email, Name: name, PhotoURL: photoURL,
		})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
