package filesensor

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
	"time"
)

// DefaultHTTPTimeout is the client timeout used when [SourceConfig.Timeout] is unset.
const DefaultHTTPTimeout = 30 * time.Second

var defaultHTTPClient = &http.Client{Timeout: DefaultHTTPTimeout}

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
		Client: defaultHTTPClient,
	}
}

func (s *HTTPSource) client() *http.Client {
	if s.Client != nil {
		return s.Client
	}
	return defaultHTTPClient
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

func (s *HTTPSource) newRequest(ctx context.Context, method string, extra http.Header) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, s.URL, nil)
	if err != nil {
		return nil, err
	}
	for k, vs := range s.Header {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}
	for k, vs := range extra {
		for _, v := range vs {
			req.Header.Set(k, v)
		}
	}
	return req, nil
}

func closeResponseBody(resp *http.Response, peek int64) {
	if resp == nil || resp.Body == nil {
		return
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, peek))
	_ = resp.Body.Close()
}

func headRejected(err error, resp *http.Response) bool {
	if err != nil || resp == nil {
		return true
	}
	switch resp.StatusCode {
	case http.StatusMethodNotAllowed, http.StatusNotImplemented:
		return true
	}
	return false
}

func httpStatusOK(code int) bool {
	return code >= 200 && code < 300
}

// List implements [Source].
func (s *HTTPSource) List(ctx context.Context) ([]FileMeta, error) {
	req, err := s.newRequest(ctx, http.MethodHead, nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.client().Do(req)
	if headRejected(err, resp) {
		closeResponseBody(resp, 0)
		// Some servers reject HEAD; fetch headers only via a ranged GET so a
		// large body is not downloaded on every poll.
		req, err = s.newRequest(ctx, http.MethodGet, http.Header{
			"Range": []string{"bytes=0-0"},
		})
		if err != nil {
			return nil, err
		}
		req.Close = true
		resp, err = s.client().Do(req)
		if err != nil {
			return nil, err
		}
		// Headers are available as soon as Do returns. Read at most one byte
		// then close so a Range-ignoring server cannot fill the connection.
		closeResponseBody(resp, 1)
	} else {
		closeResponseBody(resp, 0)
	}
	if !httpStatusOK(resp.StatusCode) {
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
	req, err := s.newRequest(ctx, http.MethodGet, nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.client().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close() //nolint:errcheck
	if !httpStatusOK(resp.StatusCode) {
		return nil, fmt.Errorf("HTTP %s: status %d", s.URL, resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func httpClientWithTimeout(timeout time.Duration) *http.Client {
	if timeout <= 0 {
		return defaultHTTPClient
	}
	if timeout == DefaultHTTPTimeout {
		return defaultHTTPClient
	}
	return &http.Client{Timeout: timeout}
}
