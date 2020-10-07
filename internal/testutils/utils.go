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
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	kbtestutils "sigs.k8s.io/kubebuilder/test/e2e/utils"
)

const BinaryName = "operator-sdk"

// TestContext wraps kubebuilder's e2e TestContext.
type TestContext struct {
	*kbtestutils.TestContext
	// BundleImageName store the image to use to build the bundle
	BundleImageName string
	// ProjectName store the project name
	ProjectName string
	// IsPrometheusManagedBySuite is true when the suite tests is installing/uninstalling the Prometheus
	IsPrometheusManagedBySuite bool
	// IsOLMManagedBySuite is true when the suite tests is installing/uninstalling the OLM
	IsOLMManagedBySuite bool
	// Kubectx stores the k8s context from where the tests are running
	Kubectx string
}

// NewTestContext returns a TestContext containing a new kubebuilder TestContext.
func NewTestContext(binary string, env ...string) (tc TestContext, err error) {
	tc.TestContext, err = kbtestutils.NewTestContext(binary, env...)
	tc.ProjectName = strings.ToLower(filepath.Base(tc.Dir))
	tc.ImageName = fmt.Sprintf("quay.io/example/%s:v0.0.1", tc.ProjectName)
	tc.BundleImageName = fmt.Sprintf("quay.io/example/%s-bundle:v0.0.1", tc.ProjectName)
	tc.IsOLMManagedBySuite = true
	tc.IsPrometheusManagedBySuite = true
	return tc, err
}

// InstallOLM runs 'operator-sdk olm install' and returns any errors emitted by that command.
func (tc TestContext) InstallOLM() error {
	err := tc.InstallOLMVersion("latest")
	return err
}

// InstallOLM runs 'operator-sdk olm install' for specific version
// and returns any errors emitted by that command.
func (tc TestContext) InstallOLMVersion(version string) error {
	cmd := exec.Command(tc.BinaryName, "olm", "install", "--version", version, "--timeout", "4m")
	_, err := tc.Run(cmd)
	return err
}

// InstallOLM runs 'operator-sdk olm uninstall' and logs any errors emitted by that command.
func (tc TestContext) UninstallOLM() {
	cmd := exec.Command(tc.BinaryName, "olm", "uninstall")
	if _, err := tc.Run(cmd); err != nil {
		fmt.Fprintln(GinkgoWriter, "warning: error when uninstalling OLM:", err)
	}
}

// KustomizeBuild runs 'kustomize build <dir>' and returns its output and an error if any.
func (tc TestContext) KustomizeBuild(dir string) ([]byte, error) {
	return tc.Run(exec.Command("kustomize", "build", dir))
}

// ReplaceInFile replaces all instances of old with new in the file at path.
// todo(camilamacedo86): this func can be pushed to upstream/kb
func ReplaceInFile(path, old, new string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	if !strings.Contains(string(b), old) {
		return errors.New("unable to find the content to be replaced")
	}
	s := strings.Replace(string(b), old, new, -1)
	err = ioutil.WriteFile(path, []byte(s), info.Mode())
	if err != nil {
		return err
	}
	return nil
}

// ReplaceRegexInFile finds all strings that match `match` and replaces them
// with `replace` in the file at path.
// todo(camilamacedo86): this func can be pushed to upstream/kb
func ReplaceRegexInFile(path, match, replace string) error {
	matcher, err := regexp.Compile(match)
	if err != nil {
		return err
	}
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	s := matcher.ReplaceAllString(string(b), replace)
	if err != nil {
		return errors.New("unable to find the content to be replaced")
	}
	err = ioutil.WriteFile(path, []byte(s), info.Mode())
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

// UncommentCode searches for target in the file and remove the comment prefix
// of the target content. The target content may span multiple lines.
// todo(camilamacedo86): this func exists in upstream/kb but there the error is not thrown. We need to
// push this change. See: https://github.com/kubernetes-sigs/kubebuilder/blob/master/test/e2e/utils/util.go
func UncommentCode(filename, target, prefix string) error {
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	strContent := string(content)

	idx := strings.Index(strContent, target)
	if idx < 0 {
		// todo: push this check to upstream for we do not need have this func here
		return fmt.Errorf("unable to find the code %s to be uncomment", target)
	}

	out := new(bytes.Buffer)
	_, err = out.Write(content[:idx])
	if err != nil {
		return err
	}

	strs := strings.Split(target, "\n")
	for _, str := range strs {
		_, err := out.WriteString(strings.TrimPrefix(str, prefix) + "\n")
		if err != nil {
			return err
		}
	}

	_, err = out.Write(content[idx+len(target):])
	if err != nil {
		return err
	}
	// false positive
	// nolint:gosec
	return ioutil.WriteFile(filename, out.Bytes(), 0644)
}

// InstallPrerequisites will install OLM and Prometheus
// when the cluster kind is Kind and when they are not present on the Cluster
func (tc TestContext) InstallPrerequisites() {
	By("checking API resources applied on Cluster")
	output, err := tc.Kubectl.Command("api-resources")
	Expect(err).NotTo(HaveOccurred())
	if strings.Contains(output, "servicemonitors") {
		tc.IsPrometheusManagedBySuite = false
	}
	if strings.Contains(output, "clusterserviceversions") {
		tc.IsOLMManagedBySuite = false
	}

	if tc.IsPrometheusManagedBySuite {
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

	if tc.IsOLMManagedBySuite {
		By("installing OLM")
		Expect(tc.InstallOLMVersion(OlmVersionForTestSuite)).To(Succeed())
	}
}

// IsRunningOnKind returns true when the tests are executed in a Kind Cluster
func (tc TestContext) IsRunningOnKind() bool {
	return strings.Contains(tc.Kubectx, "kind")
}

// UninstallPrerequisites will uninstall all prerequisites installed via InstallPrerequisites()
func (tc TestContext) UninstallPrerequisites() {
	if tc.IsPrometheusManagedBySuite {
		By("uninstalling Prometheus")
		tc.UninstallPrometheusOperManager()
	}
	if tc.IsOLMManagedBySuite {
		By("uninstalling OLM")
		tc.UninstallOLM()
	}
}
