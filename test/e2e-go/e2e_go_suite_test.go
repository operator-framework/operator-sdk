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

package e2e_go_test

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	kbtestutils "sigs.k8s.io/kubebuilder/test/e2e/utils"

	"github.com/operator-framework/operator-sdk/internal/testutils"
)

// TestE2EGo ensures the Go projects built with the SDK tool by using its binary.
func TestE2EGo(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping Operator SDK E2E Go Suite testing in short mode")
	}
	RegisterFailHandler(Fail)
	RunSpecs(t, "E2EGo Suite")
}

var (
	tc testutils.TestContext
	// isPrometheusManagedBySuite is true when the suite tests is installing/uninstalling the Prometheus
	isPrometheusManagedBySuite = true
	// isOLMManagedBySuite is true when the suite tests is installing/uninstalling the OLM
	isOLMManagedBySuite = true
	// kubectx stores the k8s context from where the tests are running
	kubectx string
)

// BeforeSuite run before any specs are run to perform the required actions for all e2e Go tests.
var _ = BeforeSuite(func() {
	var err error

	By("creating a new test context")
	tc, err = testutils.NewTestContext(testutils.BinaryName, "GO111MODULE=on")
	Expect(err).NotTo(HaveOccurred())

	By("creating a new directory")
	Expect(tc.Prepare()).To(Succeed())

	By("checking the cluster type")
	kubectx, err = tc.Kubectl.Command("config", "current-context")
	Expect(err).NotTo(HaveOccurred())

	By("checking API resources applied on Cluster")
	output, err := tc.Kubectl.Command("api-resources")
	Expect(err).NotTo(HaveOccurred())
	if strings.Contains(output, "servicemonitors") {
		isPrometheusManagedBySuite = false
	}
	if strings.Contains(output, "clusterserviceversions") {
		isOLMManagedBySuite = false
	}

	if isPrometheusManagedBySuite {
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

	if isOLMManagedBySuite {
		By("installing OLM")
		Expect(tc.InstallOLMVersion(testutils.OlmVersionForTestSuite)).To(Succeed())
	}

	By("initializing a project")
	projectName := filepath.Base(tc.Dir)
	err = tc.Init(
		"--project-version", "3-alpha",
		"--repo", path.Join("github.com", "example", projectName),
		"--domain", tc.Domain,
		"--fetch-deps=false")
	Expect(err).NotTo(HaveOccurred())

	By("by adding scorecard custom patch file")
	err = tc.AddScorecardCustomPatchFile()
	Expect(err).NotTo(HaveOccurred())

	By("using dev image for scorecard-test")
	err = tc.ReplaceScorecardImagesForDev()
	Expect(err).NotTo(HaveOccurred())

	By("creating an API definition")
	err = tc.CreateAPI(
		"--group", tc.Group,
		"--version", tc.Version,
		"--kind", tc.Kind,
		"--namespaced",
		"--resource",
		"--controller",
		"--make=false")
	Expect(err).NotTo(HaveOccurred())

	By("implementing the API")
	Expect(kbtestutils.InsertCode(
		filepath.Join(tc.Dir, "api", tc.Version, fmt.Sprintf("%s_types.go", strings.ToLower(tc.Kind))),
		fmt.Sprintf(`type %sSpec struct {
`, tc.Kind),
		`	// +optional
	Count int `+"`"+`json:"count,omitempty"`+"`"+`
`)).Should(Succeed())

	By("enabling Prometheus via the kustomization.yaml")
	Expect(kbtestutils.UncommentCode(
		filepath.Join(tc.Dir, "config", "default", "kustomization.yaml"),
		"#- ../prometheus", "#")).To(Succeed())

	By("turning off interactive prompts for all generation tasks.")
	err = tc.DisableOLMBundleInteractiveMode()
	Expect(err).NotTo(HaveOccurred())

	By("checking the kustomize setup")
	err = tc.Make("kustomize")
	Expect(err).NotTo(HaveOccurred())

	By("building the project image")
	err = tc.Make("docker-build", "IMG="+tc.ImageName)
	Expect(err).NotTo(HaveOccurred())

	if isRunningOnKind() {
		By("loading the required images into Kind cluster")
		Expect(tc.LoadImageToKindCluster()).To(Succeed())
		Expect(tc.LoadImageToKindClusterWithName("quay.io/operator-framework/scorecard-test:dev")).To(Succeed())
		Expect(tc.LoadImageToKindClusterWithName("quay.io/operator-framework/custom-scorecard-tests:dev")).To(Succeed())
	}

	By("generating the operator bundle")
	err = tc.Make("bundle", "IMG="+tc.ImageName)
	Expect(err).NotTo(HaveOccurred())
})

// AfterSuite run after all the specs have run, regardless of whether any tests have failed to ensures that
// all be cleaned up
var _ = AfterSuite(func() {
	if isPrometheusManagedBySuite {
		By("uninstalling Prometheus")
		tc.UninstallPrometheusOperManager()
	}
	if isOLMManagedBySuite {
		By("uninstalling OLM")
		tc.UninstallOLM()
	}

	By("destroying container image and work dir")
	tc.Destroy()
})

// isRunningOnKind returns true when the tests are executed in a Kind Cluster
func isRunningOnKind() bool {
	return strings.Contains(kubectx, "kind")
}
