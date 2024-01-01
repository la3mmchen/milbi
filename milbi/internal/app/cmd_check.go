package app

import (
	"github.com/la3mmchen/milbi/internal/restic"
	"github.com/urfave/cli/v2"
)

func Check(cfg *Milbi) *cli.Command {
	cmd := cli.Command{
		Name:  "check",
		Usage: "check the consistency of the backup.",
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

			info, err := resticRepo.Check()

			if err != nil {
				return err
			}

			resticOutputToConsole(info)
		}

		return nil
	}

	return &cmd
}
