// Copyright 2020 The Operator-SDK Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package testutils

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	kbutil "sigs.k8s.io/kubebuilder/v4/pkg/plugin/util"
	kbtestutils "sigs.k8s.io/kubebuilder/v4/test/e2e/utils"
)

const BinaryName = "operator-sdk"

// TestContext wraps kubebuilder's e2e TestContext.
type TestContext struct {
	*kbtestutils.TestContext
	// BundleImageName store the image to use to build the bundle
	BundleImageName string
	// ProjectName store the project name
	ProjectName string
	// isPrometheusManagedBySuite is true when the suite tests is installing/uninstalling the Prometheus
	isPrometheusManagedBySuite bool
	// isOLMManagedBySuite is true when the suite tests is installing/uninstalling the OLM
	isOLMManagedBySuite bool
}

// NewTestContext returns a TestContext containing a new kubebuilder TestContext.
// Construct if your environment is connected to a live cluster, ex. for e2e tests.
func NewTestContext(binaryName string, env ...string) (tc TestContext, err error) {
	if tc.TestContext, err = kbtestutils.NewTestContext(binaryName, env...); err != nil {
		return tc, err
	}
	tc.ProjectName = strings.ToLower(filepath.Base(tc.Dir))
	tc.ImageName = makeImageName(tc.ProjectName)
	tc.BundleImageName = makeBundleImageName(tc.ProjectName)
	tc.isOLMManagedBySuite = true
	tc.isPrometheusManagedBySuite = true
	return tc, nil
}

// NewPartialTestContext returns a TestContext containing a partial kubebuilder TestContext.
// This object needs to be populated with GVK information. The underlying TestContext is
// created directly rather than through a constructor so cluster-based setup is skipped.
func NewPartialTestContext(binaryName, dir string, env ...string) (tc TestContext, err error) {
	cc := &kbtestutils.CmdContext{
		Env: env,
	}
	if cc.Dir, err = filepath.Abs(dir); err != nil {
		return tc, err
	}
	projectName := strings.ToLower(filepath.Base(dir))

	return TestContext{
		TestContext: &kbtestutils.TestContext{
			CmdContext: cc,
			BinaryName: binaryName,
			ImageName:  makeImageName(projectName),
		},
		ProjectName:     projectName,
		BundleImageName: makeBundleImageName(projectName),
	}, nil
}

func makeImageName(projectName string) string {
	return fmt.Sprintf("quay.io/example/%s:v0.0.1", projectName)
}

func makeBundleImageName(projectName string) string {
	return fmt.Sprintf("quay.io/example/%s-bundle:v0.0.1", projectName)
}

// InstallOLMVersion runs 'operator-sdk olm install' for specific version
// and returns any errors emitted by that command.
func (tc TestContext) InstallOLMVersion(version string) error {
	cmd := exec.Command(tc.BinaryName, "olm", "install", "--version", version, "--timeout", "4m")
	_, err := tc.Run(cmd)
	return err
}

// UninstallOLM runs 'operator-sdk olm uninstall' and logs any errors emitted by that command.
func (tc TestContext) UninstallOLM() {
	cmd := exec.Command(tc.BinaryName, "olm", "uninstall")
	if _, err := tc.Run(cmd); err != nil {
		fmt.Fprintln(GinkgoWriter, "warning: error when uninstalling OLM:", err)
	}
}

// ReplaceInFile replaces all instances of old with new in the file at path.
// todo(camilamacedo86): this func can be pushed to upstream/kb
func ReplaceInFile(path, o, n string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if !strings.Contains(string(b), o) {
		return errors.New("unable to find the content to be replaced")
	}
	s := strings.Replace(string(b), o, n, -1)
	err = os.WriteFile(path, []byte(s), info.Mode())
	if err != nil {
		return err
	}
	return nil
}

// LoadImageToKindClusterWithName loads a local docker image with the name informed to the kind cluster
func (tc TestContext) LoadImageToKindClusterWithName(image string) error {
	cluster := "kind"
	if v, ok := os.LookupEnv("KIND_CLUSTER"); ok {
		cluster = v
	}
	kindOptions := []string{"load", "docker-image", "--name", cluster, image}
	cmd := exec.Command("kind", kindOptions...)
	_, err := tc.Run(cmd)
	return err
}

// InstallPrerequisites will install OLM and Prometheus
// when the cluster kind is Kind and when they are not present on the Cluster
func (tc TestContext) InstallPrerequisites() {
	By("checking API resources applied on Cluster")
	output, err := tc.Kubectl.Command("api-resources")
	Expect(err).NotTo(HaveOccurred())
	if strings.Contains(output, "servicemonitors") {
		tc.isPrometheusManagedBySuite = false
	}
	if strings.Contains(output, "clusterserviceversions") {
		tc.isOLMManagedBySuite = false
	}

	if tc.isPrometheusManagedBySuite {
		By("installing Prometheus")
		Expect(tc.InstallPrometheusOperManager()).To(Succeed())

		By("ensuring provisioned Prometheus Manager Service")
		Eventually(func() error {
			_, err := tc.Kubectl.Get(
				false,
				"Service", "prometheus-operator")
			return err
		}, 3*time.Minute, time.Second).Should(Succeed())
	}

	if tc.isOLMManagedBySuite {
		By("installing OLM")
		Expect(tc.InstallOLMVersion(OlmVersionForTestSuite)).To(Succeed())
	}
}

// IsRunningOnKind returns true when the tests are executed in a Kind Cluster
func (tc TestContext) IsRunningOnKind() (bool, error) {
	kubectx, err := tc.Kubectl.Command("config", "current-context")
	if err != nil {
		return false, err
	}
	return strings.Contains(kubectx, "kind"), nil
}

// UninstallPrerequisites will uninstall all prerequisites installed via InstallPrerequisites()
func (tc TestContext) UninstallPrerequisites() {
	if tc.isPrometheusManagedBySuite {
		By("uninstalling Prometheus")
		tc.UninstallPrometheusOperManager()
	}
	if tc.isOLMManagedBySuite {
		By("uninstalling OLM")
		tc.UninstallOLM()
	}
}

// WrapWarnOutput is a one-liner to wrap an error from a command that returns (string, error) in a warning.
func WrapWarnOutput(_ string, err error) {
	if err != nil {
		fmt.Fprintf(GinkgoWriter, "warning: %s", err)
	}
}

// WrapWarn is a one-liner to wrap an error from a command that returns (error) in a warning.
func WrapWarn(err error) {
	WrapWarnOutput("", err)
}

func (tc TestContext) UncommentRestrictivePodStandards() error {
	configManager := filepath.Join(tc.Dir, "config", "manager", "manager.yaml")

	if err := kbutil.ReplaceInFile(configManager, `# TODO(user): For common cases that do not require escalating privileges
        # it is recommended to ensure that all your Pods/Containers are restrictive.
        # More info: https://kubernetes.io/docs/concepts/security/pod-security-standards/#restricted
        # Please uncomment the following code if your project does NOT have to work on old Kubernetes
        # versions < 1.19 or on vendors versions which do NOT support this field by default (i.e. Openshift < 4.11 ).
        # seccompProfile:
        #   type: RuntimeDefault`, `seccompProfile:
          type: RuntimeDefault`); err == nil {
		return err
	}

	return nil
}
