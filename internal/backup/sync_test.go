package backup

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/la3mmchen/milbi/internal/config"
)

// fakeRsync records argv to $FAKE_DIR/rsync-argv; FAKE_RSYNC_EXIT makes it fail with a message.
const fakeRsync = `#!/bin/sh
echo "$*" >> "$FAKE_DIR/rsync-argv"
if [ -n "$FAKE_RSYNC_EXIT" ]; then
  [ -z "$FAKE_RSYNC_SILENT" ] && echo "rsync error: some files vanished" >&2
  exit "$FAKE_RSYNC_EXIT"
fi
echo "Number of files transferred: 7"
echo "Total bytes sent: 2.048"
`

const syncConfig = `version: 1
syncs:
  keep: {from: /src/, to: /dst}
  mirror: {from: /src, to: "host:/dst", delete: true, excludes: ["*.tmp"]}
  photos: {from: /src/, to: "b2://bucket/p", b2_key_id: env:X, b2_key: env:Y}
`

func newSyncRunner(t *testing.T) (*Runner, string, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	dir := t.TempDir()
	bin := filepath.Join(dir, "rsync")
	if err := os.WriteFile(bin, []byte(fakeRsync), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FAKE_DIR", dir)
	c, _, err := config.Parse([]byte(syncConfig), dir)
	if err != nil {
		t.Fatal(err)
	}
	out, errOut := &bytes.Buffer{}, &bytes.Buffer{}
	r := &Runner{Config: c, Rsync: bin, StateDir: filepath.Join(dir, "state"), Out: out, Err: errOut, Now: time.Now}
	return r, dir, out, errOut
}

func TestRsyncArgs(t *testing.T) {
	tests := []struct {
		s      config.Sync
		dryRun bool
		want   string
	}{
		{config.Sync{From: "/a/", To: "/b"}, false, "-a|--partial|--stats|/a/|/b"},
		{config.Sync{From: "/a", To: "h:/b", Delete: true, Excludes: []string{"*.tmp"}}, true,
			"-a|--partial|--stats|-n|-v|--delete|--exclude|*.tmp|/a|h:/b"},
	}
	for _, tc := range tests {
		if got := strings.Join(RsyncArgs(tc.s, tc.dryRun), "|"); got != tc.want {
			t.Errorf("got %s, want %s", got, tc.want)
		}
	}
}

func TestParseRsyncStats(t *testing.T) {
	tests := map[string]SyncStats{
		"Number of regular files transferred: 1,234\nTotal bytes sent: 5,678,901\n": {1234, 5678901},
		"Number of files transferred: 3\nTotal bytes sent: 1.024\n":                 {3, 1024},
		"Number of files transferred: 2\nTotal sent: 118 B\n":                       {2, 118},
		"Total sent: 1.2 KB\n": {},
		"nothing here":         {},
	}
	for in, want := range tests {
		if got := ParseRsyncStats(in); got != want {
			t.Errorf("%q: got %+v, want %+v", in, got, want)
		}
	}
}

func TestRunSync(t *testing.T) {
	r, dir, _, _ := newSyncRunner(t)
	res := r.Run("mirror")
	if res.Status != StatusSuccess || res.Sync == nil || *res.Sync != (SyncStats{7, 2048}) {
		t.Fatalf("got %+v", res)
	}
	argv, _ := os.ReadFile(filepath.Join(dir, "rsync-argv"))
	if string(argv) != "-a --partial --stats --delete --exclude *.tmp /src host:/dst\n" {
		t.Errorf("argv: %q", argv)
	}
	st, err := ReadState(r.StateDir, "mirror")
	if err != nil || st.Status != StatusSuccess || st.Sync == nil || st.Sync.FilesTransferred != 7 || st.LastSuccessAt == nil {
		t.Errorf("state %+v, err %v", st, err)
	}
	log, _ := os.ReadFile(st.Log)
	if !strings.Contains(string(log), "$ rsync -a --partial --stats --delete") {
		t.Errorf("log: %s", log)
	}
}

func TestRunSyncFailure(t *testing.T) {
	r, _, _, _ := newSyncRunner(t)
	t.Setenv("FAKE_RSYNC_EXIT", "23")
	res := r.Run("keep")
	if res.Status != StatusFailed || res.ExitCode != 23 || res.Error != "rsync: rsync error: some files vanished" {
		t.Errorf("got %+v", res)
	}
	t.Setenv("FAKE_RSYNC_SILENT", "1")
	if res := r.Run("keep"); res.Error != "rsync: exit status 23" {
		t.Errorf("silent failure: %+v", res)
	}
	st, _ := ReadState(r.StateDir, "keep")
	if st.Status != StatusFailed || st.ExitCode != 23 {
		t.Errorf("state: %+v", st)
	}
}

func TestRunSyncCannotStart(t *testing.T) {
	r, _, _, _ := newSyncRunner(t)
	if res := r.Run("photos"); res.Status != StatusFailed || !strings.Contains(res.Error, "b2 syncs are not supported yet") {
		t.Errorf("b2: %+v", res)
	}
	r.Rsync = ""
	if res := r.Run("keep"); res.Status != StatusFailed || res.Error != "rsync not found on PATH" {
		t.Errorf("no rsync: %+v", res)
	}
	if err := r.DrySync("keep"); err == nil {
		t.Error("DrySync without rsync: want error")
	}
}

func TestDrySync(t *testing.T) {
	r, dir, out, _ := newSyncRunner(t)
	if err := r.DrySync("keep"); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(out.String(), "keep: rsync -a --partial --stats -n -v /src/ /dst\n") {
		t.Errorf("out: %s", out)
	}
	argv, _ := os.ReadFile(filepath.Join(dir, "rsync-argv"))
	if string(argv) != "-a --partial --stats -n -v /src/ /dst\n" {
		t.Errorf("argv: %q", argv)
	}
	if _, err := os.Stat(r.StateDir); !os.IsNotExist(err) {
		t.Errorf("dry run wrote state: %v", err)
	}
}
