package kind

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/operator-framework/operator-sdk/testutils/command"
	"github.com/operator-framework/operator-sdk/testutils/kubernetes"
)

// IsRunningOnKind checks if the Kubernetes cluster is a KinD cluster
func IsRunningOnKind(kubectl kubernetes.Kubectl) (bool, error) {
	kubectx, err := kubectl.Command("config", "current-context")
	if err != nil {
		return false, err
	}
	return strings.Contains(kubectx, "kind"), nil
}

// LoadImageToKindCluster will load an image onto a KinD cluster
func LoadImageToKindCluster(cc command.CommandContext, image string) error {
	cluster := "kind"
	if v, ok := os.LookupEnv("KIND_CLUSTER"); ok {
		cluster = v
	}
	kindOptions := []string{"load", "docker-image", image, "--name", cluster}
	cmd := exec.Command("kind", kindOptions...)
	o, err := cc.Run(cmd)
	if err != nil {
		return fmt.Errorf("encountered an error attempting to load image to KinD cluster: %w | OUTPUT: %s", err, o)
	}
	return nil
}
