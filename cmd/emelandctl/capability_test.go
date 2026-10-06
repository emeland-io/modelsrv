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

func readCapabilityVersions(t *testing.T, dir string) []Resource {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(dir, "capabilityversion-*.yaml"))
	require.NoError(t, err)
	out := make([]Resource, 0, len(matches))
	for _, m := range matches {
		out = append(out, readYAMLFile(t, m))
	}
	return out
}

func versionBlock(t *testing.T, r Resource) map[string]any {
	t.Helper()
	ver, ok := r.Spec["version"].(map[string]any)
	require.True(t, ok, "version block must be a map, got %T", r.Spec["version"])
	return ver
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

	cap := readYAMLFile(t, filepath.Join(dir, "capability-*.yaml"))
	assert.Equal(t, "Capability", cap.Kind)
	assert.Equal(t, "Mail Service", cap.Spec["displayName"])
	assert.NotEmpty(t, cap.Spec["capabilityId"])
	assert.Nil(t, cap.Spec["versions"])

	versions := readCapabilityVersions(t, dir)
	require.Len(t, versions, 2)

	byVer := map[string]Resource{}
	for _, v := range versions {
		assert.Equal(t, "CapabilityVersion", v.Kind)
		assert.Equal(t, cap.Spec["capabilityId"], v.Spec["capability"])
		vb := versionBlock(t, v)
		byVer[vb["version"].(string)] = v
	}

	v1 := versionBlock(t, byVer["1.0.0"])
	assert.Equal(t, "2026-01-01T00:00:00Z", v1["availableFrom"])
	assert.Equal(t, "2027-01-01T00:00:00Z", v1["deprecatedFrom"])
	assert.Equal(t, "2028-01-01T00:00:00Z", v1["terminatedFrom"])

	v2 := versionBlock(t, byVer["2.0.0"])
	assert.Equal(t, "2027-06-01T00:00:00Z", v2["availableFrom"])
	assert.Nil(t, v2["deprecatedFrom"])
	assert.Nil(t, v2["terminatedFrom"])
}

func TestCreateCapabilityInlineFlagValue(t *testing.T) {
	dir := t.TempDir()
	err := executeCmd("create", "-d", dir, "capability", "Cap",
		"--version=3.1.4", "--available-from=2026-05-05")
	require.NoError(t, err)

	versions := readCapabilityVersions(t, dir)
	require.Len(t, versions, 1)
	vb := versionBlock(t, versions[0])
	assert.Equal(t, "3.1.4", vb["version"])
	assert.Equal(t, "2026-05-05T00:00:00Z", vb["availableFrom"])
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

func TestCreateCapabilityNoVersionWritesCapabilityOnly(t *testing.T) {
	dir := t.TempDir()
	err := executeCmd("create", "-d", dir, "capability", "Bare Cap")
	require.NoError(t, err)

	r := readYAMLFile(t, filepath.Join(dir, "capability-*.yaml"))
	assert.Equal(t, "Bare Cap", r.Spec["displayName"])
	assert.Nil(t, r.Spec["versions"])
	assert.Empty(t, readCapabilityVersions(t, dir))
}

func TestCreateCapabilityRFC3339DatePassthrough(t *testing.T) {
	dir := t.TempDir()
	err := executeCmd("create", "-d", dir, "capability", "Cap",
		"--version", "1.0.0", "--available-from", "2026-01-02T15:04:05Z")
	require.NoError(t, err)

	versions := readCapabilityVersions(t, dir)
	require.Len(t, versions, 1)
	assert.Equal(t, "2026-01-02T15:04:05Z", versionBlock(t, versions[0])["availableFrom"])
}
