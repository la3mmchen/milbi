package backup

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"
)

// KeepLogs is how many log files milbi keeps per job.
const KeepLogs = 30

// ErrLocked means another run of the job holds its lock.
var ErrLocked = errors.New("job is locked by another run")

// State is the content of status/<job>.json.
type State struct {
	Job             string      `json:"job"`
	Status          Status      `json:"status"`
	StartedAt       time.Time   `json:"startedAt"`
	EndedAt         *time.Time  `json:"endedAt,omitempty"`
	DurationSeconds float64     `json:"durationSeconds"`
	LastSuccessAt   *time.Time  `json:"lastSuccessAt,omitempty"`
	ExitCode        int         `json:"exitCode"`
	Error           string      `json:"error,omitempty"`
	Stats           []RepoStats `json:"stats,omitempty"`
	Sync            *SyncStats  `json:"sync,omitempty"`
	Log             string      `json:"log"`
}

// DefaultStateDir is $XDG_STATE_HOME/milbi, else ~/.local/state/milbi.
func DefaultStateDir() (string, error) {
	if x := os.Getenv("XDG_STATE_HOME"); x != "" {
		return filepath.Join(x, "milbi"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "state", "milbi"), nil
}

// ReadState reads a job's state file.
func ReadState(dir, job string) (State, error) {
	var st State
	data, err := os.ReadFile(filepath.Join(dir, "status", job+".json"))
	if err != nil {
		return st, err
	}
	return st, json.Unmarshal(data, &st)
}

// writeState replaces status/<job>.json atomically (temp file plus rename).
func writeState(dir string, st State) error {
	sdir := filepath.Join(dir, "status")
	if err := os.MkdirAll(sdir, 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(sdir, "."+st.Job+"-*.json")
	if err != nil {
		return err
	}
	_, werr := tmp.Write(append(data, '\n'))
	cerr := tmp.Close()
	if err := errors.Join(werr, cerr); err != nil {
		_ = os.Remove(tmp.Name())
		return err
	}
	return os.Rename(tmp.Name(), filepath.Join(sdir, st.Job+".json"))
}

// lock takes locks/<job>.lock with flock, so a crashed run never leaves a stale lock.
// Closing the returned file releases it.
func (r *Runner) lock(job string) (*os.File, error) {
	dir := filepath.Join(r.StateDir, "locks")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(filepath.Join(dir, job+".lock"), os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = f.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return nil, ErrLocked
		}
		return nil, fmt.Errorf("lock %s: %w", job, err)
	}
	return f, nil
}

// openLog creates logs/<job>/<timestamp>.log.
func (r *Runner) openLog(job string, at time.Time) (*os.File, error) {
	dir := filepath.Join(r.StateDir, "logs", job)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	name := at.UTC().Format("20060102T150405.000Z") + ".log"
	return os.OpenFile(filepath.Join(dir, name), os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o600)
}

// trimLogs keeps the newest KeepLogs logs of a job; the names sort by time.
func (r *Runner) trimLogs(job string) {
	dir := filepath.Join(r.StateDir, "logs", job)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	var logs []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".log") {
			logs = append(logs, e.Name())
		}
	}
	sort.Strings(logs)
	for len(logs) > KeepLogs {
		_ = os.Remove(filepath.Join(dir, logs[0]))
		logs = logs[1:]
	}
}
