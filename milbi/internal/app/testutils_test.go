package app

import (
	"fmt"
	"testing"
)

func TestBootstrapTestApp(t *testing.T) {
	cfg := getTestCfg()

	if fmt.Sprintf("%T", cfg) != "app.Milbi" {
		t.Fail()
	}
}

func TestGetTestCfg(t *testing.T) {
	_ = getTestCfg()
}
