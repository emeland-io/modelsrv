package main

import (
	"net/url"
	"strings"

	"go.emeland.io/modelsrv/pkg/events"
	"go.uber.org/zap"
)

// parseCommaSeparatedList splits a comma-separated string into trimmed non-empty entries.
func parseCommaSeparatedList(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if s := strings.TrimSpace(part); s != "" {
			out = append(out, s)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// isValidSubscriberURL reports whether s is an absolute http(s) URL with a host.
func isValidSubscriberURL(s string) bool {
	u, err := url.Parse(s)
	if err != nil {
		return false
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}
	return u.Host != ""
}

// registerStartupSubscribers pre-registers downstream callback URLs via AddSubscriber.
// Invalid URLs and AddSubscriber errors are logged; registration continues for remaining URLs.
func registerStartupSubscribers(em events.EventManager, urls []string, log *zap.SugaredLogger) {
	if log == nil {
		log = zap.NewNop().Sugar()
	}
	for _, raw := range urls {
		if !isValidSubscriberURL(raw) {
			log.Errorw("invalid subscriber URL", "url", raw)
			continue
		}
		if err := em.AddSubscriber(raw); err != nil {
			log.Errorw("failed to register subscriber", "url", raw, "error", err.Error())
			continue
		}
		log.Infow("registered subscriber", "url", raw)
	}
}
