package cli

import (
	"fmt"
	"strings"

	"github.com/la3mmchen/milbi/internal/config"
	"github.com/spf13/cobra"
)

func newConfigCmd(path *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Work with the config file",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "validate",
		Short: "Check the config and list its backups and syncs",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			p, err := config.Path(*path)
			if err != nil {
				return &ExitError{Code: ExitConfig, Msg: err.Error()}
			}
			c, warnings, err := config.Load(p)
			for _, w := range warnings {
				_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "warning:", w)
			}
			if err != nil {
				return &ExitError{Code: ExitConfig, Msg: err.Error()}
			}
			out := cmd.OutOrStdout()
			_, _ = fmt.Fprintf(out, "config %s is valid\n", p)
			_, _ = fmt.Fprintln(out, "backups:")
			for _, name := range config.SortedKeys(c.Backups) {
				_, _ = fmt.Fprintf(out, "  %s -> %s\n", name, strings.Join(c.Backups[name].To, ", "))
			}
			_, _ = fmt.Fprintln(out, "syncs:")
			for _, name := range config.SortedKeys(c.Syncs) {
				_, _ = fmt.Fprintf(out, "  %s -> %s\n", name, c.Syncs[name].To)
			}
			return nil
		},
	})
	return cmd
}
