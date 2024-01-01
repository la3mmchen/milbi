package main

import (
	"fmt"
	"os"

	"github.com/la3mmchen/milbi/internal/app"
)

var (
	// AppVersion to be injected during build time
	AppVersion string
)

func main() {

	cfg := app.Milbi{
		AppName:    "milbi",
		AppUsage:   "backup and restore based on specific configs.",
		AppVersion: AppVersion,
	}

	app := app.CreateApp(&cfg)

	if err := app.Run(os.Args); err != nil {
		fmt.Printf("Error: %v \n", err)
		os.Exit(1)
	}
}
