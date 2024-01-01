package app

import (
	"fmt"

	"github.com/urfave/cli/v2"
)

func Explain(cfg *Milbi) *cli.Command {
	cmd := cli.Command{
		Name:  "explain",
		Usage: "explains the loaded config.",
	}

	cmd.Flags = []cli.Flag{
		&cli.BoolFlag{
			Name:        "new",
			Value:       false,
			Destination: &cfg.ExplainNew,
			DefaultText: "create a new config as start point",
		},
	}

	cmd.Action = func(c *cli.Context) error {

		err := preloadConfig(cfg)

		if err != nil {
			return err
		}

		fmt.Println("")
		fmt.Println("Repos:")
		for _, repo := range cfg.Repos {
			fmt.Printf("%v \n", repo.MetaData.Name)
			fmt.Printf("%v \n", repo.Spec.Content)
		}

		return nil
	}
	return &cmd
}
