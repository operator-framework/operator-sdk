package operator

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/operator-framework/operator-sdk/testutils/kubernetes"
	"github.com/operator-framework/operator-sdk/testutils/sample"
	kbutil "sigs.k8s.io/kubebuilder/v3/pkg/plugin/util"
)

// BuildOperatorImage will build an operator image by running `make docker-build IMG=<image>`
func BuildOperatorImage(sample sample.Sample, image string) error {
	cmd := exec.Command("make", "docker-build", "IMG="+image)
	_, err := sample.CommandContext().Run(cmd, sample.Name())
	if err != nil {
		fmt.Errorf("encountered an error when building the operator image: %w", err)
	}

	return nil
}

// DeployOperator will deploy an operator onto a Kubernetes cluster by running `make deploy IMG=<image>`
func DeployOperator(sample sample.Sample, image string) error {
	cmd := exec.Command("make", "deploy", "IMG="+image)
	_, err := sample.CommandContext().Run(cmd, sample.Name())
	if err != nil {
		fmt.Errorf("encountered an error when deploying the operator: %w", err)
	}

	return nil
}

// UndeployOperator will clean up an operator from a Kubernetes cluster by running `make undeploy`
func UndeployOperator(sample sample.Sample) error {
	cmd := exec.Command("make", "undeploy")
	_, err := sample.CommandContext().Run(cmd, sample.Name())
	if err != nil {
		fmt.Errorf("encountered an error when undeploying the operator: %w", err)
	}

	return nil
}

// EnsureOperatorRunning makes sure that an operator is running with with expected number of pods,
// the pod name contains a specific substring and is running in the specified control-plane
func EnsureOperatorRunning(kubectl kubernetes.Kubectl, expectedNumPods int, podNameShouldContain string, controlPlane string) error {
	// Get the controller-manager pod name
	podOutput, err := kubectl.Get(
		true,
		"pods", "-l", "control-plane="+controlPlane,
		"-o", "go-template={{ range .items }}{{ if not .metadata.deletionTimestamp }}{{ .metadata.name }}"+
			"{{ \"\\n\" }}{{ end }}{{ end }}")
	if err != nil {
		return fmt.Errorf("could not get pods: %v", err)
	}
	podNames := kbutil.GetNonEmptyLines(podOutput)
	if len(podNames) != expectedNumPods {
		return fmt.Errorf("expecting %d pod(s), have %d", expectedNumPods, len(podNames))
	}
	controllerPodName := podNames[0]
	if !strings.Contains(controllerPodName, podNameShouldContain) {
		return fmt.Errorf("expecting pod name %q to contain %q", controllerPodName, podNameShouldContain)
	}

	// Ensure the controller-manager Pod is running.
	status, err := kubectl.Get(
		true,
		"pods", controllerPodName, "-o", "jsonpath={.status.phase}")
	if err != nil {
		return fmt.Errorf("failed to get pod status for %q: %v", controllerPodName, err)
	}
	if status != "Running" {
		return fmt.Errorf("controller pod in %s status", status)
	}
	return nil
}

// InstallCRDs will install the CRDs for a sample onto the cluster
func InstallCRDs(sample sample.Sample) error {
	cmd := exec.Command("make", "install")
	_, err := sample.CommandContext().Run(cmd, sample.Name())
	return err
}

// UninstallCRDs will uninstall the CRDs for a sample from the cluster
func UninstallCRDs(sample sample.Sample) error {
	cmd := exec.Command("make", "uninstall")
	o, err := sample.CommandContext().Run(cmd, sample.Name())
	if err != nil {
		return fmt.Errorf("encountered an error uninstalling the CRDs: %w | OUTPUT: %s", err, o)
	}

	return nil
}
