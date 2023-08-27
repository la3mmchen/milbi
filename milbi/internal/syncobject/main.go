package syncobject

import (
	"errors"
	"fmt"
	"os"
)

func LoadSyncObject(name string, syncbinary string, source string, target string, flags []string) (*SyncObject, error) {

	syncobject := &SyncObject{
		Name:   name,
		Binary: syncbinary,
		Source: source,
		Target: target,
		Flags:  flags,
	}

	err := syncobject.verify()

	if err != nil {
		return &SyncObject{}, fmt.Errorf("problem on veryfing %v, error: %v", syncobject.Name, err)
	}

	return syncobject, nil

}

func (s SyncObject) Sync() error {

	_, err := osExec(s.Binary, append([]string{s.Source, s.Target}, s.Flags...))

	if err != nil {
		return fmt.Errorf("sync failed with [%v]", err)
	}

	return nil
}

func (s SyncObject) verify() error {

	if verifyBinary(s.Binary) != nil {
		return fmt.Errorf("binary for rsync not working")
	}

	// check if source is accessible
	if _, err := os.Stat(s.Source); errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("path %v is not accessible", s.Source)
	}

	return nil
}
