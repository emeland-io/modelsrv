package endpointprobe

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/google/uuid"
	"go.emeland.io/modelsrv/pkg/model/api"
)

const (
	annProtocol = "emeland.io/endpoint.protocol"
	annHost     = "emeland.io/endpoint.host"
	annPort     = "emeland.io/endpoint.port"
	annPath     = "emeland.io/endpoint.path"
)

// ProbeTarget describes a synthetic HTTP/TLS probe destination derived from an ApiInstance.
type ProbeTarget struct {
	ApiInstanceID    uuid.UUID
	DisplayName      string
	APIID            uuid.UUID // zero if unset
	SystemInstanceID uuid.UUID // zero if unset
	URL              string
	DedupeKey        string // host:port
}

// BuildURL constructs a probe URL from protocol, host, port, and path components.
func BuildURL(protocol, host, port, path string) (string, error) {
	if err := validateProtocol(protocol); err != nil {
		return "", err
	}

	resolvedPort := defaultPort(protocol, port)
	normalizedPath := normalizePath(path)

	return (&url.URL{
		Scheme: protocol,
		Host:   host + ":" + resolvedPort,
		Path:   normalizedPath,
	}).String(), nil
}

// TargetFromApiInstance derives a probe target from ApiInstance endpoint annotations.
// When emeland.io/endpoint.host is missing, the instance is skipped (ok == false, err == nil).
func TargetFromApiInstance(ai api.ApiInstance) (ProbeTarget, bool, error) {
	annotations := ai.GetAnnotations()
	host := annotations.GetValue(annHost)
	if host == "" {
		return ProbeTarget{}, false, nil
	}

	protocol := annotations.GetValue(annProtocol)
	port := defaultPort(protocol, annotations.GetValue(annPort))
	path := normalizePath(annotations.GetValue(annPath))

	url, err := BuildURL(protocol, host, port, path)
	if err != nil {
		return ProbeTarget{}, false, err
	}

	target := ProbeTarget{
		ApiInstanceID: ai.GetInstanceId(),
		DisplayName:   ai.GetDisplayName(),
		URL:           url,
		DedupeKey:     host + ":" + port,
	}

	if apiRef := ai.GetApiRef(); apiRef != nil {
		target.APIID = apiRef.ApiID
	}

	if sysInstRef := ai.GetSystemInstance(); sysInstRef != nil {
		target.SystemInstanceID = sysInstRef.InstanceId
	}

	return target, true, nil
}

func validateProtocol(protocol string) error {
	switch protocol {
	case "http", "https":
		return nil
	default:
		return fmt.Errorf("invalid protocol %q (expected http or https)", protocol)
	}
}

func defaultPort(protocol, port string) string {
	if port != "" {
		return port
	}

	switch protocol {
	case "https":
		return "443"
	case "http":
		return "80"
	default:
		return port
	}
}

func normalizePath(path string) string {
	if path == "" {
		return "/"
	}
	if !strings.HasPrefix(path, "/") {
		return "/" + path
	}
	return path
}
