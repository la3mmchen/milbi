package main

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// binPath is the real milbi binary, built once with the same ldflags mechanism as the release.
var binPath string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "milbi-bin")
	if err != nil {
		panic(err)
	}
	binPath = filepath.Join(dir, "milbi")
	ldflags := "-X main.version=v9.9.9 -X main.commit=abc1234 -X main.date=2026-01-01"
	if out, err := exec.Command("go", "build", "-ldflags", ldflags, "-o", binPath, ".").CombinedOutput(); err != nil {
		_, _ = os.Stderr.Write(out)
		_ = os.RemoveAll(dir)
		os.Exit(1)
	}
	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

// fakeTools writes restic and rsync stand-ins into a new dir and returns it; restic rejects `--version` like the real one.
func fakeTools(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	tools := map[string]string{
		"restic": "#!/bin/sh\n[ \"$1\" = version ] || exit 1\necho 'restic 0.17.0 fake'\n",
		"rsync":  "#!/bin/sh\n[ \"$1\" = --version ] || exit 1\necho 'rsync  version 3.3.0 fake'\n",
	}
	for name, script := range tools {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(script), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// runBin executes the built binary with only the given PATH and returns exit code, stdout and stderr.
func runBin(t *testing.T, path string, args ...string) (int, string, string) {
	t.Helper()
	cmd := exec.Command(binPath, args...)
	cmd.Env = []string{"PATH=" + path, "HOME=" + t.TempDir()}
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	code := 0
	if err != nil {
		var ee *exec.ExitError
		if !errors.As(err, &ee) {
			t.Fatalf("running %v: %v", args, err)
		}
		code = ee.ExitCode()
	}
	return code, stdout.String(), stderr.String()
}

func TestBinaryExitCodes(t *testing.T) {
	tools := fakeTools(t)
	empty := t.TempDir()

	tests := []struct {
		name       string
		path       string
		args       []string
		wantCode   int
		wantStdout []string
		wantStderr []string
	}{
		{"version", empty, []string{"version"}, 0, []string{"milbi v9.9.9", "commit: abc1234", "date: 2026-01-01"}, nil},
		{"no args shows help", empty, nil, 0, []string{"Available Commands"}, nil},
		{"doctor with tools", tools, []string{"doctor"}, 0, []string{"restic 0.17.0 fake", "rsync  version 3.3.0 fake"}, nil},
		{"doctor without tools", empty, []string{"doctor"}, 3, []string{"restic MISSING", "rsync  MISSING"}, []string{"missing required binaries: restic, rsync"}},
		{"doctor with extra arg", tools, []string{"doctor", "extra"}, 1, nil, []string{"error:"}},
		{"unknown command", tools, []string{"nope"}, 1, nil, []string{"unknown command"}},
		{"unknown flag", tools, []string{"version", "--nope"}, 1, nil, []string{"unknown flag"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, stdout, stderr := runBin(t, tt.path, tt.args...)
			if code != tt.wantCode {
				t.Errorf("exit code %d, want %d\nstdout: %s\nstderr: %s", code, tt.wantCode, stdout, stderr)
			}
			for _, want := range tt.wantStdout {
				if !strings.Contains(stdout, want) {
					t.Errorf("stdout lacks %q:\n%s", want, stdout)
				}
			}
			for _, want := range tt.wantStderr {
				if !strings.Contains(stderr, want) {
					t.Errorf("stderr lacks %q:\n%s", want, stderr)
				}
			}
		})
	}
}
