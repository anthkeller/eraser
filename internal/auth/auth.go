package auth

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
)

type contextKey struct{}

type Principal struct {
	Subject string `json:"subject"`
}

type Verifier interface {
	Verify(context.Context, string) (Principal, error)
}

type OIDCVerifier struct {
	verifier *oidc.IDTokenVerifier
}

func NewOIDCVerifier(ctx context.Context, issuer, audience string) (*OIDCVerifier, error) {
	provider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		return nil, fmt.Errorf("discover OIDC provider: %w", err)
	}
	return &OIDCVerifier{verifier: provider.Verifier(&oidc.Config{ClientID: audience})}, nil
}

func (v *OIDCVerifier) Verify(ctx context.Context, raw string) (Principal, error) {
	token, err := v.verifier.Verify(ctx, raw)
	if err != nil {
		return Principal{}, fmt.Errorf("verify ID token: %w", err)
	}
	if strings.TrimSpace(token.Subject) == "" {
		return Principal{}, errors.New("verified token has no subject")
	}
	return Principal{Subject: token.Subject}, nil
}

func Middleware(verifier Verifier, requiredSubject string, publicPaths map[string]struct{}) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, public := publicPaths[r.URL.Path]; public {
				next.ServeHTTP(w, r)
				return
			}
			raw, err := bearerToken(r.Header.Get("Authorization"))
			if err != nil {
				writeUnauthorized(w)
				return
			}
			principal, err := verifier.Verify(r.Context(), raw)
			if err != nil || !sameSubject(principal.Subject, requiredSubject) {
				writeUnauthorized(w)
				return
			}
			ctx := context.WithValue(r.Context(), contextKey{}, principal)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func PrincipalFromContext(ctx context.Context) (Principal, bool) {
	principal, ok := ctx.Value(contextKey{}).(Principal)
	return principal, ok
}

func bearerToken(header string) (string, error) {
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
		return "", errors.New("missing bearer token")
	}
	return parts[1], nil
}

func sameSubject(actual, expected string) bool {
	if len(actual) != len(expected) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(actual), []byte(expected)) == 1
}

func writeUnauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("WWW-Authenticate", `Bearer realm="eraser"`)
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
}
