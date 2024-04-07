package app

import (
	"fmt"

	"github.com/davecgh/go-spew/spew"
	"github.com/urfave/cli/v2"
)

func Dump(cfg *Milbi) *cli.Command {
	cmd := cli.Command{
		Name:  "dump",
		Usage: "debug output.",
	}

	cmd.Action = func(c *cli.Context) error {

		err := preloadConfig(cfg)

		if err != nil {
			return err
		}

		fmt.Printf("%+v \n", cfg.Configfile)
		fmt.Println("")
		spew.Dump(cfg)

		return nil
	}
	return &cmd
}
