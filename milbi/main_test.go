package main

import (
	"os"
	"strings"
	"testing"

	"github.com/la3mmchen/milbi/internal/app"
)

func TestAppRun(t *testing.T) {
	subCmd := []string{}
	app := app.BootstrapTestApp()

	args := os.Args[0:1]
	args = append(args, subCmd...)

	if err := app.Run(args); err != nil {
		t.Fail()
		t.Logf("cli command [%v] failed. Error: %v", strings.Join(subCmd, ", "), err)
	}
}

func getGenericFlags() []string {

	return []string{
		"--config",
		"./tests/manifests.yaml",
	}
}

func TestSubcmds(t *testing.T) {
	genericFlags := getGenericFlags()

	pre := map[string][]string{
		"init": {"init"},
	}

	cases := map[string][]string{
		"check":            {"check"},
		"explain":          {"explain"},
		"explainNew":       {"explain", "--new"},
		"info":             {"info"},
		"prune":            {"prune"},
		"snapshot":         {"snapshot"},
		"snapshotSimulate": {"snapshot", "--simulate"},
		"sync":             {"sync"},
	}

	// delete an existing test-repo
	os.RemoveAll("./tests/files/resticrepo")

	args := os.Args[0:1]

	// create a local repo for the test
	for testcase, subcmds := range pre {
		// create a new test app
		app := app.BootstrapTestApp()
		argsCpy := args
		t.Logf("__Test: [%v]", testcase)
		argsCpy = append(argsCpy, genericFlags...)
		argsCpy = append(argsCpy, subcmds...)
		t.Logf("%v", argsCpy)

		if err := app.Run(argsCpy); err != nil {
			t.Logf("SubCmd [%v]: cli command [%v] failed. Error: %v", testcase, strings.Join(subcmds, ", "), err)
			t.FailNow()
		}
	}

	// execute tests
	for testcase, subcmds := range cases {
		// create a new test app
		app := app.BootstrapTestApp()
		argsCpy := args
		t.Logf("__Test: [%v]", testcase)
		argsCpy = append(argsCpy, genericFlags...)
		argsCpy = append(argsCpy, subcmds...)
		t.Logf("%v", argsCpy)

		if err := app.Run(argsCpy); err != nil {
			t.Logf("SubCmd [%v]: cli command [%v] failed. Error: %v", testcase, strings.Join(subcmds, ", "), err)
			t.FailNow()
		}
	}
}
