package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
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
		wantErr string // substring of the error, empty means no error
	}{
		{"first line only", "printf 'tool 1.0\\nsecond line\\n'", "tool 1.0", ""},
		{"non-zero exit with output is an error that keeps the first line", "echo 'broken: no libc'; exit 1", "", "broken: no libc"},
		{"non-zero exit without output", "exit 1", "", "exit status 1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := systemEnv{}.Version(writeScript(t, tt.body), "--version")
			if (err != nil) != (tt.wantErr != "") {
				t.Fatalf("err = %v, wantErr %q", err, tt.wantErr)
			}
			if err != nil && !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("err %q lacks %q", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSystemEnvVersionPassesArg(t *testing.T) {
	got, err := systemEnv{}.Version(writeScript(t, `echo "args: $*"`), "version")
	if err != nil || got != "args: version" {
		t.Errorf("got %q, err %v", got, err)
	}
}

func TestSystemEnvVersionNotExecutable(t *testing.T) {
	if _, err := (systemEnv{}).Version(filepath.Join(t.TempDir(), "missing"), "--version"); err == nil {
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
