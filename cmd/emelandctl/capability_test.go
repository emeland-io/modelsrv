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
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// capVersions extracts the versions list from a Capability resource spec.
func capVersions(t *testing.T, r Resource) []map[string]any {
	t.Helper()
	raw, ok := r.Spec["versions"].([]any)
	require.True(t, ok, "versions must be a list, got %T", r.Spec["versions"])
	out := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		m, ok := item.(map[string]any)
		require.True(t, ok, "version entry must be a map, got %T", item)
		out = append(out, m)
	}
	return out
}

// versionField returns the nested version.<key> value for a version entry.
func versionField(t *testing.T, entry map[string]any, key string) any {
	t.Helper()
	ver, ok := entry["version"].(map[string]any)
	require.True(t, ok, "version block must be a map, got %T", entry["version"])
	return ver[key]
}

func TestCreateCapabilityMultiVersionScoping(t *testing.T) {
	dir := t.TempDir()
	err := executeCmd("create", "-d", dir, "capability", "Mail Service",
		"--version", "1.0.0",
		"--available-from", "2026-01-01",
		"--deprecated-from", "2027-01-01",
		"--terminated-from", "2028-01-01",
		"--version", "2.0.0",
		"--available-from", "2027-06-01",
	)
	require.NoError(t, err)

	r := readYAMLFile(t, filepath.Join(dir, "capability-*.yaml"))
	assert.Equal(t, "Capability", r.Kind)
	assert.Equal(t, "Mail Service", r.Spec["displayName"])
	assert.NotEmpty(t, r.Spec["capabilityId"])

	versions := capVersions(t, r)
	require.Len(t, versions, 2)

	// Version 1 got all three dates, scoped to the first --version.
	assert.Equal(t, "1.0.0", versionField(t, versions[0], "version"))
	assert.Equal(t, "2026-01-01T00:00:00Z", versionField(t, versions[0], "availableFrom"))
	assert.Equal(t, "2027-01-01T00:00:00Z", versionField(t, versions[0], "deprecatedFrom"))
	assert.Equal(t, "2028-01-01T00:00:00Z", versionField(t, versions[0], "terminatedFrom"))

	// Version 2 only got availableFrom; the other dates must not leak from v1.
	assert.Equal(t, "2.0.0", versionField(t, versions[1], "version"))
	assert.Equal(t, "2027-06-01T00:00:00Z", versionField(t, versions[1], "availableFrom"))
	assert.Nil(t, versionField(t, versions[1], "deprecatedFrom"))
	assert.Nil(t, versionField(t, versions[1], "terminatedFrom"))
}

func TestCreateCapabilityScaffoldKeys(t *testing.T) {
	dir := t.TempDir()
	err := executeCmd("create", "-d", dir, "capability", "Cap", "--version", "1.0.0")
	require.NoError(t, err)

	r := readYAMLFile(t, filepath.Join(dir, "capability-*.yaml"))
	versions := capVersions(t, r)
	require.Len(t, versions, 1)

	entry := versions[0]
	assert.NotEmpty(t, entry["capabilityVersionId"])

	// Empty dependencies scaffold.
	deps, ok := entry["dependencies"].([]any)
	require.True(t, ok, "dependencies must be a list")
	assert.Empty(t, deps)

	// Single variant scaffold with its own empty dependencies.
	variants, ok := entry["variants"].([]any)
	require.True(t, ok, "variants must be a list")
	require.Len(t, variants, 1)
	variant, ok := variants[0].(map[string]any)
	require.True(t, ok)
	assert.NotEmpty(t, variant["variantId"])
	vdeps, ok := variant["dependencies"].([]any)
	require.True(t, ok)
	assert.Empty(t, vdeps)
}

func TestCreateCapabilityInlineFlagValue(t *testing.T) {
	dir := t.TempDir()
	err := executeCmd("create", "-d", dir, "capability", "Cap",
		"--version=3.1.4", "--available-from=2026-05-05")
	require.NoError(t, err)

	r := readYAMLFile(t, filepath.Join(dir, "capability-*.yaml"))
	versions := capVersions(t, r)
	require.Len(t, versions, 1)
	assert.Equal(t, "3.1.4", versionField(t, versions[0], "version"))
	assert.Equal(t, "2026-05-05T00:00:00Z", versionField(t, versions[0], "availableFrom"))
}

func TestCreateCapabilityNameFlagAndAnnotation(t *testing.T) {
	dir := t.TempDir()
	err := executeCmd("create", "-d", dir, "capability",
		"-n", "Named Cap",
		"--version", "1.0.0",
		"--annotation", "emeland.io/owner=platform",
	)
	require.NoError(t, err)

	r := readYAMLFile(t, filepath.Join(dir, "capability-*.yaml"))
	assert.Equal(t, "Named Cap", r.Spec["displayName"])
	ann, ok := r.Spec["annotations"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "platform", ann["emeland.io/owner"])
}

func TestCreateCapabilityDateWithoutVersionFails(t *testing.T) {
	dir := t.TempDir()
	err := executeCmd("create", "-d", dir, "capability", "Cap", "--available-from", "2026-01-01")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must follow a --version")
}

func TestCreateCapabilityInvalidDateFails(t *testing.T) {
	dir := t.TempDir()
	err := executeCmd("create", "-d", dir, "capability", "Cap",
		"--version", "1.0.0", "--available-from", "01/02/2026")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid date")
}

func TestCreateCapabilityRequiresDisplayName(t *testing.T) {
	dir := t.TempDir()
	err := executeCmd("create", "-d", dir, "capability", "--version", "1.0.0")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "display name is required")
}

func TestCreateCapabilityNoVersionStillProducesFullConstruct(t *testing.T) {
	// TestCreateAllResourceTypes covers this too, but assert the shape here:
	// a capability with only a display name still yields a single scaffold version.
	dir := t.TempDir()
	err := executeCmd("create", "-d", dir, "capability", "Bare Cap")
	require.NoError(t, err)

	r := readYAMLFile(t, filepath.Join(dir, "capability-*.yaml"))
	versions := capVersions(t, r)
	require.Len(t, versions, 1)
	assert.Equal(t, "", versionField(t, versions[0], "version"))
	_, ok := versions[0]["variants"].([]any)
	assert.True(t, ok, "even a bare capability carries the variant scaffold")
}

func TestCreateCapabilityRFC3339DatePassthrough(t *testing.T) {
	dir := t.TempDir()
	err := executeCmd("create", "-d", dir, "capability", "Cap",
		"--version", "1.0.0", "--available-from", "2026-01-02T15:04:05Z")
	require.NoError(t, err)

	r := readYAMLFile(t, filepath.Join(dir, "capability-*.yaml"))
	versions := capVersions(t, r)
	assert.Equal(t, "2026-01-02T15:04:05Z", versionField(t, versions[0], "availableFrom"))
}
