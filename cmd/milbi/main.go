package main

import (
	"os"

	"github.com/la3mmchen/milbi/internal/cli"
)

// Injected via ldflags, see Taskfile.yaml and .goreleaser.yaml.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	os.Exit(cli.Execute(cli.BuildInfo{Version: version, Commit: commit, Date: date}, os.Stdout, os.Stderr))
}
