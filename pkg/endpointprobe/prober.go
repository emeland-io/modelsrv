package endpointprobe

import (
	"context"
	"crypto/tls"
	"net/http"
	"strings"
	"time"
)

// ProbeResult holds the outcome of a single HTTP/TLS probe.
type ProbeResult struct {
	Target        ProbeTarget
	Success       bool
	CertRemaining time.Duration
	HasCert       bool
	Err           error
	ProbedAt      time.Time
}

// Prober performs HTTP/TLS probes against probe targets.
type Prober struct {
	client *http.Client
}

// NewProber returns a prober that uses the given per-request timeout.
func NewProber(timeout time.Duration) *Prober {
	return NewProberWithClient(&http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				// Skip chain verification so expired or self-signed certs still
				// yield NotAfter for cert_remaining_seconds monitoring.
				InsecureSkipVerify: true, //nolint:gosec // intentional for cert expiry probing
			},
		},
	})
}

// NewProberWithClient returns a prober that uses the supplied HTTP client.
func NewProberWithClient(client *http.Client) *Prober {
	return &Prober{client: client}
}

// Probe performs a GET against target.URL and extracts TLS certificate expiry when applicable.
// ctx cancellation aborts the request when possible.
func (p *Prober) Probe(ctx context.Context, target ProbeTarget) ProbeResult {
	result := ProbeResult{
		Target:   target,
		ProbedAt: time.Now(),
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target.URL, nil)
	if err != nil {
		result.Err = err
		return result
	}

	resp, err := p.client.Do(req)
	if err != nil {
		result.Err = err
		return result
	}
	defer func() { _ = resp.Body.Close() }()

	if strings.HasPrefix(strings.ToLower(target.URL), "https://") {
		if resp.TLS != nil && len(resp.TLS.PeerCertificates) > 0 {
			notAfter := resp.TLS.PeerCertificates[0].NotAfter
			result.HasCert = true
			result.CertRemaining = notAfter.Sub(result.ProbedAt)
		}
	}

	result.Success = true
	return result
}
