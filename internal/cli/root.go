// Package cli holds the cobra commands. Commands stay thin: they parse flags and call into other packages.
package cli

import (
	"errors"
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

// Exit codes shared by all commands.
const (
	ExitOK      = 0
	ExitFailure = 1
	ExitMissing = 3
)

// BuildInfo is injected into the binary at build time.
type BuildInfo struct {
	Version string
	Commit  string
	Date    string
}

// ExitError makes a command end with a specific exit code.
type ExitError struct {
	Code int
	Msg  string
}

func (e *ExitError) Error() string { return e.Msg }

// Execute runs the root command and returns the process exit code.
func Execute(info BuildInfo, stdout, stderr io.Writer) int {
	return execute(newRootCmd(info, stdout, stderr), stderr)
}

func execute(root *cobra.Command, stderr io.Writer) int {
	err := root.Execute()
	if err == nil {
		return ExitOK
	}
	_, _ = fmt.Fprintln(stderr, "error:", err)
	var ee *ExitError
	if errors.As(err, &ee) {
		return ee.Code
	}
	return ExitFailure
}

func newRootCmd(info BuildInfo, stdout, stderr io.Writer) *cobra.Command {
	root := &cobra.Command{
		Use:           "milbi",
		Short:         "Run restic backups and rsync syncs from one config",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.AddCommand(newVersionCmd(info), newDoctorCmd(systemEnv{}))
	return root
}
