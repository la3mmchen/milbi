package cli_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/la3mmchen/milbi/internal/cli"
	"github.com/rogpeppe/go-internal/testscript"
)

func TestMain(m *testing.M) {
	testscript.Main(m, map[string]func(){
		"milbi": func() {
			os.Exit(cli.Execute(cli.BuildInfo{Version: "test", Commit: "abc", Date: "2026-01-01"}, os.Stdout, os.Stderr))
		},
	})
}

// TestScripts runs testdata/script/*.txtar. Fake restic/rsync scripts live in the archives and are put on PATH there;
// the real tools are never run, and HOME/XDG_* point into the temp work dir.
func TestScripts(t *testing.T) {
	testscript.Run(t, testscript.Params{
		Dir: filepath.Join("testdata", "script"),
		Setup: func(e *testscript.Env) error {
			home := filepath.Join(e.WorkDir, "home")
			if err := os.MkdirAll(home, 0o755); err != nil {
				return err
			}
			e.Setenv("HOME", home)
			e.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
			e.Setenv("XDG_STATE_HOME", filepath.Join(home, ".local", "state"))
			e.Setenv("XDG_CACHE_HOME", filepath.Join(home, ".cache"))
			return nil
		},
	})
}
