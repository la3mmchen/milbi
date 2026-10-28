//go:build e2e

// Package e2e runs the built milbi binary against real restic and rsync.
// Each folder in testdata/ is one scenario: script.txtar (commands and expectations)
// plus any fixture files (milbi.yaml, files/), copied into the script's $WORK first.
package e2e

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"
)

var binDir string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "milbi-e2e-")
	if err != nil {
		panic(err)
	}
	build := exec.Command("go", "build", "-o", filepath.Join(dir, "milbi"), "../cmd/milbi")
	build.Stdout, build.Stderr = os.Stdout, os.Stderr
	if err := build.Run(); err != nil {
		panic(err)
	}
	binDir = dir
	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

func TestScenarios(t *testing.T) {
	for _, tool := range []string{"restic", "rsync"} {
		if _, err := exec.LookPath(tool); err != nil {
			if os.Getenv("MILBI_E2E_REQUIRE_TOOLS") != "" {
				t.Fatalf("%s not installed and MILBI_E2E_REQUIRE_TOOLS is set", tool)
			}
			t.Skipf("%s not installed", tool)
		}
	}
	scenarios, err := os.ReadDir("testdata")
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range scenarios {
		if !s.IsDir() || s.Name()[0] == '_' {
			continue
		}
		dir := filepath.Join("testdata", s.Name())
		t.Run(s.Name(), func(t *testing.T) {
			testscript.Run(t, testscript.Params{
				Dir: dir,
				Setup: func(env *testscript.Env) error {
					env.Setenv("PATH", binDir+string(os.PathListSeparator)+env.Getenv("PATH"))
					env.Setenv("HOME", filepath.Join(env.WorkDir, "home"))
					return copyFixtures(dir, env.WorkDir)
				},
			})
		})
	}
}

// copyFixtures copies everything in a scenario folder except its .txtar scripts into work.
func copyFixtures(src, work string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(work, rel)
		switch {
		case d.IsDir():
			return os.MkdirAll(target, 0o755)
		case filepath.Ext(path) == ".txtar":
			return nil
		case d.Type()&fs.ModeSymlink != 0:
			link, err := os.Readlink(path)
			if err != nil {
				return err
			}
			return os.Symlink(link, target)
		default:
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			info, err := d.Info()
			if err != nil {
				return err
			}
			return os.WriteFile(target, data, info.Mode().Perm())
		}
	})
}
