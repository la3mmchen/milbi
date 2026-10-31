// Package backup runs restic for backup jobs (spec 05): init, run, snapshots and prune,
// with a state file and a lock per job. Secrets reach restic only through its environment.
package backup

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/la3mmchen/milbi/internal/config"
)

// Status is the outcome of a job run.
type Status string

// Job states, as written to the state file.
const (
	StatusRunning Status = "running"
	StatusSuccess Status = "success"
	StatusWarning Status = "warning"
	StatusFailed  Status = "failed"
	StatusSkipped Status = "skipped"
)

// resticIncomplete is restic's exit code for a snapshot that misses unreadable files.
const resticIncomplete = 3

// Runner runs restic and rsync for the jobs of one config.
type Runner struct {
	Config   *config.Config
	Restic   string // path of the restic binary
	Rsync    string // path of the rsync binary, "" when not installed
	StateDir string
	Out, Err io.Writer
	Now      func() time.Time
}

// RepoStats is what one backup into one repository added.
type RepoStats struct {
	Repository   string `json:"repository"`
	SnapshotID   string `json:"snapshotId,omitempty"`
	FilesNew     int    `json:"filesNew"`
	FilesChanged int    `json:"filesChanged"`
	BytesAdded   int64  `json:"bytesAdded"`
}

// Result is the outcome of one job run.
type Result struct {
	Job      string
	Status   Status
	ExitCode int
	Error    string
	Stats    []RepoStats
	Sync     *SyncStats
	Log      string
}

// BackupArgs is the restic argv (without the binary) for a backup job.
func BackupArgs(job string, excludes, from []string) []string {
	args := []string{"backup", "--json", "--tag", "milbi", "--tag", "job:" + job}
	for _, e := range excludes {
		args = append(args, "--exclude", e)
	}
	return append(args, from...)
}

// ForgetArgs is the restic argv for applying a job's retention, scoped to the job's tag
// so jobs sharing a repository never remove each other's snapshots. It returns nil when
// the retention keeps nothing, so forget is never run without a keep policy.
func ForgetArgs(job string, keep config.Retention) []string {
	args := []string{"forget", "--tag", "job:" + job, "--group-by", "host,tags"}
	n := len(args)
	for _, k := range config.SortedKeys(keep) {
		if keep[k] > 0 {
			args = append(args, "--keep-"+k, strconv.Itoa(keep[k]))
		}
	}
	if len(args) == n {
		return nil
	}
	return args
}

// Init runs `restic init` for a repository; it refuses one that is already initialized.
func (r *Runner) Init(repo string) error {
	env, err := r.repoEnv(repo)
	if err != nil {
		return err
	}
	if r.restic(env, io.Discard, io.Discard, "cat", "config") == nil {
		return fmt.Errorf("repository %s is already initialized", repo)
	}
	return r.restic(env, r.Out, r.Err, "init")
}

// Prune runs `restic prune` for a repository.
func (r *Runner) Prune(repo string) error {
	env, err := r.repoEnv(repo)
	if err != nil {
		return err
	}
	return r.restic(env, r.Out, r.Err, "prune")
}

// Snapshots lists a job's snapshots in each of its repositories.
func (r *Runner) Snapshots(job string) error {
	var errs []error
	for _, repo := range r.Config.Backups[job].To {
		_, _ = fmt.Fprintf(r.Out, "repository %s:\n", repo)
		env, err := r.repoEnv(repo)
		if err == nil {
			err = r.restic(env, r.Out, r.Err, "snapshots", "--tag", "job:"+job)
		}
		if err != nil {
			errs = append(errs, fmt.Errorf("repository %s: %w", repo, err))
		}
	}
	return errors.Join(errs...)
}

// DryRun prints the restic commands a run of job would execute, without running them.
func (r *Runner) DryRun(job string) {
	b := r.Config.Backups[job]
	forget := ForgetArgs(job, r.Config.RetentionFor(b))
	for _, repo := range b.To {
		_, _ = fmt.Fprintf(r.Out, "%s [%s]: restic %s\n", job, repo, strings.Join(BackupArgs(job, r.Config.ExcludesFor(b), b.From), " "))
		if forget != nil {
			_, _ = fmt.Fprintf(r.Out, "%s [%s]: restic %s\n", job, repo, strings.Join(forget, " "))
		}
	}
}

// Run runs a backup or sync job. A backup goes into each of its repositories, followed by
// its retention there; a sync runs rsync once.
// A job whose lock is held by another run is skipped and leaves the state file alone.
func (r *Runner) Run(job string) Result {
	res := Result{Job: job}
	lock, err := r.lock(job)
	if errors.Is(err, ErrLocked) {
		res.Status, res.Error = StatusSkipped, "another run of this job holds the lock"
		return res
	}
	if err != nil {
		res.Status, res.Error = StatusFailed, err.Error()
		return res
	}
	defer func() { _ = lock.Close() }()

	start := r.Now()
	st := State{Job: job, Status: StatusRunning, StartedAt: start}
	if prev, err := ReadState(r.StateDir, job); err == nil {
		st.LastSuccessAt = prev.LastSuccessAt
	}
	logFile, err := r.openLog(job, start)
	if err != nil {
		res.Status, res.Error = StatusFailed, err.Error()
		return res
	}
	st.Log, res.Log = logFile.Name(), logFile.Name()
	if err := writeState(r.StateDir, st); err != nil {
		_ = logFile.Close()
		res.Status, res.Error = StatusFailed, err.Error()
		return res
	}

	if _, ok := r.Config.Syncs[job]; ok {
		r.runSync(job, logFile, &res)
	} else {
		r.runRepos(job, logFile, &res)
	}
	_ = logFile.Close()
	r.trimLogs(job)

	end := r.Now()
	st.Status, st.ExitCode, st.Error, st.Stats, st.Sync = res.Status, res.ExitCode, res.Error, res.Stats, res.Sync
	st.EndedAt = &end
	st.DurationSeconds = end.Sub(start).Seconds()
	if res.Status == StatusSuccess || res.Status == StatusWarning {
		st.LastSuccessAt = &end
	}
	if err := writeState(r.StateDir, st); err != nil && res.Status != StatusFailed {
		res.Status, res.Error = StatusFailed, err.Error()
	}
	return res
}

// runRepos backs up into every repository of the job and records the worst outcome in res.
func (r *Runner) runRepos(job string, log io.Writer, res *Result) {
	b := r.Config.Backups[job]
	res.Status = StatusSuccess
	var msgs []string
	fail := func(status Status, code int, repo, msg string) {
		if rank(status) > rank(res.Status) {
			res.Status, res.ExitCode = status, code
		}
		msgs = append(msgs, repo+": "+msg)
	}
	for _, repo := range b.To {
		_, _ = fmt.Fprintf(log, "== repository %s\n", repo)
		env, err := r.repoEnv(repo)
		if err != nil {
			fail(StatusFailed, 0, repo, err.Error())
			continue
		}
		args := BackupArgs(job, r.Config.ExcludesFor(b), b.From)
		stats, code, err := r.backup(env, log, args)
		stats.Repository = repo
		res.Stats = append(res.Stats, stats)
		switch {
		case code == resticIncomplete:
			fail(StatusWarning, code, repo, "some files could not be read (restic exit 3)")
		case err != nil:
			fail(StatusFailed, code, repo, "backup: "+err.Error())
			continue
		}
		if code != 0 {
			continue // retention only after a complete backup
		}
		if forget := ForgetArgs(job, r.Config.RetentionFor(b)); forget != nil {
			_, _ = fmt.Fprintf(log, "$ restic %s\n", strings.Join(forget, " "))
			if err := r.restic(env, log, log, forget...); err != nil {
				fail(StatusFailed, exitCode(err), repo, "forget: "+err.Error())
			}
		}
	}
	res.Error = strings.Join(msgs, "; ")
}

func rank(s Status) int {
	switch s {
	case StatusWarning:
		return 1
	case StatusFailed:
		return 2
	}
	return 0
}

// backup runs restic backup, writes its output except progress lines to log and parses the summary.
func (r *Runner) backup(env []string, log io.Writer, args []string) (RepoStats, int, error) {
	_, _ = fmt.Fprintf(log, "$ restic %s\n", strings.Join(args, " "))
	var stdout bytes.Buffer
	err := r.restic(env, &stdout, log, args...)
	var stats RepoStats
	sc := bufio.NewScanner(&stdout)
	sc.Buffer(make([]byte, 64*1024), 16*1024*1024)
	for sc.Scan() {
		line := sc.Bytes()
		var msg struct {
			MessageType  string `json:"message_type"`
			FilesNew     int    `json:"files_new"`
			FilesChanged int    `json:"files_changed"`
			DataAdded    int64  `json:"data_added"`
			SnapshotID   string `json:"snapshot_id"`
		}
		if json.Unmarshal(line, &msg) == nil && msg.MessageType == "status" {
			continue
		}
		_, _ = fmt.Fprintf(log, "%s\n", line)
		if msg.MessageType == "summary" {
			stats = RepoStats{SnapshotID: msg.SnapshotID, FilesNew: msg.FilesNew, FilesChanged: msg.FilesChanged, BytesAdded: msg.DataAdded}
		}
	}
	return stats, exitCode(err), err
}

func exitCode(err error) int {
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return ee.ExitCode()
	}
	return 0
}

func (r *Runner) restic(env []string, stdout, stderr io.Writer, args ...string) error {
	cmd := exec.Command(r.Restic, args...)
	cmd.Env, cmd.Stdout, cmd.Stderr = env, stdout, stderr
	return cmd.Run()
}

// repoEnv is milbi's environment minus inherited restic credentials, plus the repository's
// location, password and env, with secrets resolved. Secret-file warnings go to r.Err.
func (r *Runner) repoEnv(name string) ([]string, error) {
	repo, ok := r.Config.Repositories[name]
	if !ok {
		return nil, fmt.Errorf("repository %q does not exist", name)
	}
	var env []string
	for _, kv := range os.Environ() {
		if !strings.HasPrefix(kv, "RESTIC_PASSWORD") && !strings.HasPrefix(kv, "RESTIC_REPOSITORY") {
			env = append(env, kv)
		}
	}
	pw, err := r.secret(repo.Password)
	if err != nil {
		return nil, fmt.Errorf("repository %s: password: %w", name, err)
	}
	env = append(env, "RESTIC_REPOSITORY="+repo.URL, "RESTIC_PASSWORD="+pw)
	for _, k := range config.SortedKeys(repo.Env) {
		v, err := r.secret(repo.Env[k])
		if err != nil {
			return nil, fmt.Errorf("repository %s: env %s: %w", name, k, err)
		}
		env = append(env, k+"="+v)
	}
	return env, nil
}

func (r *Runner) secret(ref string) (string, error) {
	v, warning, err := config.ResolveSecret(ref, r.Config.Dir)
	if warning != "" {
		_, _ = fmt.Fprintln(r.Err, "warning:", warning)
	}
	return v, err
}

// FormatBytes renders a byte count with a binary unit, for run summaries.
func FormatBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}
