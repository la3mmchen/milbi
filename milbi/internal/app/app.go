package app

import (
	"github.com/urfave/cli/v2"
)

// CreateApp builds the cli app by enriching the
// urfave/cli app struct with our params, flags, and commands.
// returns a pointer to a cli.App struct
func CreateApp(cfg *Milbi) *cli.App {
	cliFlags := []cli.Flag{
		&cli.StringFlag{
			Name:        "config",
			Aliases:     []string{"c"},
			Value:       "~/.milbi/config.yaml",
			Usage:       "Config to read.",
			Destination: &cfg.Configfile,
		},
	}

	app := cli.App{
		Name:    cfg.AppName,
		Usage:   cfg.AppUsage,
		Version: cfg.AppVersion,
		Commands: []*cli.Command{
			Check(cfg),
			Explain(cfg),
			Info(cfg),
			Init(cfg),
			Prune(cfg),
			Snapshot(cfg),
			Sync(cfg),
		},
		Flags: cliFlags,
	}

	cli.VersionFlag = &cli.BoolFlag{
		Name:    "print-version",
		Aliases: []string{"V"},
		Usage:   "print only the version",
	}

	return &app
}
