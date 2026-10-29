package cli

import (
	"fmt"
	"io"
	"time"

	"github.com/la3mmchen/milbi/internal/backup"
	"github.com/la3mmchen/milbi/internal/config"
	"github.com/spf13/cobra"
)

// ExitLocked: a job was skipped because another run held its lock.
const ExitLocked = 4

// newRunner loads the config, finds restic and returns a runner writing to cmd's output.
func newRunner(e env, path string, cmd *cobra.Command) (*backup.Runner, error) {
	c, err := loadConfig(path, cmd.ErrOrStderr())
	if err != nil {
		return nil, err
	}
	restic, err := e.LookPath("restic")
	if err != nil {
		return nil, &ExitError{Code: ExitMissing, Msg: "restic not found on PATH"}
	}
	dir, err := backup.DefaultStateDir()
	if err != nil {
		return nil, err
	}
	return &backup.Runner{Config: c, Restic: restic, StateDir: dir, Out: cmd.OutOrStdout(), Err: cmd.ErrOrStderr(), Now: time.Now}, nil
}

// loadConfig loads the config at path (or the default), printing warnings; errors exit 2.
func loadConfig(path string, stderr io.Writer) (*config.Config, error) {
	p, err := config.Path(path)
	if err != nil {
		return nil, &ExitError{Code: ExitConfig, Msg: err.Error()}
	}
	c, warnings, err := config.Load(p)
	for _, w := range warnings {
		_, _ = fmt.Fprintln(stderr, "warning:", w)
	}
	if err != nil {
		return nil, &ExitError{Code: ExitConfig, Msg: err.Error()}
	}
	return c, nil
}

func checkRepository(c *config.Config, name string) error {
	if _, ok := c.Repositories[name]; !ok {
		return &ExitError{Code: ExitConfig, Msg: fmt.Sprintf("repository %q is not in the config", name)}
	}
	return nil
}

func checkBackup(c *config.Config, name string) error {
	if _, ok := c.Backups[name]; ok {
		return nil
	}
	if _, ok := c.Syncs[name]; ok {
		return &ExitError{Code: ExitConfig, Msg: fmt.Sprintf("%s is a sync; syncs can't be run yet", name)}
	}
	return &ExitError{Code: ExitConfig, Msg: fmt.Sprintf("backup %q is not in the config", name)}
}

func newInitCmd(e env, path *string) *cobra.Command {
	return &cobra.Command{
		Use:   "init <repository>",
		Short: "Initialize a restic repository from the config",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := newRunner(e, *path, cmd)
			if err != nil {
				return err
			}
			if err := checkRepository(r.Config, args[0]); err != nil {
				return err
			}
			if err := r.Init(args[0]); err != nil {
				return &ExitError{Code: ExitFailure, Msg: "init: " + err.Error()}
			}
			return nil
		},
	}
}

func newPruneCmd(e env, path *string) *cobra.Command {
	return &cobra.Command{
		Use:   "prune <repository>",
		Short: "Run restic prune on a repository (never run automatically)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := newRunner(e, *path, cmd)
			if err != nil {
				return err
			}
			if err := checkRepository(r.Config, args[0]); err != nil {
				return err
			}
			if err := r.Prune(args[0]); err != nil {
				return &ExitError{Code: ExitFailure, Msg: "prune: " + err.Error()}
			}
			return nil
		},
	}
}

func newSnapshotsCmd(e env, path *string) *cobra.Command {
	return &cobra.Command{
		Use:   "snapshots <job>",
		Short: "List a backup job's snapshots in each of its repositories",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := newRunner(e, *path, cmd)
			if err != nil {
				return err
			}
			if err := checkBackup(r.Config, args[0]); err != nil {
				return err
			}
			if err := r.Snapshots(args[0]); err != nil {
				return &ExitError{Code: ExitFailure, Msg: "snapshots: " + err.Error()}
			}
			return nil
		},
	}
}

func newRunCmd(e env, path *string) *cobra.Command {
	var all, dryRun bool
	cmd := &cobra.Command{
		Use:   "run [<job>...]",
		Short: "Run backup jobs, in the given order or with --all in config order",
		RunE: func(cmd *cobra.Command, args []string) error {
			if all == (len(args) > 0) {
				return &ExitError{Code: ExitConfig, Msg: "name one or more jobs, or use --all"}
			}
			r, err := newRunner(e, *path, cmd)
			if err != nil {
				return err
			}
			jobs := args
			if all {
				jobs = r.Config.BackupNames()
			}
			for _, j := range jobs {
				if err := checkBackup(r.Config, j); err != nil {
					return err
				}
			}
			if dryRun {
				for _, j := range jobs {
					r.DryRun(j)
				}
				return nil
			}
			return runJobs(r, jobs, cmd.OutOrStdout())
		},
	}
	cmd.Flags().BoolVar(&all, "all", false, "run every backup job in config order")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print the restic commands without running them")
	return cmd
}

// runJobs runs the jobs in order and maps their outcomes to the exit code: 1 if any failed,
// else 4 if any was skipped, else 0.
func runJobs(r *backup.Runner, jobs []string, out io.Writer) error {
	var failed, skipped []string
	for _, j := range jobs {
		res := r.Run(j)
		_, _ = fmt.Fprintf(out, "%s: %s\n", j, res.Status)
		for _, s := range res.Stats {
			_, _ = fmt.Fprintf(out, "  %s: snapshot %s, %d new, %d changed, %s added\n",
				s.Repository, short(s.SnapshotID), s.FilesNew, s.FilesChanged, backup.FormatBytes(s.BytesAdded))
		}
		if res.Error != "" {
			_, _ = fmt.Fprintf(out, "  %s\n", res.Error)
		}
		if res.Log != "" {
			_, _ = fmt.Fprintf(out, "  log: %s\n", res.Log)
		}
		switch res.Status {
		case backup.StatusFailed:
			failed = append(failed, j)
		case backup.StatusSkipped:
			skipped = append(skipped, j)
		}
	}
	switch {
	case len(failed) > 0:
		return &ExitError{Code: ExitFailure, Msg: fmt.Sprintf("failed: %v", failed)}
	case len(skipped) > 0:
		return &ExitError{Code: ExitLocked, Msg: fmt.Sprintf("skipped (locked): %v", skipped)}
	}
	return nil
}

func short(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	if id == "" {
		return "-"
	}
	return id
}
