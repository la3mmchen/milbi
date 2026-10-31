package backup

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/la3mmchen/milbi/internal/config"
)

// fakeRestic records each call's argv to $FAKE_DIR/argv and its RESTIC_* env to $FAKE_DIR/env.
// FAKE_<SUBCOMMAND>_EXIT sets a subcommand's exit code.
const fakeRestic = `#!/bin/sh
echo "$*" >> "$FAKE_DIR/argv"
env | grep '^RESTIC_\|^B2_' | sort >> "$FAKE_DIR/env"
case "$1" in
backup)
  echo '{"message_type":"status","percent_done":0.5}'
  echo 'not json'
  echo '{"message_type":"summary","files_new":2,"files_changed":1,"data_added":2048,"snapshot_id":"0123456789abcdef"}'
  echo "backup stderr" >&2
  exit ${FAKE_BACKUP_EXIT:-0} ;;
forget) exit ${FAKE_FORGET_EXIT:-0} ;;
cat) exit ${FAKE_CAT_EXIT:-1} ;;
init) echo "created restic repository" ;;
snapshots) echo "ID  Time  Tags"; exit ${FAKE_SNAPSHOTS_EXIT:-0} ;;
prune) echo "pruned" ;;
esac
`

const testConfig = `version: 1
defaults:
  excludes: ["*.tmp"]
repositories:
  local: {url: /repo/local, password: s3cret-pw}
  other: {url: /repo/other, password: s3cret-pw, env: {AWS_KEY: env:FAKE_AWS}}
  broken: {url: /repo/broken, password: "env:MILBI_TEST_UNSET"}
backups:
  docs: {from: [/src/docs], to: [local, other], retention: {last: 2, daily: 0}}
  pics: {from: [/src/pics], to: [local], retention: {last: 1}}
  bad: {from: [/src/bad], to: [broken], retention: {last: 1}}
`

type fixture struct {
	r       *Runner
	fake    string
	out     *bytes.Buffer
	err     *bytes.Buffer
	nowTick time.Time
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	dir := t.TempDir()
	fake := filepath.Join(dir, "fake")
	if err := os.Mkdir(fake, 0o755); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(dir, "restic")
	if err := os.WriteFile(bin, []byte(fakeRestic), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FAKE_DIR", fake)
	t.Setenv("FAKE_AWS", "aws-value")
	t.Setenv("RESTIC_PASSWORD_FILE", "/inherited/must-not-leak")
	c, _, err := config.Parse([]byte(testConfig), dir)
	if err != nil {
		t.Fatal(err)
	}
	f := &fixture{fake: fake, out: &bytes.Buffer{}, err: &bytes.Buffer{}, nowTick: time.Date(2026, 10, 28, 9, 0, 0, 0, time.UTC)}
	f.r = &Runner{Config: c, Restic: bin, StateDir: filepath.Join(dir, "state"), Out: f.out, Err: f.err,
		Now: func() time.Time { f.nowTick = f.nowTick.Add(time.Second); return f.nowTick }}
	return f
}

func (f *fixture) read(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(f.fake, name))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Fatal(err)
	}
	return string(data)
}

func TestBackupArgs(t *testing.T) {
	got := strings.Join(BackupArgs("docs", []string{"*.tmp", "cache"}, []string{"/a", "/b c"}), "|")
	want := "backup|--json|--tag|milbi|--tag|job:docs|--exclude|*.tmp|--exclude|cache|/a|/b c"
	if got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestForgetArgs(t *testing.T) {
	tests := []struct {
		keep config.Retention
		want string
	}{
		{config.Retention{"last": 2, "daily": 7, "weekly": 0}, "forget|--tag|job:docs|--group-by|host,tags|--keep-daily|7|--keep-last|2"},
		{config.Retention{"last": 0}, ""},
		{nil, ""},
	}
	for _, tc := range tests {
		if got := strings.Join(ForgetArgs("docs", tc.keep), "|"); got != tc.want {
			t.Errorf("%v: got %s, want %s", tc.keep, got, tc.want)
		}
	}
}

func TestRunSuccess(t *testing.T) {
	f := newFixture(t)
	res := f.r.Run("docs")
	if res.Status != StatusSuccess || res.Error != "" {
		t.Fatalf("got %+v", res)
	}
	if len(res.Stats) != 2 || res.Stats[1].Repository != "other" || res.Stats[0].SnapshotID != "0123456789abcdef" ||
		res.Stats[0].FilesNew != 2 || res.Stats[0].FilesChanged != 1 || res.Stats[0].BytesAdded != 2048 {
		t.Errorf("stats: %+v", res.Stats)
	}
	argv := f.read(t, "argv")
	wantArgv := "backup --json --tag milbi --tag job:docs --exclude *.tmp /src/docs\n" +
		"forget --tag job:docs --group-by host,tags --keep-last 2\n"
	if argv != wantArgv+wantArgv {
		t.Errorf("argv:\n%s", argv)
	}
	env := f.read(t, "env")
	for _, want := range []string{"RESTIC_PASSWORD=s3cret-pw", "RESTIC_REPOSITORY=/repo/local", "RESTIC_REPOSITORY=/repo/other"} {
		if !strings.Contains(env, want) {
			t.Errorf("env lacks %s:\n%s", want, env)
		}
	}
	if strings.Contains(env, "RESTIC_PASSWORD_FILE") {
		t.Errorf("inherited RESTIC_PASSWORD_FILE leaked:\n%s", env)
	}
	st, err := ReadState(f.r.StateDir, "docs")
	if err != nil {
		t.Fatal(err)
	}
	if st.Status != StatusSuccess || st.EndedAt == nil || st.LastSuccessAt == nil || st.DurationSeconds <= 0 || st.Log != res.Log {
		t.Errorf("state: %+v", st)
	}
	log, err := os.ReadFile(res.Log)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(log), "percent_done") || !strings.Contains(string(log), "backup stderr") || !strings.Contains(string(log), `"summary"`) {
		t.Errorf("log:\n%s", log)
	}
	state, _ := os.ReadFile(filepath.Join(f.r.StateDir, "status", "docs.json"))
	for name, data := range map[string]string{"argv": argv, "log": string(log), "state": string(state)} {
		if strings.Contains(data, "s3cret-pw") {
			t.Errorf("password in %s", name)
		}
	}
}

func TestRunOutcomes(t *testing.T) {
	tests := []struct {
		name, job, env string
		status         Status
		code           int
		errPart        string
		forgets        int
	}{
		{"unreadable files", "pics", "FAKE_BACKUP_EXIT=3", StatusWarning, 3, "could not be read", 0},
		{"backup fails", "pics", "FAKE_BACKUP_EXIT=1", StatusFailed, 1, "backup: exit status 1", 0},
		{"forget fails", "pics", "FAKE_FORGET_EXIT=1", StatusFailed, 1, "forget: exit status 1", 1},
		{"secret unresolved", "bad", "", StatusFailed, 0, "MILBI_TEST_UNSET is not set", 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			if k, v, ok := strings.Cut(tc.env, "="); ok {
				t.Setenv(k, v)
			}
			res := f.r.Run(tc.job)
			if res.Status != tc.status || res.ExitCode != tc.code || !strings.Contains(res.Error, tc.errPart) {
				t.Errorf("got %+v", res)
			}
			if n := strings.Count(f.read(t, "argv"), "forget"); n != tc.forgets {
				t.Errorf("forget ran %d times", n)
			}
			st, err := ReadState(f.r.StateDir, tc.job)
			if err != nil || st.Status != tc.status {
				t.Errorf("state %+v, %v", st, err)
			}
			if (st.LastSuccessAt != nil) != (tc.status == StatusWarning) {
				t.Errorf("lastSuccessAt: %v", st.LastSuccessAt)
			}
		})
	}
}

func TestRunKeepsLastSuccess(t *testing.T) {
	f := newFixture(t)
	first := f.r.Run("pics")
	t.Setenv("FAKE_BACKUP_EXIT", "1")
	_ = f.r.Run("pics")
	st, err := ReadState(f.r.StateDir, "pics")
	if err != nil {
		t.Fatal(err)
	}
	if first.Status != StatusSuccess || st.Status != StatusFailed || st.LastSuccessAt == nil {
		t.Errorf("state: %+v", st)
	}
}

func TestRunLocked(t *testing.T) {
	f := newFixture(t)
	held, err := f.r.lock("pics")
	if err != nil {
		t.Fatal(err)
	}
	res := f.r.Run("pics")
	_ = held.Close()
	if res.Status != StatusSkipped || f.read(t, "argv") != "" {
		t.Errorf("got %+v", res)
	}
	if _, err := ReadState(f.r.StateDir, "pics"); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("skipped run wrote state: %v", err)
	}
	if res := f.r.Run("pics"); res.Status != StatusSuccess {
		t.Errorf("after release: %+v", res)
	}
}

func TestRunStateDirErrors(t *testing.T) {
	f := newFixture(t)
	blocker := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocker, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	f.r.StateDir = blocker // not a directory: lock fails
	if res := f.r.Run("pics"); res.Status != StatusFailed {
		t.Errorf("lock: %+v", res)
	}
	f.r.StateDir = t.TempDir()
	if err := os.WriteFile(filepath.Join(f.r.StateDir, "logs"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if res := f.r.Run("pics"); res.Status != StatusFailed {
		t.Errorf("log: %+v", res)
	}
	f.r.StateDir = t.TempDir()
	if err := os.WriteFile(filepath.Join(f.r.StateDir, "status"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if res := f.r.Run("pics"); res.Status != StatusFailed {
		t.Errorf("state: %+v", res)
	}
}

func TestTrimLogs(t *testing.T) {
	f := newFixture(t)
	dir := filepath.Join(f.r.StateDir, "logs", "pics")
	if err := os.MkdirAll(filepath.Join(dir, "sub"), 0o700); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < KeepLogs+2; i++ {
		if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("%03d.log", i)), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	f.r.trimLogs("pics")
	entries, _ := os.ReadDir(dir)
	if len(entries) != KeepLogs+1 { // the logs plus sub/
		t.Errorf("%d entries left", len(entries))
	}
	if _, err := os.Stat(filepath.Join(dir, "000.log")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("oldest log kept: %v", err)
	}
	f.r.trimLogs("missing") // no dir: no panic
}

func TestInit(t *testing.T) {
	f := newFixture(t)
	if err := f.r.Init("local"); err != nil || !strings.Contains(f.out.String(), "created") {
		t.Fatalf("init: %v, %s", err, f.out)
	}
	t.Setenv("FAKE_CAT_EXIT", "0")
	if err := f.r.Init("local"); err == nil || !strings.Contains(err.Error(), "already initialized") {
		t.Errorf("second init: %v", err)
	}
	if err := f.r.Init("nope"); err == nil {
		t.Error("unknown repository: want error")
	}
}

func TestPrune(t *testing.T) {
	f := newFixture(t)
	if err := f.r.Prune("local"); err != nil || f.read(t, "argv") != "prune\n" {
		t.Errorf("prune: %v, %s", err, f.read(t, "argv"))
	}
	if err := f.r.Prune("broken"); err == nil {
		t.Error("broken repository: want error")
	}
}

func TestSnapshots(t *testing.T) {
	f := newFixture(t)
	if err := f.r.Snapshots("docs"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(f.out.String(), "repository other:") || strings.Count(f.read(t, "argv"), "snapshots --tag job:docs") != 2 {
		t.Errorf("out %s argv %s", f.out, f.read(t, "argv"))
	}
	t.Setenv("FAKE_SNAPSHOTS_EXIT", "1")
	if err := f.r.Snapshots("pics"); err == nil || !strings.Contains(err.Error(), "repository local") {
		t.Errorf("failing restic: %v", err)
	}
	if err := f.r.Snapshots("bad"); err == nil {
		t.Error("unresolved secret: want error")
	}
}

func TestDryRun(t *testing.T) {
	f := newFixture(t)
	f.r.DryRun("docs")
	f.r.Config.Backups["pics"] = config.Backup{From: []string{"/p"}, To: []string{"local"}, Retention: config.Retention{"last": 0}}
	f.r.DryRun("pics")
	want := "docs [local]: restic backup --json --tag milbi --tag job:docs --exclude *.tmp /src/docs\n" +
		"docs [local]: restic forget --tag job:docs --group-by host,tags --keep-last 2\n" +
		"docs [other]: restic backup --json --tag milbi --tag job:docs --exclude *.tmp /src/docs\n" +
		"docs [other]: restic forget --tag job:docs --group-by host,tags --keep-last 2\n" +
		"pics [local]: restic backup --json --tag milbi --tag job:pics --exclude *.tmp /p\n"
	if f.out.String() != want || f.read(t, "argv") != "" {
		t.Errorf("got:\n%s", f.out)
	}
}

func TestSecretWarning(t *testing.T) {
	f := newFixture(t)
	pw := filepath.Join(t.TempDir(), "pw")
	if err := os.WriteFile(pw, []byte("filepw\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	f.r.Config.Repositories["local"] = config.Repository{URL: "/r", Password: "file:" + pw}
	env, err := f.r.repoEnv("local")
	if err != nil || !strings.Contains(strings.Join(env, "\n"), "RESTIC_PASSWORD=filepw") || !strings.Contains(f.err.String(), "broader than 0600") {
		t.Errorf("env %v err %v stderr %s", env, err, f.err)
	}
}

func TestDefaultStateDir(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", "/xdg")
	if d, _ := DefaultStateDir(); d != "/xdg/milbi" {
		t.Errorf("xdg: %s", d)
	}
	t.Setenv("XDG_STATE_HOME", "")
	t.Setenv("HOME", "/home/u")
	if d, _ := DefaultStateDir(); d != "/home/u/.local/state/milbi" {
		t.Errorf("home: %s", d)
	}
	t.Setenv("HOME", "")
	if _, err := DefaultStateDir(); err == nil {
		t.Error("no HOME: want error")
	}
}

func TestReadStateInvalid(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "status"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "status", "x.json"), []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadState(dir, "x"); err == nil {
		t.Error("broken JSON: want error")
	}
}

func TestFormatBytes(t *testing.T) {
	for n, want := range map[int64]string{0: "0 B", 1023: "1023 B", 2048: "2.0 KiB", 5 << 20: "5.0 MiB", 3 << 30: "3.0 GiB"} {
		if got := FormatBytes(n); got != want {
			t.Errorf("%d: got %s, want %s", n, got, want)
		}
	}
}

func TestExitCode(t *testing.T) {
	if exitCode(nil) != 0 || exitCode(errors.New("x")) != 0 {
		t.Error("non-exit errors map to 0")
	}
}
