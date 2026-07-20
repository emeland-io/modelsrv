package endpointprobe

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.emeland.io/modelsrv/pkg/model/api"
	"go.emeland.io/modelsrv/pkg/model/system"
)

func TestBuildURL(t *testing.T) {
	tests := []struct {
		name      string
		protocol  string
		host      string
		port      string
		path      string
		wantURL   string
		wantError string
	}{
		{
			name:     "full annotation set",
			protocol: "https",
			host:     "payments.prod.eu.example.com",
			port:     "443",
			path:     "/api/v1/health",
			wantURL:  "https://payments.prod.eu.example.com:443/api/v1/health",
		},
		{
			name:     "empty port defaults to 443 for https",
			protocol: "https",
			host:     "example.com",
			port:     "",
			path:     "/",
			wantURL:  "https://example.com:443/",
		},
		{
			name:     "empty port defaults to 80 for http",
			protocol: "http",
			host:     "example.com",
			port:     "",
			path:     "/",
			wantURL:  "http://example.com:80/",
		},
		{
			name:     "empty path defaults to slash",
			protocol: "https",
			host:     "example.com",
			port:     "443",
			path:     "",
			wantURL:  "https://example.com:443/",
		},
		{
			name:     "path without leading slash is normalized",
			protocol: "https",
			host:     "example.com",
			port:     "443",
			path:     "api/v1/health",
			wantURL:  "https://example.com:443/api/v1/health",
		},
		{
			name:      "invalid protocol ftp",
			protocol:  "ftp",
			host:      "example.com",
			port:      "21",
			path:      "/",
			wantError: `invalid protocol "ftp" (expected http or https)`,
		},
		{
			name:      "empty protocol is invalid",
			protocol:  "",
			host:      "example.com",
			port:      "443",
			path:      "/",
			wantError: `invalid protocol "" (expected http or https)`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotURL, err := BuildURL(tt.protocol, tt.host, tt.port, tt.path)

			if tt.wantError != "" {
				require.Error(t, err)
				assert.EqualError(t, err, tt.wantError)
				assert.Empty(t, gotURL)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantURL, gotURL)
		})
	}
}

func TestTargetFromApiInstance(t *testing.T) {
	instanceID := uuid.MustParse("88888888-0000-4000-8000-000000000001")
	apiID := uuid.MustParse("aaaaaaaa-0000-4000-8000-000000000001")
	systemInstanceID := uuid.MustParse("77777777-0000-4000-8000-000000000102")

	tests := []struct {
		name      string
		setup     func(t *testing.T) api.ApiInstance
		wantOK    bool
		want      ProbeTarget
		wantError string
	}{
		{
			name: "full annotation set",
			setup: func(t *testing.T) api.ApiInstance {
				t.Helper()

				ai := api.NewApiInstance(instanceID)
				ai.SetDisplayName("Payments API (prod EU)")
				ai.SetApiRefByRef(api.NewAPI(apiID))
				ai.SetSystemInstanceByRef(system.NewSystemInstance(systemInstanceID))

				annotations := ai.GetAnnotations()
				annotations.Add(annProtocol, "https")
				annotations.Add(annHost, "payments.prod.eu.example.com")
				annotations.Add(annPort, "443")
				annotations.Add(annPath, "/api/v1/health")

				return ai
			},
			wantOK: true,
			want: ProbeTarget{
				ApiInstanceID:    instanceID,
				DisplayName:      "Payments API (prod EU)",
				APIID:            apiID,
				SystemInstanceID: systemInstanceID,
				URL:              "https://payments.prod.eu.example.com:443/api/v1/health",
				DedupeKey:        "payments.prod.eu.example.com:443",
			},
		},
		{
			name: "missing host is skipped",
			setup: func(t *testing.T) api.ApiInstance {
				t.Helper()

				ai := api.NewApiInstance(instanceID)
				ai.GetAnnotations().Add(annProtocol, "https")

				return ai
			},
			wantOK: false,
		},
		{
			name: "empty port and path use defaults",
			setup: func(t *testing.T) api.ApiInstance {
				t.Helper()

				ai := api.NewApiInstance(instanceID)
				ai.SetDisplayName("Health API")

				annotations := ai.GetAnnotations()
				annotations.Add(annProtocol, "http")
				annotations.Add(annHost, "example.com")

				return ai
			},
			wantOK: true,
			want: ProbeTarget{
				ApiInstanceID: instanceID,
				DisplayName:   "Health API",
				URL:           "http://example.com:80/",
				DedupeKey:     "example.com:80",
			},
		},
		{
			name: "invalid protocol returns error",
			setup: func(t *testing.T) api.ApiInstance {
				t.Helper()

				ai := api.NewApiInstance(instanceID)
				annotations := ai.GetAnnotations()
				annotations.Add(annProtocol, "ftp")
				annotations.Add(annHost, "example.com")

				return ai
			},
			wantOK:    false,
			wantError: `invalid protocol "ftp" (expected http or https)`,
		},
		{
			name: "unset refs produce zero UUIDs",
			setup: func(t *testing.T) api.ApiInstance {
				t.Helper()

				ai := api.NewApiInstance(instanceID)
				annotations := ai.GetAnnotations()
				annotations.Add(annProtocol, "https")
				annotations.Add(annHost, "example.com")

				return ai
			},
			wantOK: true,
			want: ProbeTarget{
				ApiInstanceID: instanceID,
				URL:           "https://example.com:443/",
				DedupeKey:     "example.com:443",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ai := tt.setup(t)

			got, ok, err := TargetFromApiInstance(ai)

			if tt.wantError != "" {
				require.Error(t, err)
				assert.EqualError(t, err, tt.wantError)
				assert.False(t, ok)
				assert.Equal(t, ProbeTarget{}, got)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantOK, ok)

			if !tt.wantOK {
				assert.Equal(t, ProbeTarget{}, got)
				return
			}

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDedupeKeyStable(t *testing.T) {
	host := "payments.prod.eu.example.com"
	port := "443"

	makeInstance := func(path string) api.ApiInstance {
		ai := api.NewApiInstance(uuid.New())
		annotations := ai.GetAnnotations()
		annotations.Add(annProtocol, "https")
		annotations.Add(annHost, host)
		annotations.Add(annPort, port)
		annotations.Add(annPath, path)
		return ai
	}

	first, ok, err := TargetFromApiInstance(makeInstance("/api/v1/health"))
	require.NoError(t, err)
	require.True(t, ok)

	second, ok, err := TargetFromApiInstance(makeInstance("/metrics"))
	require.NoError(t, err)
	require.True(t, ok)

	assert.Equal(t, first.DedupeKey, second.DedupeKey)
	assert.Equal(t, "payments.prod.eu.example.com:443", first.DedupeKey)
	assert.NotEqual(t, first.URL, second.URL)
}
