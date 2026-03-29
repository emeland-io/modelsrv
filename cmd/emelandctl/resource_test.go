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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// executeCmd builds a fresh command tree and runs it with the given args.
func executeCmd(args ...string) error {
	cmd := newRootCmd()
	cmd.SetArgs(args)
	return cmd.Execute()
}

// readYAMLFile finds the single YAML file matching the glob and unmarshals it.
func readYAMLFile(t *testing.T, pattern string) Resource {
	t.Helper()
	matches, err := filepath.Glob(pattern)
	require.NoError(t, err)
	require.Len(t, matches, 1, "expected exactly one file matching %s", pattern)

	data, err := os.ReadFile(matches[0])
	require.NoError(t, err)

	var r Resource
	require.NoError(t, yaml.Unmarshal(data, &r))
	return r
}

func TestCreateSystemPositionalArg(t *testing.T) {
	dir := t.TempDir()
	err := executeCmd("create", "-d", dir, "system", "Order Service", "--desc", "Handles orders")
	require.NoError(t, err)

	r := readYAMLFile(t, filepath.Join(dir, "system-*.yaml"))
	assert.Equal(t, resourceVersion, r.Version)
	assert.Equal(t, "System", r.Kind)
	assert.Equal(t, "Order Service", r.Spec["displayName"])
	assert.Equal(t, "Handles orders", r.Spec["description"])
	assert.NotEmpty(t, r.Spec["systemId"])
}

func TestCreateComponentWithNameFlag(t *testing.T) {
	dir := t.TempDir()
	err := executeCmd("create", "-d", dir, "component",
		"-n", "policy proxy",
		"--desc", "Governs access",
		"--system", "019beb6c-d1a1-73bd-893b-2aef9497b59a",
		"--annotation", "emeland.io/owner=abc123",
	)
	require.NoError(t, err)

	r := readYAMLFile(t, filepath.Join(dir, "component-*.yaml"))
	assert.Equal(t, "Component", r.Kind)
	assert.Equal(t, "policy proxy", r.Spec["displayName"])
	assert.Equal(t, "Governs access", r.Spec["description"])
	assert.Equal(t, "019beb6c-d1a1-73bd-893b-2aef9497b59a", r.Spec["system"])

	ann, ok := r.Spec["annotations"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "abc123", ann["emeland.io/owner"])
}

func TestCreateAPIWithAllFlags(t *testing.T) {
	dir := t.TempDir()
	err := executeCmd("create", "-d", dir, "api", "Order API",
		"--desc", "REST API",
		"--type", "OpenAPI",
		"--system", "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
	)
	require.NoError(t, err)

	r := readYAMLFile(t, filepath.Join(dir, "api-*.yaml"))
	assert.Equal(t, "API", r.Kind)
	assert.Equal(t, "Order API", r.Spec["displayName"])
	assert.Equal(t, "OpenAPI", r.Spec["type"])
	assert.Equal(t, "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", r.Spec["system"])
}

func TestCreateFindingUsesSummaryField(t *testing.T) {
	dir := t.TempDir()
	err := executeCmd("create", "-d", dir, "finding", "Missing TLS")
	require.NoError(t, err)

	r := readYAMLFile(t, filepath.Join(dir, "finding-*.yaml"))
	assert.Equal(t, "Finding", r.Kind)
	assert.Equal(t, "Missing TLS", r.Spec["summary"])
	assert.Nil(t, r.Spec["displayName"])
}

func TestCreateSystemWithAbstractFlag(t *testing.T) {
	dir := t.TempDir()
	err := executeCmd("create", "-d", dir, "system", "Abstract Sys", "--abstract")
	require.NoError(t, err)

	r := readYAMLFile(t, filepath.Join(dir, "system-*.yaml"))
	assert.Equal(t, true, r.Spec["abstract"])
}

func TestCreateFailsWithoutDisplayName(t *testing.T) {
	dir := t.TempDir()
	err := executeCmd("create", "-d", dir, "system")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "display name is required")
}

func TestCreateAllResourceTypes(t *testing.T) {
	for _, def := range resourceTypes {
		t.Run(def.kind, func(t *testing.T) {
			dir := t.TempDir()
			err := executeCmd("create", "-d", dir, def.use, "Test "+def.kind)
			require.NoError(t, err)

			r := readYAMLFile(t, filepath.Join(dir, strings.ToLower(def.kind)+"-*.yaml"))
			assert.Equal(t, resourceVersion, r.Version)
			assert.Equal(t, def.kind, r.Kind)
			assert.NotEmpty(t, r.Spec[def.idField])
		})
	}
}

func TestCreateWritesToDefaultDataDir(t *testing.T) {
	// Run from a temp directory so the default "data" subdir is created there.
	orig, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		require.NoError(t, os.Chdir(orig), "failed to restore working directory")
	}()

	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))

	err = executeCmd("create", "node-type", "Compute")
	require.NoError(t, err)

	matches, err := filepath.Glob(filepath.Join(dir, "data", "nodetype-*.yaml"))
	require.NoError(t, err)
	assert.Len(t, matches, 1)
}

func TestCreateWithOutputFlag(t *testing.T) {
	outFile := filepath.Join(t.TempDir(), "sub", "my-system.yaml")
	err := executeCmd("create", "-o", outFile, "system", "My System")
	require.NoError(t, err)

	data, err := os.ReadFile(outFile)
	require.NoError(t, err)

	var r Resource
	require.NoError(t, yaml.Unmarshal(data, &r))
	assert.Equal(t, "System", r.Kind)
	assert.Equal(t, "My System", r.Spec["displayName"])
}

func TestCreateFailsWithInvalidResourceType(t *testing.T) {
	err := executeCmd("create", "foobar")
	assert.Error(t, err)
}
