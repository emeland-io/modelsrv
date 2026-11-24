package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/twmb/franz-go/pkg/sasl/oauth"
)

const (
	defaultTokenTimeout = 30 * time.Second
	// refreshSkew is how long before expiry a cached token is replaced.
	refreshSkew = 30 * time.Second
)

// oauthTokens caches an OIDC client-credentials access token and refreshes it
// shortly before it expires. franz-go calls auth on every SASL session.
type oauthTokens struct {
	tokenURL string
	clientID string
	secret   string
	client   *http.Client

	mu        sync.Mutex
	token     string
	refreshAt time.Time
}

func newOAuthTokens(cfg SASLConfig) *oauthTokens {
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: defaultTokenTimeout}
	}
	return &oauthTokens{
		tokenURL: strings.TrimSpace(cfg.TokenURL),
		clientID: strings.TrimSpace(cfg.ClientID),
		secret:   strings.TrimSpace(cfg.ClientSecret),
		client:   client,
	}
}

func (t *oauthTokens) auth(ctx context.Context) (oauth.Auth, error) {
	token, err := t.accessToken(ctx)
	if err != nil {
		return oauth.Auth{}, err
	}
	return oauth.Auth{Token: token}, nil
}

func (t *oauthTokens) accessToken(ctx context.Context) (string, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.token != "" && time.Now().Before(t.refreshAt) {
		return t.token, nil
	}
	token, expiresIn, err := t.fetch(ctx)
	if err != nil {
		return "", err
	}
	t.token = token
	t.refreshAt = time.Now().Add(refreshLead(expiresIn))
	return token, nil
}

func refreshLead(expiresIn time.Duration) time.Duration {
	if expiresIn <= 0 {
		return time.Minute
	}
	lead := refreshSkew
	if half := expiresIn / 2; half < lead {
		lead = half
	}
	if expiresIn-lead < time.Second {
		return time.Second
	}
	return expiresIn - lead
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	TokenType   string `json:"token_type"`
	Error       string `json:"error"`
	ErrorDesc   string `json:"error_description"`
}

func (t *oauthTokens) fetch(ctx context.Context) (string, time.Duration, error) {
	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	form.Set("client_id", t.clientID)
	form.Set("client_secret", t.secret)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", 0, fmt.Errorf("kafka: oauth token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := t.client.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("kafka: oauth token endpoint: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", 0, fmt.Errorf("kafka: oauth token response: %w", err)
	}
	var parsed tokenResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", 0, fmt.Errorf("kafka: oauth token response is not JSON (HTTP %d)", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK || parsed.AccessToken == "" {
		msg := parsed.Error
		if parsed.ErrorDesc != "" {
			msg = parsed.Error + ": " + parsed.ErrorDesc
		}
		if msg == "" {
			msg = resp.Status
		}
		return "", 0, fmt.Errorf("kafka: oauth token rejected: %s", msg)
	}
	expires := time.Duration(parsed.ExpiresIn) * time.Second
	return parsed.AccessToken, expires, nil
}
