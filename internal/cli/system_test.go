package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func writeScript(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "tool")
	if err := os.WriteFile(p, []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestSystemEnvLookPath(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "faketool"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	tests := []struct {
		name    string
		tool    string
		wantErr bool
	}{
		{"found", "faketool", false},
		{"missing", "no-such-tool", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, err := systemEnv{}.LookPath(tt.tool)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && path != filepath.Join(dir, tt.tool) {
				t.Errorf("path = %q", path)
			}
		})
	}
}

func TestSystemEnvVersion(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		want    string
		wantErr bool
	}{
		{"first line only", "printf 'tool 1.0\\nsecond line\\n'", "tool 1.0", false},
		{"non-zero exit with output", "echo 'tool 2.0'; exit 1", "tool 2.0", false},
		{"non-zero exit without output", "exit 1", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := systemEnv{}.Version(writeScript(t, tt.body))
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSystemEnvVersionNotExecutable(t *testing.T) {
	if _, err := (systemEnv{}).Version(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Error("expected an error for a missing binary")
	}
}

func TestExecuteWithoutArgs(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := Execute(BuildInfo{Version: "x"}, &stdout, &stderr); code != ExitOK {
		t.Errorf("exit %d, stderr %s", code, stderr.String())
	}
}

func TestExitError(t *testing.T) {
	e := &ExitError{Code: ExitMissing, Msg: "boom"}
	if e.Error() != "boom" {
		t.Errorf("Error() = %q", e.Error())
	}
}
