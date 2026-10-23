//go:build integration

package cli

import (
	"os/exec"
	"testing"
)

// The integration suite uses the real restic and rsync; it skips cleanly when one is missing.
// It grows with specs 05 and 06 (init, backup, snapshots, sync against temp dirs).
func requireTools(t *testing.T, tools ...string) {
	t.Helper()
	for _, tool := range tools {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("%s not installed", tool)
		}
	}
}

func TestIntegrationDoctorRealTools(t *testing.T) {
	requireTools(t, "restic", "rsync")
	t.Setenv("HOME", t.TempDir())
	code, out, errOut := run(t, systemEnv{}, "doctor")
	if code != ExitOK {
		t.Fatalf("exit %d, out %s, err %s", code, out, errOut)
	}
}
