package e2e

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/operator-framework/operator-sdk/testutils/kubernetes"
	"github.com/operator-framework/operator-sdk/testutils/sample"
)

// CleanUpTestDir removes the test directory
func CleanUpTestDir(path string) error {
	return os.RemoveAll(path)
}

// CreateCustomResources will create the CRs that are specified in a Sample
func CreateCustomResources(sample sample.Sample, kubectl kubernetes.Kubectl) error {
	for _, gvk := range sample.GVKs() {
		sampleFile := filepath.Join(sample.CommandContext().Dir(),
			sample.Name(),
			"config",
			"sample",
			fmt.Sprintf("%s_%s_%s.yaml", gvk.Group, gvk.Version, strings.ToLower(gvk.Kind)))

		_, err := kubectl.Apply(true, "-f", sampleFile)
		if err != nil {
			return fmt.Errorf("encountered an error when applying CRD (%s): %w", sampleFile, err)
		}
	}

	return nil
}
