package kafka

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestOAuthTokenCache(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if err := r.ParseForm(); err != nil {
			t.Errorf("parse form: %v", err)
		}
		if r.PostForm.Get("grant_type") != "client_credentials" {
			t.Errorf("grant_type = %q", r.PostForm.Get("grant_type"))
		}
		if r.PostForm.Get("client_id") != "kafka-emeland-sandbox" {
			t.Errorf("client_id = %q", r.PostForm.Get("client_id"))
		}
		if r.PostForm.Get("client_secret") != "secret" {
			t.Errorf("client_secret missing")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"tok-1","expires_in":300,"token_type":"Bearer"}`))
	}))
	defer srv.Close()

	tokens := newOAuthTokens(SASLConfig{
		TokenURL:     " " + srv.URL + " ",
		ClientID:     " kafka-emeland-sandbox ",
		ClientSecret: " secret ",
		HTTPClient:   srv.Client(),
	})
	ctx := context.Background()
	a1, err := tokens.auth(ctx)
	if err != nil {
		t.Fatal(err)
	}
	a2, err := tokens.auth(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if a1.Token != "tok-1" || a2.Token != "tok-1" {
		t.Fatalf("tokens = %q %q", a1.Token, a2.Token)
	}
	if calls.Load() != 1 {
		t.Fatalf("token endpoint called %d times, want 1 (cached)", calls.Load())
	}
}

func TestOAuthTokenRefreshBeforeExpiry(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		_, _ = fmt.Fprintf(w, `{"access_token":"tok-%d","expires_in":2,"token_type":"Bearer"}`, n)
	}))
	defer srv.Close()

	tokens := newOAuthTokens(SASLConfig{
		TokenURL:     srv.URL,
		ClientID:     "id",
		ClientSecret: "secret",
		HTTPClient:   srv.Client(),
	})
	if _, err := tokens.auth(context.Background()); err != nil {
		t.Fatal(err)
	}
	time.Sleep(1200 * time.Millisecond)
	if _, err := tokens.auth(context.Background()); err != nil {
		t.Fatal(err)
	}
	if calls.Load() < 2 {
		t.Fatalf("short-lived token was not refreshed, calls=%d", calls.Load())
	}
}

func TestOAuthTokenRejected(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"invalid_client","error_description":"bad secret"}`))
	}))
	defer srv.Close()

	tokens := newOAuthTokens(SASLConfig{
		TokenURL:     srv.URL,
		ClientID:     "id",
		ClientSecret: "nope",
		HTTPClient:   srv.Client(),
	})
	_, err := tokens.auth(context.Background())
	if err == nil || !strings.Contains(err.Error(), "invalid_client") {
		t.Fatalf("err = %v", err)
	}
}
