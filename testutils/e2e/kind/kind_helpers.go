package kind

import (
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
	_, err := cc.Run(cmd)
	return err
}
