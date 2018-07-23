package framework

import (
	"log"
	"os"
	"testing"
)

func MainEntry(m *testing.M) {
	if err := setup(); err != nil {
		log.Fatalf("Failed to set up framework: %v", err)
	}

	os.Exit(m.Run())
}
