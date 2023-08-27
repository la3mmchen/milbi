package app

import "github.com/urfave/cli/v2"

// Milbi contains the structured config during our lifetime
type Milbi struct {
	// basic settings for the cli itself
	AppName    string
	AppUsage   string
	AppVersion string

	// content to be injected via flags
	Configfile       string
	ExplainNew       bool
	SnapshotSimulate bool

	// configuration
	Manifests []Manifest

	Repos []Manifest
	Syncs []Manifest

	// internal fields

	// mixed stuff
	AdditionalFlags []cli.Flag
}

// Manifest defines the behaviour of Milbi
type Manifest struct {
	ApiVersion string   `yaml:"apiVersion"`
	Kind       string   `yaml:"kind"`
	MetaData   Metadata `yaml:"metadata"`
	Spec       Spec     `yaml:"spec"`
}

type Metadata struct {
	Name      string `yaml:"name"`
	Hostalias string `yaml:"hostalias"`
}

type Spec struct {
	// components for kind: sync
	Name   string   `yaml:"name,omitempty"`
	Binary string   `yaml:"binary,omitempty"`
	Source string   `yaml:"source,omitempty"`
	Target string   `yaml:"target,omitempty"`
	Flags  []string `yaml:"flags,omitempty"`

	// components for kind: repo
	Passphrase  string   `yaml:"passphrase,omitempty"`
	Generations int      `yaml:"generations,omitempty"`
	Directory   string   `yaml:"directory,omitempty"`
	Excludes    []string `yaml:"excludes,omitempty,flow"`
	Content     []string `yaml:"content,omitempty,flow"`
}

// Config contains the parse config from the injected file.
type Config struct {
	Version int `yaml:"version"`

	Global map[string]string `yaml:"global,omitempty"`

	Repos []Repo       `yaml:"repos"`
	Syncs []SyncConfig `yaml:"syncs,omitempty"`
}

type Repo struct {
	Name        string   `yaml:"name"`
	Passphrase  string   `yaml:"passphrase"`
	Generations int      `yaml:"generations"`
	Directory   string   `yaml:"directory"`
	Excludes    []string `yaml:"excludes,omitempty"`
	Content     []string `yaml:"content"`
}

type SyncConfig struct {
	Name   string   `yaml:"name"`
	Binary string   `yaml:"binary"`
	Source string   `yaml:"source"`
	Target string   `yaml:"target"`
	Flags  []string `yaml:"flags"`
}
