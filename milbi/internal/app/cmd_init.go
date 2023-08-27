package app

import (
	"github.com/la3mmchen/milbi/internal/restic"
	"github.com/urfave/cli/v2"

	"fmt"
)

func Init(cfg *Milbi) *cli.Command {
	cmd := cli.Command{
		Name:  "init",
		Usage: "create non-existing repos based on the config.",
	}

	cmd.Action = func(c *cli.Context) error {

		// do not catch errors here, as we expect that
		_ = preloadConfig(cfg)

		for _, repo := range cfg.Repos {

			_, err := restic.LoadRepo(repo.MetaData.Name, repo.Spec.Passphrase, repo.Spec.Directory)

			if err != nil {
				fmt.Printf("Creating a new repo at %v\n", repo.Spec.Directory)
				_, out, err := restic.NewRepo(repo.Spec.Passphrase, repo.Spec.Directory)

				if err != nil {
					return err
				}

				resticOutputToConsole(out)
			}

			resticRepo, _ := restic.LoadRepo(repo.MetaData.Name, repo.Spec.Passphrase, repo.Spec.Directory)
			info, err := resticRepo.GetInfo()

			if err != nil {
				return err
			}

			resticOutputToConsole(info)
		}

		return nil
	}

	return &cmd
}
