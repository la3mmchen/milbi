package app

import (
	"fmt"

	"github.com/la3mmchen/milbi/internal/syncobject"
	"github.com/urfave/cli/v2"
)

func Sync(cfg *Milbi) *cli.Command {
	cmd := cli.Command{
		Name:  "sync",
		Usage: "execute configured syncs.",
	}

	cmd.Action = func(c *cli.Context) error {

		err := preloadConfig(cfg)

		if err != nil {
			return err
		}

		for _, sync := range cfg.Syncs {

			syncObject, err := syncobject.LoadSyncObject(sync.MetaData.Name, sync.Spec.Binary, sync.Spec.Source, sync.Spec.Target, sync.Spec.Flags)

			if err != nil {
				return err
			}

			err = syncObject.Sync()

			if err != nil {
				return err
			}

			fmt.Println("--")

		}

		return nil
	}

	return &cmd
}
