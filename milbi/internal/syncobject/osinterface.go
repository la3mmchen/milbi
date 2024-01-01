package syncobject

// there is no restic lib so we have to
// use exec to get information from a
// repo and/or do things.
import (
	"fmt"
	"os/exec"
)

func verifyBinary(binary string) error {

	_, err := exec.LookPath(binary)
	if err != nil {
		return err
	}

	return nil
}

func osExec(binary string, args []string) ([]byte, error) {

	cmd := exec.Command(binary, args...)

	fmt.Printf("Executing: [%v] \n", cmd)

	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Executed: [%v] \n", cmd)
		return []byte{}, err
	}

	fmt.Printf("%s\n", out)

	return out, nil
}
