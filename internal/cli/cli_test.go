package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fakeEnv struct {
	paths    map[string]string
	versions map[string]string
}

func (f fakeEnv) LookPath(name string) (string, error) {
	if p, ok := f.paths[name]; ok {
		return p, nil
	}
	return "", errors.New("not found")
}

func (f fakeEnv) Version(path, _ string) (string, error) {
	if v, ok := f.versions[path]; ok {
		return v, nil
	}
	return "", errors.New("no version")
}

func run(t *testing.T, e env, args ...string) (int, string, string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	root := newRootCmd(BuildInfo{Version: "1.2.3", Commit: "abc", Date: "2026-01-01"}, e, &stdout, &stderr)
	root.SetArgs(args)
	code := execute(root, &stderr)
	return code, stdout.String(), stderr.String()
}

func TestVersion(t *testing.T) {
	code, out, _ := run(t, fakeEnv{}, "version")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	for _, want := range []string{"1.2.3", "abc", "2026-01-01"} {
		if !strings.Contains(out, want) {
			t.Errorf("output %q lacks %q", out, want)
		}
	}
}

func TestDoctorOK(t *testing.T) {
	e := fakeEnv{
		paths:    map[string]string{"restic": "/bin/restic", "rsync": "/bin/rsync"},
		versions: map[string]string{"/bin/restic": "restic 0.17.0", "/bin/rsync": "rsync  version 3.3.0"},
	}
	code, out, _ := run(t, e, "doctor")
	if code != 0 {
		t.Fatalf("exit %d, out %s", code, out)
	}
	if strings.Contains(out, "warning") {
		t.Errorf("unexpected warning: %s", out)
	}
}

func TestDoctorMissingRestic(t *testing.T) {
	e := fakeEnv{
		paths:    map[string]string{"rsync": "/bin/rsync"},
		versions: map[string]string{"/bin/rsync": "rsync  version 3.3.0"},
	}
	code, out, errOut := run(t, e, "doctor")
	if code != ExitMissing {
		t.Fatalf("exit %d, want 3", code)
	}
	if !strings.Contains(out, "MISSING") || !strings.Contains(errOut, "restic") {
		t.Errorf("out %q err %q", out, errOut)
	}
}

func TestRunMissingRestic(t *testing.T) {
	cfg := filepath.Join(t.TempDir(), "milbi.yaml")
	src := "version: 1\nrepositories:\n  r: {url: /r, password: env:X}\nbackups:\n  b: {from: [/a], to: [r], retention: {last: 1}}\n"
	if err := os.WriteFile(cfg, []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"run", "b"}, {"init", "r"}, {"snapshots", "b"}, {"prune", "r"}} {
		code, _, errOut := run(t, fakeEnv{}, append([]string{"--config", cfg}, args...)...)
		if code != ExitMissing || !strings.Contains(errOut, "restic not found") {
			t.Errorf("%v: exit %d, err %q", args, code, errOut)
		}
	}
}

func TestShort(t *testing.T) {
	for id, want := range map[string]string{"": "-", "abc": "abc", "0123456789": "01234567"} {
		if got := short(id); got != want {
			t.Errorf("%q: got %q", id, got)
		}
	}
}

func TestDoctorOpenrsyncWarns(t *testing.T) {
	e := fakeEnv{
		paths:    map[string]string{"restic": "/bin/restic", "rsync": "/bin/rsync"},
		versions: map[string]string{"/bin/restic": "restic 0.17.0", "/bin/rsync": "openrsync: protocol version 29"},
	}
	code, out, _ := run(t, e, "doctor")
	if code != 0 || !strings.Contains(out, "openrsync") || !strings.Contains(out, "warning") {
		t.Errorf("exit %d out %s", code, out)
	}
}

func TestDoctorVersionError(t *testing.T) {
	e := fakeEnv{paths: map[string]string{"restic": "/bin/restic", "rsync": "/bin/rsync"}}
	code, out, _ := run(t, e, "doctor")
	if code != 0 || !strings.Contains(out, "version unknown") {
		t.Errorf("exit %d out %s", code, out)
	}
}

func TestUnknownCommandFails(t *testing.T) {
	code, _, _ := run(t, fakeEnv{}, "nope")
	if code != ExitFailure {
		t.Errorf("exit %d, want 1", code)
	}
}
