package operator

import (
	"fmt"
	"os/exec"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/operator-framework/operator-sdk/testutils/kubernetes"
	"github.com/operator-framework/operator-sdk/testutils/sample"
	kbutil "sigs.k8s.io/kubebuilder/v3/pkg/plugin/util"
)

// LocalTest will run a few basic tests for testing that an operator will run locally (outside of a cluster)
func LocalTest(sample sample.Sample) {
	BeforeEach(func() {
		By("Installing CRD's")
		cmd := exec.Command("make", "install")
		cmdCtx := sample.CommandContext()
		_, err := cmdCtx.Run(cmd, sample.Name())
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		By("Uninstalling CRD's")
		cmd := exec.Command("make", "uninstall")
		cmdCtx := sample.CommandContext()
		_, err := cmdCtx.Run(cmd, sample.Name())
		Expect(err).NotTo(HaveOccurred())
	})

	It("Should run correctly when run locally", func() {
		By("Running the project")
		cmd := exec.Command("make", "run")
		err := cmd.Start()
		Expect(err).NotTo(HaveOccurred())

		By("Killing the project")
		err = cmd.Process.Kill()
		Expect(err).NotTo(HaveOccurred())
	})
}

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
