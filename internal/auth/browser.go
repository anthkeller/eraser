package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gorilla/securecookie"
	"golang.org/x/oauth2"
)

const (
	transactionCookie = "eraser_oidc_transaction"
	sessionCookie     = "eraser_session"
	transactionTTL    = 10 * time.Minute
	sessionTTL        = 8 * time.Hour
)

type BrowserSession interface {
	Login(http.ResponseWriter, *http.Request)
	Callback(http.ResponseWriter, *http.Request)
	Logout(http.ResponseWriter, *http.Request)
	Principal(*http.Request) (Principal, bool)
}

type transaction struct {
	State    string
	Verifier string
	ReturnTo string
	Expires  time.Time
}

type browserSession struct {
	oauth   oauth2.Config
	verify  Verifier
	owner   string
	cookies *securecookie.SecureCookie
	secure  bool
}

func NewBrowserSession(ctx context.Context, issuer, clientID, clientSecret, publicURL, owner, encodedKey string) (BrowserSession, error) {
	provider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		return nil, fmt.Errorf("discover OIDC provider: %w", err)
	}
	key, err := base64.RawURLEncoding.DecodeString(encodedKey)
	if err != nil || len(key) != 64 {
		return nil, errors.New("session key must be 64 random bytes encoded with unpadded base64url")
	}
	base, err := url.Parse(publicURL)
	if err != nil {
		return nil, fmt.Errorf("parse public URL: %w", err)
	}
	return &browserSession{
		oauth: oauth2.Config{
			ClientID: clientID, ClientSecret: clientSecret,
			Endpoint: provider.Endpoint(), RedirectURL: strings.TrimRight(base.String(), "/") + "/auth/callback",
			Scopes: []string{oidc.ScopeOpenID, "profile", "email"},
		},
		verify: &OIDCVerifier{verifier: provider.Verifier(&oidc.Config{ClientID: clientID})},
		owner:  owner, cookies: securecookie.New(key[:32], key[32:]), secure: base.Scheme == "https",
	}, nil
}

func (s *browserSession) Login(w http.ResponseWriter, r *http.Request) {
	state, err := randomToken(32)
	if err != nil {
		http.Error(w, "could not start login", http.StatusInternalServerError)
		return
	}
	verifier, err := randomToken(32)
	if err != nil {
		http.Error(w, "could not start login", http.StatusInternalServerError)
		return
	}
	txn := transaction{State: state, Verifier: verifier, ReturnTo: safeReturnTo(r.URL.Query().Get("return_to")), Expires: time.Now().Add(transactionTTL)}
	value, err := s.cookies.Encode(transactionCookie, txn)
	if err != nil {
		http.Error(w, "could not start login", http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, s.cookie(transactionCookie, value, transactionTTL))
	challenge := sha256.Sum256([]byte(verifier))
	http.Redirect(w, r, s.oauth.AuthCodeURL(state, oauth2.SetAuthURLParam("code_challenge", base64.RawURLEncoding.EncodeToString(challenge[:])), oauth2.SetAuthURLParam("code_challenge_method", "S256")), http.StatusFound)
}

func (s *browserSession) Callback(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(transactionCookie)
	if err != nil {
		http.Error(w, "login transaction expired", http.StatusBadRequest)
		return
	}
	var txn transaction
	if err := s.cookies.Decode(transactionCookie, cookie.Value, &txn); err != nil || time.Now().After(txn.Expires) || !constantEqual(txn.State, r.URL.Query().Get("state")) {
		s.clear(w, transactionCookie)
		http.Error(w, "invalid login transaction", http.StatusBadRequest)
		return
	}
	if providerError := r.URL.Query().Get("error"); providerError != "" {
		s.clear(w, transactionCookie)
		http.Error(w, "identity provider rejected login", http.StatusUnauthorized)
		return
	}
	token, err := s.oauth.Exchange(r.Context(), r.URL.Query().Get("code"), oauth2.VerifierOption(txn.Verifier))
	if err != nil {
		s.clear(w, transactionCookie)
		http.Error(w, "login exchange failed", http.StatusUnauthorized)
		return
	}
	raw, ok := token.Extra("id_token").(string)
	if !ok {
		s.clear(w, transactionCookie)
		http.Error(w, "identity provider returned no ID token", http.StatusUnauthorized)
		return
	}
	principal, err := s.verify.Verify(r.Context(), raw)
	if err != nil || !sameSubject(principal.Subject, s.owner) {
		s.clear(w, transactionCookie)
		http.Error(w, "account is not authorized for this Eraser instance", http.StatusForbidden)
		return
	}
	value, err := s.cookies.Encode(sessionCookie, struct {
		Subject string
		Expires time.Time
	}{principal.Subject, time.Now().Add(sessionTTL)})
	if err != nil {
		http.Error(w, "could not create session", http.StatusInternalServerError)
		return
	}
	s.clear(w, transactionCookie)
	http.SetCookie(w, s.cookie(sessionCookie, value, sessionTTL))
	http.Redirect(w, r, txn.ReturnTo, http.StatusSeeOther)
}

func (s *browserSession) Logout(w http.ResponseWriter, r *http.Request) {
	s.clear(w, sessionCookie)
	http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
}

func (s *browserSession) Principal(r *http.Request) (Principal, bool) {
	cookie, err := r.Cookie(sessionCookie)
	if err != nil {
		return Principal{}, false
	}
	var session struct {
		Subject string
		Expires time.Time
	}
	if err := s.cookies.Decode(sessionCookie, cookie.Value, &session); err != nil || time.Now().After(session.Expires) || !sameSubject(session.Subject, s.owner) {
		return Principal{}, false
	}
	return Principal{Subject: session.Subject}, true
}

func (s *browserSession) cookie(name, value string, ttl time.Duration) *http.Cookie {
	return &http.Cookie{Name: name, Value: value, Path: "/", MaxAge: int(ttl.Seconds()), HttpOnly: true, Secure: s.secure, SameSite: http.SameSiteLaxMode}
}

func (s *browserSession) clear(w http.ResponseWriter, name string) {
	c := s.cookie(name, "", -time.Hour)
	c.MaxAge = -1
	http.SetCookie(w, c)
}

func randomToken(size int) (string, error) {
	value := make([]byte, size)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

func safeReturnTo(value string) string {
	if value == "" || !strings.HasPrefix(value, "/") || strings.HasPrefix(value, "//") {
		return "/"
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.IsAbs() || parsed.Host != "" {
		return "/"
	}
	return parsed.RequestURI()
}

func constantEqual(a, b string) bool {
	return len(a) == len(b) && subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
