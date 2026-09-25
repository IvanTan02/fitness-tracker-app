// Package auth verifies Supabase-issued JWTs and attaches the resolved
// user_id to the request context. It does not implement any session or
// password logic itself — that's Supabase Auth's job.
//
// Verification uses Supabase's published JWKS (project signing keys), the
// current recommended approach now that Supabase is phasing out the legacy
// shared JWT secret.
package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const userIDContextKey contextKey = "user_id"

// Verifier verifies Supabase-issued JWTs against the project's JWKS.
type Verifier struct {
	keyfunc keyfunc.Keyfunc
}

// NewVerifier creates a Verifier that fetches and auto-refreshes the given
// project's JWKS. supabaseURL is the project's base URL, e.g.
// "https://xxxx.supabase.co".
func NewVerifier(ctx context.Context, supabaseURL string) (*Verifier, error) {
	jwksURL := strings.TrimRight(supabaseURL, "/") + "/auth/v1/.well-known/jwks.json"

	kf, err := keyfunc.NewDefaultCtx(ctx, []string{jwksURL})
	if err != nil {
		return nil, fmt.Errorf("fetching JWKS from %s: %w", jwksURL, err)
	}

	return &Verifier{keyfunc: kf}, nil
}

// Middleware returns HTTP middleware that verifies the Supabase JWT from the
// Authorization header on every request and attaches the resolved user_id to
// the request context. Requests without a valid token are rejected with 401.
func (v *Verifier) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, err := v.verifyToken(r)
			if err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), userIDContextKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func (v *Verifier) verifyToken(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", fmt.Errorf("missing Authorization header")
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", fmt.Errorf("malformed Authorization header")
	}
	tokenString := parts[1]

	token, err := jwt.Parse(tokenString, v.keyfunc.Keyfunc)
	if err != nil {
		return "", fmt.Errorf("parsing token: %w", err)
	}
	if !token.Valid {
		return "", fmt.Errorf("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", fmt.Errorf("unexpected claims type")
	}

	sub, ok := claims["sub"].(string)
	if !ok || sub == "" {
		return "", fmt.Errorf("token missing sub claim")
	}

	return sub, nil
}

// UserID extracts the authenticated user's ID from a request context
// populated by Middleware. It panics if called on a context that was never
// passed through Middleware — that's a programming error, not a runtime one,
// since every protected route must go through it.
func UserID(ctx context.Context) string {
	userID, ok := ctx.Value(userIDContextKey).(string)
	if !ok {
		panic("auth.UserID called on a context without an authenticated user; is this route behind auth.Middleware?")
	}
	return userID
}
