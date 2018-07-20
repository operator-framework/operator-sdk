package e2e

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	// TODO: create a setup step for the framework here
	os.Exit(m.Run())
}
