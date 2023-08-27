package restic

import (
	"errors"
	"fmt"
	"os"
)

func NewRepo(passphrase string, location string) (*Repo, []byte, error) {

	repo := &Repo{
		Passphrase: passphrase,
		Location:   location,
	}

	out, err := withOutput(repo.Passphrase, []string{"--repo", repo.Location, "init"})

	if err != nil {
		return &Repo{}, []byte{}, fmt.Errorf("error on init of repo at [%v], error: %v", repo.Location, err)
	}

	return repo, out, nil

}

func LoadRepo(name string, passphrase string, location string) (*Repo, error) {

	repo := &Repo{
		Name:       name,
		Passphrase: passphrase,
		Location:   location,
	}

	err := repo.verify()

	if err != nil {
		return &Repo{}, fmt.Errorf("problem on veryfing %v, error: %v", repo.Name, err)
	}

	return repo, nil

}

// verify do a sanity check on the provided config
func (r Repo) verify() error {

	// check if the location is accessible
	if _, err := os.Stat(r.Location); errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("path %v is not accessible", r.Location)
	}

	// check if the location contains a repo by executing a basic command
	_, err := withOutput(r.Passphrase, []string{"--repo", r.Location, "stats"})

	if err != nil {
		return fmt.Errorf("path %v is probably not containing a restic repo", r.Location)
	}

	return nil
}

// GetInfo retrieves basic information about a repo
func (r Repo) GetInfo() ([]byte, error) {

	out, err := withOutput(r.Passphrase, []string{"--repo", r.Location, "snapshots"})

	if err != nil {
		return out, err
	}

	return out, nil
}

// CreateBackup stores a new snapshot in the repo
// it is oppiniated with some flags
func (r Repo) CreateBackup(host string, content []string, excludes []string) ([]byte, error) {

	backupFlags := []string{"--verbose", "--compression", "auto", "--host", host}
	backupFlags = append(backupFlags, "--exclude")
	backupFlags = append(backupFlags, excludes...)
	backupFlags = append(backupFlags, content...)

	out, err := withOutput(r.Passphrase, append([]string{"--repo", r.Location, "backup"}, backupFlags...))

	if err != nil {
		return []byte{}, err
	}

	return out, nil

}

// SimulateBackup simulate the creation of a new snapshot in the repo
func (r Repo) SimulateBackup(host string, content []string, excludes []string) ([]byte, error) {

	backupFlags := []string{"--dry-run", "--verbose", "--compression", "auto", "--host", host}
	backupFlags = append(backupFlags, "--exclude")
	backupFlags = append(backupFlags, excludes...)
	backupFlags = append(backupFlags, content...)

	out, err := withOutput(r.Passphrase, append([]string{"--repo", r.Location, "backup"}, backupFlags...))

	if err != nil {
		return []byte{}, err
	}

	return out, nil

}

// Check executes consistency checks
func (r Repo) Check() ([]byte, error) {

	out, err := withOutput(r.Passphrase, []string{"--repo", r.Location, "check"})

	if err != nil {
		return out, err
	}

	return out, nil
}

// Prune run prune on repo
func (r Repo) Prune() ([]byte, error) {

	out, err := withOutput(r.Passphrase, []string{"--repo", r.Location, "forget", "--keep-daily", "1"})

	if err != nil {
		return out, err
	}

	return out, nil
}

// GetName return the name of the repo
func (r Repo) GetName() string {
	return r.Name
}
