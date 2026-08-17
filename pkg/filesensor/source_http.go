package filesensor

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
)

// HTTPSource fetches a single remote file via HTTP GET.
// List returns one synthetic FileMeta; change detection uses ETag / Last-Modified when polling.
type HTTPSource struct {
	URL    string
	Client *http.Client
	Header http.Header
}

// NewHTTPSource returns a Source for a single URL.
func NewHTTPSource(rawURL string) *HTTPSource {
	return &HTTPSource{
		URL:    rawURL,
		Client: http.DefaultClient,
	}
}

func (s *HTTPSource) client() *http.Client {
	if s.Client != nil {
		return s.Client
	}
	return http.DefaultClient
}

func (s *HTTPSource) fileName() string {
	u := s.URL
	if i := strings.Index(u, "?"); i >= 0 {
		u = u[:i]
	}
	base := path.Base(u)
	if base == "" || base == "." || base == "/" {
		return "document"
	}
	return base
}

// List implements [Source].
func (s *HTTPSource) List(ctx context.Context) ([]FileMeta, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, s.URL, nil)
	if err != nil {
		return nil, err
	}
	for k, vs := range s.Header {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}
	resp, err := s.client().Do(req)
	if err != nil {
		// Some servers reject HEAD; fall back to a GET that we discard after headers.
		req, err = http.NewRequestWithContext(ctx, http.MethodGet, s.URL, nil)
		if err != nil {
			return nil, err
		}
		for k, vs := range s.Header {
			for _, v := range vs {
				req.Header.Add(k, v)
			}
		}
		resp, err = s.client().Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close() //nolint:errcheck
	} else {
		defer resp.Body.Close() //nolint:errcheck
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %s: status %d", s.URL, resp.StatusCode)
	}
	meta := FileMeta{
		Name:        s.fileName(),
		ETag:        resp.Header.Get("ETag"),
		ContentType: resp.Header.Get("Content-Type"),
	}
	if lm := resp.Header.Get("Last-Modified"); lm != "" {
		if t, err := http.ParseTime(lm); err == nil {
			meta.LastModified = t
		}
	}
	return []FileMeta{meta}, nil
}

// Read implements [Source].
func (s *HTTPSource) Read(ctx context.Context, name string) ([]byte, error) {
	if name != "" && name != s.fileName() {
		return nil, fmt.Errorf("HTTP source has no file %q", name)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.URL, nil)
	if err != nil {
		return nil, err
	}
	for k, vs := range s.Header {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}
	resp, err := s.client().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close() //nolint:errcheck
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %s: status %d", s.URL, resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}
