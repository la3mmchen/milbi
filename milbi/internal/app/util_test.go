package app

import (
	"testing"
)

func TestPreload(t *testing.T) {
	cfg := getTestCfg()

	if err := preloadConfig(&cfg); err != nil {
		t.Log(err)
		t.FailNow()
	}

}
