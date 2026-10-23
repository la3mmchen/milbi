package cli

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

// env abstracts PATH lookup and process execution so doctor can be tested without the real tools.
type env interface {
	LookPath(name string) (string, error)
	// Version runs `<path> <arg>` and returns the first line of its output.
	Version(path, arg string) (string, error)
}

// doctorTools lists the required binaries with the argument that prints their version
// (restic has a `version` subcommand and rejects `--version`).
var doctorTools = []struct{ name, versionArg string }{
	{"restic", "version"},
	{"rsync", "--version"},
}

type systemEnv struct{}

func (systemEnv) LookPath(name string) (string, error) { return exec.LookPath(name) }

func (systemEnv) Version(path, arg string) (string, error) {
	out, err := exec.Command(path, arg).CombinedOutput()
	line, _, _ := strings.Cut(strings.TrimSpace(string(out)), "\n")
	if err != nil && line == "" {
		return "", err
	}
	return line, nil
}

func newDoctorCmd(e env) *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check that restic and rsync are available",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runDoctor(e, cmd)
		},
	}
}

func runDoctor(e env, cmd *cobra.Command) error {
	out := cmd.OutOrStdout()
	var missing []string
	for _, t := range doctorTools {
		tool := t.name
		path, err := e.LookPath(tool)
		if err != nil {
			_, _ = fmt.Fprintf(out, "%-6s MISSING (not found on PATH)\n", tool)
			missing = append(missing, tool)
			continue
		}
		ver, err := e.Version(path, t.versionArg)
		if err != nil {
			ver = "version unknown: " + err.Error()
		}
		_, _ = fmt.Fprintf(out, "%-6s %s (%s)\n", tool, ver, path)
		if tool == "rsync" && strings.Contains(strings.ToLower(ver), "openrsync") {
			_, _ = fmt.Fprintln(out, "warning: rsync is openrsync (macOS default); some options are unsupported, install rsync 3.x")
		}
	}
	if len(missing) > 0 {
		return &ExitError{Code: ExitMissing, Msg: "missing required binaries: " + strings.Join(missing, ", ")}
	}
	return nil
}
