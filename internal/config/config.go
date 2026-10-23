// Package config loads and validates milbi's YAML config (spec 04).
package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"
)

// SupportedVersion is the only config version this milbi understands.
const SupportedVersion = 1

// Config is the whole config file.
type Config struct {
	Version      int                   `yaml:"version"`
	Defaults     Defaults              `yaml:"defaults"`
	Repositories map[string]Repository `yaml:"repositories"`
	Backups      map[string]Backup     `yaml:"backups"`
	Syncs        map[string]Sync       `yaml:"syncs"`

	// Dir is the config file's directory; relative paths resolve against it.
	Dir string `yaml:"-"`
}

// Defaults apply to every backup.
type Defaults struct {
	Excludes  []string  `yaml:"excludes"`
	Retention Retention `yaml:"retention"`
}

// Retention maps restic's --keep-* policies (last, daily, …) to counts.
type Retention map[string]int

// Repository is a restic repository shared by several backups.
type Repository struct {
	URL      string            `yaml:"url"`
	Password string            `yaml:"password"`
	Env      map[string]string `yaml:"env"`
}

// Backup is a restic job from local paths to one or more repositories.
type Backup struct {
	From      []string  `yaml:"from"`
	To        []string  `yaml:"to"`
	Excludes  []string  `yaml:"excludes"`
	Retention Retention `yaml:"retention"`
}

// Sync is a plain-file mirror: an rsync job, or a `b2 sync` when To starts with B2Prefix.
// B2KeyID and B2Key are secret references, set only on b2 syncs.
type Sync struct {
	From    string `yaml:"from"`
	To      string `yaml:"to"`
	Delete  bool   `yaml:"delete"`
	B2KeyID string `yaml:"b2_key_id"`
	B2Key   string `yaml:"b2_key"`
}

// B2Prefix marks a sync target in Backblaze B2.
const B2Prefix = "b2://"

// IsB2 reports whether the sync mirrors into Backblaze B2.
func (s Sync) IsB2() bool { return strings.HasPrefix(s.To, B2Prefix) }

// urlSchemes are tool prefixes that look like `host:path` but name a backend milbi does not support.
var urlSchemes = []string{"s3", "b2", "gs", "azure", "swift", "rest", "rclone", "sftp"}

// isUnsupportedURL reports whether a sync target is URL-like but not a b2:// target.
func isUnsupportedURL(to string) bool {
	if strings.HasPrefix(to, B2Prefix) {
		return false
	}
	if strings.Contains(to, "://") {
		return true
	}
	for _, s := range urlSchemes {
		if strings.HasPrefix(to, s+":") {
			return true
		}
	}
	return false
}

// Problem is one validation error, with the line it refers to (0 if unknown).
type Problem struct {
	Line int
	Msg  string
}

func (p Problem) String() string {
	if p.Line > 0 {
		return fmt.Sprintf("line %d: %s", p.Line, p.Msg)
	}
	return p.Msg
}

// Error collects every problem found in a config.
type Error struct {
	Problems []Problem
}

func (e *Error) Error() string {
	lines := make([]string, len(e.Problems))
	for i, p := range e.Problems {
		lines[i] = p.String()
	}
	return "invalid config:\n  " + strings.Join(lines, "\n  ")
}

var (
	namePattern   = regexp.MustCompile(`^[a-z0-9-]+$`)
	envKeyPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	retentionKeys = map[string]bool{"last": true, "hourly": true, "daily": true, "weekly": true, "monthly": true, "yearly": true}
)

// Path picks the config file: the flag value, else $MILBI_CONFIG, else ~/.config/milbi/config.yaml.
func Path(flag string) (string, error) {
	if flag != "" {
		return flag, nil
	}
	if p := os.Getenv("MILBI_CONFIG"); p != "" {
		return p, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "milbi", "config.yaml"), nil
}

// Load reads and validates the config at path. Warnings are returned even when err is set.
func Load(path string) (*Config, []string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, &Error{Problems: []Problem{{Msg: err.Error()}}}
	}
	dir, err := filepath.Abs(filepath.Dir(path))
	if err != nil {
		return nil, nil, &Error{Problems: []Problem{{Msg: err.Error()}}}
	}
	c, warns, err := Parse(data, dir)
	if c != nil && c.hasLiteralSecret() {
		if info, serr := os.Stat(path); serr == nil && info.Mode().Perm()&0o077 != 0 {
			warns = append(warns, fmt.Sprintf("config %s holds plain-text secrets and has mode %04o, broader than 0600", path, info.Mode().Perm()))
		}
	}
	return c, warns, err
}

// hasLiteralSecret reports whether any secret in the config is plain text rather than a file: or env: reference.
func (c *Config) hasLiteralSecret() bool {
	var secrets []string
	for _, r := range c.Repositories {
		secrets = append(secrets, r.Password)
		for _, v := range r.Env {
			secrets = append(secrets, v)
		}
	}
	for _, s := range c.Syncs {
		secrets = append(secrets, s.B2KeyID, s.B2Key)
	}
	for _, s := range secrets {
		scheme, _, _ := strings.Cut(s, ":")
		if s != "" && scheme != "file" && scheme != "env" {
			return true
		}
	}
	return false
}

// Parse decodes strictly, expands local paths against dir and validates.
func Parse(data []byte, dir string) (*Config, []string, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, nil, &Error{Problems: []Problem{{Msg: err.Error()}}}
	}
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	c := &Config{}
	if err := dec.Decode(c); err != nil && !errors.Is(err, io.EOF) {
		var te *yaml.TypeError
		if errors.As(err, &te) {
			ps := make([]Problem, len(te.Errors))
			for i, msg := range te.Errors {
				ps[i] = Problem{Msg: msg} // already "line N: …"
			}
			return nil, nil, &Error{Problems: ps}
		}
		return nil, nil, &Error{Problems: []Problem{{Msg: err.Error()}}}
	}
	c.Dir = dir
	c.expand()
	if ps := c.validate(&doc); len(ps) > 0 {
		return c, c.warnings(), &Error{Problems: ps}
	}
	return c, c.warnings(), nil
}

// RetentionFor returns the backup's own retention, else the default.
func (c *Config) RetentionFor(b Backup) Retention {
	if b.Retention != nil {
		return b.Retention
	}
	return c.Defaults.Retention
}

// ExcludesFor returns the default excludes plus the backup's own.
func (c *Config) ExcludesFor(b Backup) []string {
	return append(append([]string{}, c.Defaults.Excludes...), b.Excludes...)
}

func (c *Config) expand() {
	for name, r := range c.Repositories {
		r.URL = ExpandPath(r.URL, c.Dir)
		c.Repositories[name] = r
	}
	for name, b := range c.Backups {
		for i, p := range b.From {
			b.From[i] = ExpandPath(p, c.Dir)
		}
		c.Backups[name] = b
	}
	for name, s := range c.Syncs {
		s.From = ExpandPath(s.From, c.Dir)
		s.To = ExpandPath(s.To, c.Dir)
		c.Syncs[name] = s
	}
}

// IsRemote reports whether p is a `host:path` or `scheme:…` location rather than a local path.
func IsRemote(p string) bool {
	i := strings.Index(p, ":")
	return i > 0 && !strings.Contains(p[:i], "/")
}

// ExpandPath expands `~` and $VARS in a local path and resolves it against dir.
// Remote locations are returned unchanged; a trailing slash is kept.
func ExpandPath(p, dir string) string {
	if p == "" || IsRemote(p) {
		return p
	}
	slash := strings.HasSuffix(p, "/")
	if p == "~" || strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			p = home + p[1:]
		}
	}
	p = os.ExpandEnv(p)
	if !filepath.IsAbs(p) {
		p = filepath.Join(dir, p)
	}
	if slash && !strings.HasSuffix(p, "/") {
		p += "/"
	}
	return p
}

func (c *Config) validate(doc *yaml.Node) []Problem {
	var ps []Problem
	add := func(line int, format string, args ...any) {
		ps = append(ps, Problem{Line: line, Msg: fmt.Sprintf(format, args...)})
	}

	switch {
	case c.Version == 0:
		add(lineOf(doc, "version"), "version is missing (want %d)", SupportedVersion)
	case c.Version != SupportedVersion:
		add(lineOf(doc, "version"), "version %d is not supported (want %d)", c.Version, SupportedVersion)
	}

	for _, msg := range checkRetention(c.Defaults.Retention) {
		add(lineOf(doc, "defaults", "retention"), "defaults: %s", msg)
	}

	for _, name := range SortedKeys(c.Repositories) {
		r := c.Repositories[name]
		line := lineOf(doc, "repositories", name)
		checkName(add, line, "repository", name)
		if r.URL == "" {
			add(line, "repository %s: url is missing", name)
		}
		if r.Password == "" {
			add(line, "repository %s: password is missing", name)
		} else if msg := checkSecret(r.Password); msg != "" {
			add(lineOf(doc, "repositories", name, "password"), "repository %s: password %s", name, msg)
		}
		for _, key := range SortedKeys(r.Env) {
			kline := lineOf(doc, "repositories", name, "env", key)
			switch {
			case !envKeyPattern.MatchString(key):
				add(kline, "repository %s: env key %q is not a valid variable name", name, key)
			case strings.HasPrefix(key, "RESTIC_PASSWORD"):
				add(kline, "repository %s: env must not set %s, use password", name, key)
			}
			if msg := checkSecret(r.Env[key]); msg != "" {
				add(kline, "repository %s: env %s %s", name, key, msg)
			}
		}
	}

	for _, name := range SortedKeys(c.Backups) {
		b := c.Backups[name]
		line := lineOf(doc, "backups", name)
		checkName(add, line, "backup", name)
		if _, ok := c.Syncs[name]; ok {
			add(line, "backup %s: name is also used by a sync", name)
		}
		if len(b.From) == 0 {
			add(line, "backup %s: from is missing", name)
		}
		if len(b.To) == 0 {
			add(line, "backup %s: to is missing", name)
		}
		seen := map[string]bool{}
		for _, repo := range b.To {
			if _, ok := c.Repositories[repo]; !ok {
				add(lineOf(doc, "backups", name, "to"), "backup %s: repository %q does not exist", name, repo)
			}
			if seen[repo] {
				add(lineOf(doc, "backups", name, "to"), "backup %s: repository %q is listed twice", name, repo)
			}
			seen[repo] = true
		}
		for _, msg := range checkRetention(b.Retention) {
			add(lineOf(doc, "backups", name, "retention"), "backup %s: %s", name, msg)
		}
		if keepsNothing(c.RetentionFor(b)) {
			add(line, "backup %s: retention keeps nothing (forget would remove every snapshot)", name)
		}
	}

	for _, name := range SortedKeys(c.Syncs) {
		s := c.Syncs[name]
		line := lineOf(doc, "syncs", name)
		checkName(add, line, "sync", name)
		if s.From == "" {
			add(line, "sync %s: from is missing", name)
		}
		switch {
		case s.To == "":
			add(line, "sync %s: to is missing", name)
		case s.IsB2():
			if bucket, _, _ := strings.Cut(strings.TrimPrefix(s.To, B2Prefix), "/"); bucket == "" {
				add(lineOf(doc, "syncs", name, "to"), "sync %s: b2 target has no bucket", name)
			}
		case isUnsupportedURL(s.To):
			add(lineOf(doc, "syncs", name, "to"), "sync %s: target %q is not supported yet (only b2:// and rsync targets)", name, s.To)
		}
		keys := []struct{ field, ref string }{{"b2_key_id", s.B2KeyID}, {"b2_key", s.B2Key}}
		for _, k := range keys {
			switch {
			case s.IsB2() && k.ref == "":
				add(line, "sync %s: %s is missing (required for b2:// targets)", name, k.field)
			case !s.IsB2() && k.ref != "":
				add(lineOf(doc, "syncs", name, k.field), "sync %s: %s is only allowed on b2:// targets", name, k.field)
			case k.ref != "":
				if msg := checkSecret(k.ref); msg != "" {
					add(lineOf(doc, "syncs", name, k.field), "sync %s: %s %s", name, k.field, msg)
				}
			}
		}
	}
	return ps
}

func (c *Config) warnings() []string {
	var ws []string
	check := func(kind, name, p string) {
		if p == "" || IsRemote(p) {
			return
		}
		if _, err := os.Stat(p); err != nil {
			ws = append(ws, fmt.Sprintf("%s %s: %s does not exist", kind, name, p))
		}
	}
	for _, name := range SortedKeys(c.Backups) {
		for _, p := range c.Backups[name].From {
			check("backup", name, p)
		}
	}
	for _, name := range SortedKeys(c.Syncs) {
		check("sync", name, c.Syncs[name].From)
	}
	return ws
}

func checkName(add func(int, string, ...any), line int, kind, name string) {
	if !namePattern.MatchString(name) {
		add(line, "%s name %q must match [a-z0-9-]+", kind, name)
	}
}

func checkRetention(r Retention) []string {
	var msgs []string
	for _, key := range SortedKeys(r) {
		if !retentionKeys[key] {
			msgs = append(msgs, fmt.Sprintf("retention key %q is unknown (use last, hourly, daily, weekly, monthly, yearly)", key))
		} else if r[key] < 0 {
			msgs = append(msgs, fmt.Sprintf("retention %s is negative", key))
		}
	}
	return msgs
}

func keepsNothing(r Retention) bool {
	for key, n := range r {
		if retentionKeys[key] && n > 0 {
			return false
		}
	}
	return true
}

// checkSecret returns why a secret is invalid, or "". A value without a file: or env: prefix is a
// plain-text literal and is allowed. It never echoes the value.
func checkSecret(ref string) string {
	scheme, rest, ok := strings.Cut(ref, ":")
	if ok && (scheme == "file" || scheme == "env") && rest == "" {
		return fmt.Sprintf("has an empty %s: reference", scheme)
	}
	return ""
}

// ResolveSecret reads a file: or env: reference, or returns a literal as is. The warning is set when a
// secret file is readable by others.
func ResolveSecret(ref, dir string) (value, warning string, err error) {
	if msg := checkSecret(ref); msg != "" {
		return "", "", fmt.Errorf("secret %s", msg)
	}
	scheme, rest, _ := strings.Cut(ref, ":")
	if scheme != "file" && scheme != "env" {
		return ref, "", nil
	}
	if scheme == "env" {
		v, ok := os.LookupEnv(rest)
		if !ok {
			return "", "", fmt.Errorf("secret: environment variable %s is not set", rest)
		}
		return v, "", nil
	}
	p := ExpandPath(rest, dir)
	info, err := os.Stat(p)
	if err != nil {
		return "", "", fmt.Errorf("secret: %w", err)
	}
	if info.Mode().Perm()&0o077 != 0 {
		warning = fmt.Sprintf("secret file %s has mode %04o, broader than 0600", p, info.Mode().Perm())
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return "", "", fmt.Errorf("secret: %w", err)
	}
	return strings.TrimSuffix(string(data), "\n"), warning, nil
}

// lineOf follows mapping keys from the document root and returns the line of the deepest key found.
func lineOf(doc *yaml.Node, path ...string) int {
	n := doc
	if n.Kind == yaml.DocumentNode && len(n.Content) > 0 {
		n = n.Content[0]
	}
	line := 0
	for _, key := range path {
		if n.Kind != yaml.MappingNode {
			return line
		}
		found := false
		for i := 0; i+1 < len(n.Content); i += 2 {
			if n.Content[i].Value == key {
				line, n, found = n.Content[i].Line, n.Content[i+1], true
				break
			}
		}
		if !found {
			return line
		}
	}
	return line
}

// SortedKeys returns the map's keys in order, for stable output.
func SortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
