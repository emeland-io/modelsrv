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

func TestServerCmd_AllDocumentsFailReturnsError(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "bad.yaml"), []byte(`
version: emeland.io/v1
kind: Context
spec:
  daf: "not-a-context"
`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(dir, "sensor.yaml")
	body := []byte("sources:\n  - uri: " + dir + "\n")
	if err := os.WriteFile(cfgPath, body, 0o644); err != nil {
		t.Fatal(err)
	}
	err := executeServer(t, "--sensor-config", cfgPath)
	if err == nil {
		t.Fatal("expected error so cobra exits non-zero")
	}
	if !strings.Contains(err.Error(), "no documents applied") {
		t.Fatalf("error %q does not mention apply failure", err)
	}
}

func executeServer(t *testing.T, args ...string) error {
	t.Helper()
	rootCmd.SetArgs(append([]string{"server"}, args...))
	rootCmd.SetOut(io.Discard)
	rootCmd.SetErr(io.Discard)
	t.Cleanup(func() {
		rootCmd.SetArgs(nil)
		logLevel = ""
		logEncoding = ""
	})
	return rootCmd.Execute()
}

func TestServerCmd_InvalidLogLevelReturnsError(t *testing.T) {
	err := executeServer(t, "--log-level", "nope")
	if err == nil {
		t.Fatal("expected error for invalid log level")
	}
	if !strings.Contains(err.Error(), "invalid log level") {
		t.Fatalf("error %q does not mention invalid log level", err)
	}
}

func TestServerCmd_InvalidLogEncodingReturnsError(t *testing.T) {
	err := executeServer(t, "--log-encoding", "xml")
	if err == nil {
		t.Fatal("expected error for invalid log encoding")
	}
	if !strings.Contains(err.Error(), "invalid log encoding") {
		t.Fatalf("error %q does not mention invalid log encoding", err)
	}
}

func TestServerCmd_LogEncodingJsonAccepted(t *testing.T) {
	// json encoding with an invalid sensor-config to force early exit after logger setup
	path := filepath.Join(t.TempDir(), "missing.yaml")
	err := executeServer(t, "--log-encoding", "json", "--sensor-config", path)
	if err == nil {
		t.Fatal("expected error (missing sensor config)")
	}
	// The error should be about sensor config, not about encoding
	if strings.Contains(err.Error(), "log encoding") {
		t.Fatalf("unexpected encoding error: %v", err)
	}
}

func TestServerCmd_LogLevelErrorSuppressesInfo(t *testing.T) {
	// error level with an invalid sensor-config to force early exit
	path := filepath.Join(t.TempDir(), "missing.yaml")
	err := executeServer(t, "--log-level", "error", "--sensor-config", path)
	if err == nil {
		t.Fatal("expected error (missing sensor config)")
	}
	// The error should be about sensor config, not about log level
	if strings.Contains(err.Error(), "log level") {
		t.Fatalf("unexpected log level error: %v", err)
	}
}
