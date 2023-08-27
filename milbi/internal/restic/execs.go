package restic

// there is no restic lib so we have to
// use exec to get information from a
// repo and/or do things.
import (
	"fmt"
	"os/exec"
)

func withOutput(passphrase string, args []string) ([]byte, error) {

	// restic
	args = append([]string{"--verbose"}, args...)
	cmd := exec.Command("restic", args...)

	cmd.Env = append(cmd.Environ(), "RESTIC_PASSWORD="+passphrase)

	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Executed: [%v] \n", cmd)
		return []byte{}, err
	}

	return out, nil
}
