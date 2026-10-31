package backup

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/la3mmchen/milbi/internal/config"
)

// SyncStats is what one sync run transferred, read from rsync's --stats.
type SyncStats struct {
	FilesTransferred int   `json:"filesTransferred"`
	BytesSent        int64 `json:"bytesSent"`
}

var (
	rsyncFiles = regexp.MustCompile(`(?m)^Number of (?:regular )?files transferred: ([\d,.]+)`)
	// rsync 3.x: "Total bytes sent: 1,234"; openrsync: "Total sent: 1234 B" (larger units are skipped)
	rsyncBytes = regexp.MustCompile(`(?m)^Total (?:bytes )?sent: ([\d,.]+)(?: B)?$`)
)

// RsyncArgs is the rsync argv (without the binary) for a sync job. `from` and `to` are
// passed verbatim, so a trailing slash on `from` keeps rsync's meaning. --human-readable
// is left out: it turns the --stats numbers into units milbi would have to parse back.
func RsyncArgs(s config.Sync, dryRun bool) []string {
	args := []string{"-a", "--partial", "--stats"}
	if dryRun {
		args = append(args, "-n", "-v")
	}
	if s.Delete {
		args = append(args, "--delete")
	}
	for _, e := range s.Excludes {
		args = append(args, "--exclude", e)
	}
	return append(args, s.From, s.To)
}

// ParseRsyncStats reads the transferred file count and bytes sent from rsync's --stats
// output; numbers may carry thousands separators. Missing lines leave zeros.
func ParseRsyncStats(out string) SyncStats {
	var s SyncStats
	if m := rsyncFiles.FindStringSubmatch(out); m != nil {
		s.FilesTransferred, _ = strconv.Atoi(digits(m[1]))
	}
	if m := rsyncBytes.FindStringSubmatch(out); m != nil {
		s.BytesSent, _ = strconv.ParseInt(digits(m[1]), 10, 64)
	}
	return s
}

func digits(s string) string { return strings.NewReplacer(",", "", ".", "").Replace(s) }

// checkSync returns why a sync job can't run with this runner, or nil.
func (r *Runner) checkSync(s config.Sync) error {
	if s.IsB2() {
		return fmt.Errorf("b2 syncs are not supported yet")
	}
	if r.Rsync == "" {
		return fmt.Errorf("rsync not found on PATH")
	}
	return nil
}

// DrySync runs rsync with -n for a sync job, so the planned changes are printed and nothing
// is copied or deleted. It takes no lock and writes no state.
func (r *Runner) DrySync(job string) error {
	s := r.Config.Syncs[job]
	if err := r.checkSync(s); err != nil {
		return err
	}
	args := RsyncArgs(s, true)
	_, _ = fmt.Fprintf(r.Out, "%s: rsync %s\n", job, strings.Join(args, " "))
	cmd := exec.Command(r.Rsync, args...)
	cmd.Stdout, cmd.Stderr = r.Out, r.Err
	return cmd.Run()
}

// runSync runs rsync for a sync job, tees its output into log and records the outcome in res.
// Any non-zero rsync exit fails the job, with rsync's stderr as the error.
func (r *Runner) runSync(job string, log io.Writer, res *Result) {
	s := r.Config.Syncs[job]
	if err := r.checkSync(s); err != nil {
		res.Status, res.Error = StatusFailed, err.Error()
		return
	}
	args := RsyncArgs(s, false)
	_, _ = fmt.Fprintf(log, "$ rsync %s\n", strings.Join(args, " "))
	var stdout, stderr bytes.Buffer
	cmd := exec.Command(r.Rsync, args...)
	cmd.Stdout, cmd.Stderr = io.MultiWriter(&stdout, log), io.MultiWriter(&stderr, log)
	err := cmd.Run()
	stats := ParseRsyncStats(stdout.String())
	res.Sync = &stats
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		res.Status, res.ExitCode, res.Error = StatusFailed, exitCode(err), "rsync: "+msg
		return
	}
	res.Status = StatusSuccess
}
