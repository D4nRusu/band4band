package main

import (
	"context"
	"net/http"
	"strings"

	"firebase.google.com/go/v4/auth"
)

type tokenVerifier interface {
	VerifyIDToken(context.Context, string) (*auth.Token, error)
}

type uidContextKey struct{}

func uidFromContext(ctx context.Context) (string, bool) {
	uid, ok := ctx.Value(uidContextKey{}).(string)
	return uid, ok && uid != ""
}

func authenticate(verifier tokenVerifier, next http.Handler) http.Handler {
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
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
