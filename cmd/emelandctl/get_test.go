/*
Copyright © 2025 Lutz Behnke

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// executeCmdOut runs the CLI and captures stdout.
func executeCmdOut(args ...string) (string, error) {
	buf := new(bytes.Buffer)
	cmd := newRootCmd()
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return buf.String(), err
}

func TestGetFindingsTable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/landscape/findings", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[
			{"instanceId":"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa","displayName":"Missing TLS","reference":"/landscape/findings/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"},
			{"instanceId":"bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb","displayName":"Open Port"}
		]`))
	}))
	defer srv.Close()

	out, err := executeCmdOut("get", "findings", "--server", srv.URL)
	require.NoError(t, err)
	assert.Contains(t, out, "ID")
	assert.Contains(t, out, "NAME")
	assert.Contains(t, out, "REFERENCE")
	assert.Contains(t, out, "Missing TLS")
	assert.Contains(t, out, "Open Port")
	assert.Contains(t, out, "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
}

func TestGetFindingsJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"instanceId":"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa","displayName":"Missing TLS"}]`))
	}))
	defer srv.Close()

	out, err := executeCmdOut("get", "findings", "--server", srv.URL, "-o", "json")
	require.NoError(t, err)

	var items []map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(out), &items))
	assert.Len(t, items, 1)
	assert.Equal(t, "Missing TLS", items[0]["displayName"])
}

func TestGetFindingsEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	out, err := executeCmdOut("get", "findings", "--server", srv.URL)
	require.NoError(t, err)
	assert.Contains(t, out, "ID")
	assert.NotContains(t, out, "Missing TLS")
}

func TestGetFindingsNoServer(t *testing.T) {
	_, err := executeCmdOut("get", "findings")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "server URL required")
}
