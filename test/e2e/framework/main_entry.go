package framework

import (
	"os"
	"testing"

	"github.com/prometheus/common/log"
)

func MainEntry(m *testing.M) {
	if err := setup(); err != nil {
		log.Errorf("Failed to set up framework: %v", err)
		os.Exit(1)
	}

	code := m.Run()
	os.Exit(code)
}
