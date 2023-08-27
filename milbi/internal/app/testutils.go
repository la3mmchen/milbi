package app

import (
	"log"
	"path/filepath"
	"runtime"

	"github.com/urfave/cli/v2"
)

func BootstrapTestApp() *cli.App {
	// construct an app for testing purposes
	cfg := getTestCfg()

	log.Println(cfg)
	return CreateApp(&cfg)
}

func getTestCfg() Milbi {
	// abstract away the correct location of the config for tests
	_, b, _, _ := runtime.Caller(0)
	configfile := filepath.Join(filepath.Dir(b), "../..", "/tests/manifests.yaml") // feels hacky but works

	return Milbi{
		AppName:    "milbi-test",
		AppVersion: "golang-test",
		AppUsage:   "a test invocation of milbi",
		Configfile: configfile,
		ExplainNew: false,
		AdditionalFlags: []cli.Flag{
			&cli.StringFlag{
				Name: "test.testlogfile",
			},
			&cli.StringFlag{
				Name: "test.paniconexit0",
			},
			&cli.StringFlag{
				Name: "test.v",
			},
		},
	}
}
