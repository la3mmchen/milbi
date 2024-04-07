package app

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"gopkg.in/yaml.v3"
)

// absoluteFromConfigFile enriches relative path with the project path
// TODO: might be cool to add error handling
func absoluteFromConfigFile(configgile string, s string) string {
	_, b, _, _ := runtime.Caller(0)
	path := filepath.Join(filepath.Dir(b), "../..", s) // feels hacky but works

	return path
}

// getAbsolutePath translates a relative path to an absolute path
// TODO: might be cool to add error handling
func getAbsolutePath(s string) string {

	// expand home dir
	if strings.HasPrefix(s, "~/") {
		dirname, _ := os.UserHomeDir()
		s = filepath.Join(dirname, s[2:])
	}

	path, err := filepath.Abs(s)
	if err != nil {
		return ""
	}

	return path
}

// preloadConfig common stuff that might be useful for all commands
// return error in case somethin is broken with the config
func preloadConfig(cfg *Milbi) error {

	// check if input file exists
	if _, err := os.Stat(getAbsolutePath(cfg.Configfile)); errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("%v does not exists. Exiting", cfg.Configfile)

	}

	err := yaml2struct(cfg)

	if err != nil {
		return err
	}

	// sanitize paths
	for i, manifest := range cfg.Repos {
		// kind: repo
		// spec.Directory
		if manifest.Spec.Directory != "" {
			if !filepath.IsAbs(manifest.Spec.Directory) {
				cfg.Repos[i].Spec.Directory = absoluteFromConfigFile(cfg.Configfile, manifest.Spec.Directory)
			}
		}
		//spec.content
		if len(manifest.Spec.Content) > 0 {
			for a, content := range manifest.Spec.Content {
				cfg.Repos[i].Spec.Content[a] = absoluteFromConfigFile(cfg.Configfile, content)
			}
		}
	}

	// sanitize paths
	for i, manifest := range cfg.Syncs {
		// kind: sync
		// spec.source
		if manifest.Spec.Source != "" {
			cfg.Syncs[i].Spec.Source = absoluteFromConfigFile(cfg.Configfile, manifest.Spec.Source)
		}
	}

	return nil

}

// yaml2struct convert the yaml input config to
// a object of type Milbi which contains all
// necessary config parameter
func yaml2struct(cfg *Milbi) error {

	fileContent, err := os.ReadFile(getAbsolutePath(cfg.Configfile))

	if err != nil {
		return err
	}

	decoder := yaml.NewDecoder(bytes.NewBufferString(string(fileContent[:])))
	for {
		var d Manifest
		if err := decoder.Decode(&d); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("document decode failed: %w", err)
		}
		cfg.Manifests = append(cfg.Manifests, d)

		// sort the document for faster access
		if d.Kind == "repo" {
			cfg.Repos = append(cfg.Repos, d)
		}
		// sort the document for faster access
		if d.Kind == "sync" {
			cfg.Syncs = append(cfg.Syncs, d)
		}
	}

	return nil
}

// resticOutputToConsole is the central output
func resticOutputToConsole(out []byte) {
	fmt.Printf("%s\n", out)
}
