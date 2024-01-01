package app

import (
	"github.com/la3mmchen/milbi/internal/restic"
	"github.com/urfave/cli/v2"

	"fmt"
)

func Snapshot(cfg *Milbi) *cli.Command {
	cmd := cli.Command{
		Name:    "snapshot",
		Aliases: []string{"b", "s"},
		Usage:   "create a backup snapshot.",
	}

	cmd.Flags = []cli.Flag{
		&cli.BoolFlag{
			Name:        "simulate",
			Value:       false,
			Destination: &cfg.SnapshotSimulate,
			DefaultText: "simulate a snapshot",
		},
	}

	cmd.Action = func(c *cli.Context) error {

		err := preloadConfig(cfg)

		if err != nil {
			return err
		}

		for _, repo := range cfg.Repos {

			resticRepo, err := restic.LoadRepo(repo.MetaData.Name, repo.Spec.Passphrase, repo.Spec.Directory)

			if err != nil {
				return err
			}

			var out []byte
			if cfg.SnapshotSimulate {
				fmt.Println("Simulating a snapshot.")
				out, err = resticRepo.SimulateBackup(repo.MetaData.Hostalias, repo.Spec.Content, repo.Spec.Excludes)
			} else {
				fmt.Println("Create a snapshot.")
				out, err = resticRepo.CreateBackup(repo.MetaData.Hostalias, repo.Spec.Content, repo.Spec.Excludes)
			}

			if err != nil {
				return err
			}

			resticOutputToConsole(out)
		}

		return nil
	}

	return &cmd
}
