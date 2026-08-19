package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestServerCmd_MissingSensorConfigReturnsError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.yaml")
	err := executeServer(t, "--sensor-config", path)
	if err == nil {
		t.Fatal("expected error so cobra exits non-zero")
	}
	if !strings.Contains(err.Error(), "could not load sensor config") {
		t.Fatalf("error %q does not mention load failure", err)
	}
}

func TestServerCmd_UnopenableSourcesReturnsError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sensor.yaml")
	body := []byte("sources:\n  - uri: ftp://example.com/files\n")
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatal(err)
	}
	err := executeServer(t, "--sensor-config", path)
	if err == nil {
		t.Fatal("expected error so cobra exits non-zero")
	}
	if !strings.Contains(err.Error(), "could not open sources") {
		t.Fatalf("error %q does not mention open failure", err)
	}
}

func executeServer(t *testing.T, args ...string) error {
	t.Helper()
	rootCmd.SetArgs(append([]string{"server"}, args...))
	rootCmd.SetOut(io.Discard)
	rootCmd.SetErr(io.Discard)
	t.Cleanup(func() { rootCmd.SetArgs(nil) })
	return rootCmd.Execute()
}
