package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const example = `version: 1
defaults:
  excludes: ["**/node_modules"]
  retention: { last: 3, daily: 7 }
repositories:
  local:
    url: /Volumes/Backup/restic
    password: file:~/.config/milbi/secrets/local
  b2:
    url: s3:https://s3.example.com/bucket
    password: file:secrets/b2
    env:
      AWS_ACCESS_KEY_ID: env:B2_KEY_ID
backups:
  documents:
    from: [~/Documents, rel]
    to: [local, b2]
    excludes: ["**/build"]
  configs:
    from: [~/.config]
    to: [b2]
    retention: { daily: 30 }
syncs:
  photos:
    from: ~/Pictures/
    to: nas.local:/volume1/photos/
`

func TestParseExample(t *testing.T) {
	t.Setenv("HOME", "/home/me")
	c, warns, err := Parse([]byte(example), "/etc/milbi")
	if err != nil {
		t.Fatal(err)
	}
	if len(warns) != 4 {
		t.Errorf("warnings = %v, want 4 missing paths", warns)
	}
	if got := c.Syncs["photos"]; got.From != "/home/me/Pictures/" || got.To != "nas.local:/volume1/photos/" {
		t.Errorf("sync = %+v", got)
	}
	if got := c.Backups["documents"].From; got[0] != "/home/me/Documents" || got[1] != "/etc/milbi/rel" {
		t.Errorf("from = %v", got)
	}
	if got := c.Repositories["b2"].URL; got != "s3:https://s3.example.com/bucket" {
		t.Errorf("url = %s", got)
	}
	if got := c.ExcludesFor(c.Backups["documents"]); strings.Join(got, ",") != "**/node_modules,**/build" {
		t.Errorf("excludes = %v", got)
	}
	if got := c.RetentionFor(c.Backups["configs"]); len(got) != 1 || got["daily"] != 30 {
		t.Errorf("retention = %v", got)
	}
	if got := c.RetentionFor(c.Backups["documents"]); got["last"] != 3 {
		t.Errorf("default retention = %v", got)
	}
}

func TestParseErrors(t *testing.T) {
	base := "repositories:\n  r:\n    url: /r\n    password: env:P\n"
	cases := map[string]struct{ yaml, want string }{
		"no version":      {"backups: {}\n", "version is missing"},
		"bad version":     {"version: 2\n", "version 2 is not supported"},
		"unknown field":   {"version: 1\nbackups:\n  a:\n    pathz: [x]\n", "line 4: field pathz not found"},
		"syntax":          {"version: [\n", "yaml:"},
		"bad repo name":   {"version: 1\nrepositories:\n  Bad:\n    url: /r\n    password: env:P\n", `repository name "Bad"`},
		"no url":          {"version: 1\nrepositories:\n  r:\n    password: env:P\n", "url is missing"},
		"no password":     {"version: 1\nrepositories:\n  r:\n    url: /r\n", "password is missing"},
		"empty file ref":  {"version: 1\nrepositories:\n  r:\n    url: /r\n    password: \"file:\"\n", "line 5: repository r: password has an empty file: reference"},
		"empty ref":       {"version: 1\nrepositories:\n  r:\n    url: /r\n    password: \"env:\"\n", "empty env: reference"},
		"bad env key":     {"version: 1\n" + base + "    env:\n      1X: env:A\n", "not a valid variable name"},
		"restic pw env":   {"version: 1\n" + base + "    env:\n      RESTIC_PASSWORD_FILE: env:A\n", "must not set RESTIC_PASSWORD_FILE"},
		"empty env ref":   {"version: 1\n" + base + "    env:\n      A: \"env:\"\n", "env A has an empty env: reference"},
		"name collision":  {"version: 1\n" + base + "backups:\n  x:\n    from: [/a]\n    to: [r]\n    retention: {last: 1}\nsyncs:\n  x:\n    from: /a\n    to: /b\n", "name is also used by a sync"},
		"no from":         {"version: 1\n" + base + "backups:\n  x:\n    to: [r]\n    retention: {last: 1}\n", "from is missing"},
		"no to":           {"version: 1\n" + base + "backups:\n  x:\n    from: [/a]\n    retention: {last: 1}\n", "to is missing"},
		"unknown repo":    {"version: 1\n" + base + "backups:\n  x:\n    from: [/a]\n    to: [nope]\n    retention: {last: 1}\n", `repository "nope" does not exist`},
		"repo twice":      {"version: 1\n" + base + "backups:\n  x:\n    from: [/a]\n    to: [r, r]\n    retention: {last: 1}\n", "listed twice"},
		"bad retention":   {"version: 1\n" + base + "backups:\n  x:\n    from: [/a]\n    to: [r]\n    retention: {forever: 1}\n", `retention key "forever" is unknown`},
		"negative":        {"version: 1\ndefaults:\n  retention: {last: -1}\n", "defaults: retention last is negative"},
		"keeps nothing":   {"version: 1\n" + base + "backups:\n  x:\n    from: [/a]\n    to: [r]\n    retention: {last: 0}\n", "retention keeps nothing"},
		"no retention":    {"version: 1\n" + base + "backups:\n  x:\n    from: [/a]\n    to: [r]\n", "retention keeps nothing"},
		"sync no from":    {"version: 1\nsyncs:\n  s:\n    to: /b\n", "sync s: from is missing"},
		"sync no to":      {"version: 1\nsyncs:\n  s:\n    from: /a\n", "sync s: to is missing"},
		"bad sync name":   {"version: 1\nsyncs:\n  s_1:\n    from: /a\n    to: /b\n", `sync name "s_1"`},
		"bad backup name": {"version: 1\n" + base + "backups:\n  X:\n    from: [/a]\n    to: [r]\n    retention: {last: 1}\n", `backup name "X"`},
		"s3 sync":         {"version: 1\nsyncs:\n  s:\n    from: /a\n    to: s3:https://x/b\n", "line 5: sync s: target \"s3:https://x/b\" is not supported yet"},
		"scheme sync":     {"version: 1\nsyncs:\n  s:\n    from: /a\n    to: sftp://h/p\n", "is not supported yet"},
		"b2 no bucket":    {"version: 1\nsyncs:\n  s:\n    from: /a\n    to: b2:///p\n    b2_key_id: env:I\n    b2_key: env:K\n", "b2 target has no bucket"},
		"b2 no key":       {"version: 1\nsyncs:\n  s:\n    from: /a\n    to: b2://bkt/p\n    b2_key_id: env:I\n", "sync s: b2_key is missing"},
		"b2 empty key":    {"version: 1\nsyncs:\n  s:\n    from: /a\n    to: b2://bkt/p\n    b2_key_id: env:I\n    b2_key: \"file:\"\n", "line 7: sync s: b2_key has an empty file: reference"},
		"b2 key on rsync": {"version: 1\nsyncs:\n  s:\n    from: /a\n    to: /b\n    b2_key_id: env:I\n", "b2_key_id is only allowed on b2:// targets"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, _, err := Parse([]byte(tc.yaml), "/")
			var ce *Error
			if !errors.As(err, &ce) {
				t.Fatalf("err = %v, want *Error", err)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("err = %q, want %q", err, tc.want)
			}
			if strings.Contains(err.Error(), "hunter2") {
				t.Error("error echoes the secret")
			}
		})
	}
}

func TestParseEmpty(t *testing.T) {
	if _, _, err := Parse(nil, "/"); err == nil || !strings.Contains(err.Error(), "version is missing") {
		t.Errorf("err = %v", err)
	}
}

func TestLoad(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "c.yaml")
	if err := os.WriteFile(p, []byte("version: 1\nsyncs:\n  s:\n    from: data/\n    to: host:x\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	c, warns, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if got := c.Syncs["s"].From; got != filepath.Join(dir, "data")+"/" {
		t.Errorf("from = %s", got)
	}
	if len(warns) != 1 {
		t.Errorf("warnings = %v", warns)
	}
	if _, _, err := Load(filepath.Join(dir, "missing.yaml")); err == nil {
		t.Error("missing file: no error")
	}
}

func TestPath(t *testing.T) {
	t.Setenv("HOME", "/home/me")
	t.Setenv("MILBI_CONFIG", "")
	if p, _ := Path("/x.yaml"); p != "/x.yaml" {
		t.Errorf("flag: %s", p)
	}
	if p, _ := Path(""); p != "/home/me/.config/milbi/config.yaml" {
		t.Errorf("default: %s", p)
	}
	t.Setenv("MILBI_CONFIG", "/env.yaml")
	if p, _ := Path(""); p != "/env.yaml" {
		t.Errorf("env: %s", p)
	}
}

func TestExpandPath(t *testing.T) {
	t.Setenv("HOME", "/home/me")
	t.Setenv("VOL", "/Volumes/x")
	for in, want := range map[string]string{
		"~":            "/home/me",
		"$VOL/a/":      "/Volumes/x/a/",
		"rel/":         "/cfg/rel/",
		"host:~/p/":    "host:~/p/",
		"":             "",
		"/abs/./path/": "/abs/./path/",
	} {
		if got := ExpandPath(in, "/cfg"); got != want {
			t.Errorf("ExpandPath(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestResolveSecret(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "s"), []byte("pw\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "open"), []byte("pw"), 0o644); err != nil {
		t.Fatal(err)
	}
	if v, w, err := ResolveSecret("file:s", dir); err != nil || v != "pw" || w != "" {
		t.Errorf("file: %q %q %v", v, w, err)
	}
	if _, w, err := ResolveSecret("file:open", dir); err != nil || !strings.Contains(w, "0644") {
		t.Errorf("open file: %q %v", w, err)
	}
	if _, _, err := ResolveSecret("file:missing", dir); err == nil {
		t.Error("missing file: no error")
	}
	if err := os.Mkdir(filepath.Join(dir, "d"), 0o700); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ResolveSecret("file:d", dir); err == nil {
		t.Error("directory: no error")
	}
	t.Setenv("MILBI_TEST_PW", "envpw")
	if v, _, err := ResolveSecret("env:MILBI_TEST_PW", dir); err != nil || v != "envpw" {
		t.Errorf("env: %q %v", v, err)
	}
	if _, _, err := ResolveSecret("env:MILBI_TEST_UNSET", dir); err == nil {
		t.Error("unset env: no error")
	}
	if v, w, err := ResolveSecret("hunter2", dir); err != nil || v != "hunter2" || w != "" {
		t.Errorf("literal: %q %q %v", v, w, err)
	}
	if _, _, err := ResolveSecret("env:", dir); err == nil {
		t.Error("empty env: no error")
	}
}
