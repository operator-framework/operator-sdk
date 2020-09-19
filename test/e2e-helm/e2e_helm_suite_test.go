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

package e2e_helm_test

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	testutils "github.com/operator-framework/operator-sdk/test/internal"
)

// TestE2EHelm ensures the Helm projects built with the SDK tool by using its binary.
func TestE2EHelm(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping Operator SDK E2E Helm Suite testing in short mode")
	}
	RegisterFailHandler(Fail)
	RunSpecs(t, "E2EHelm Suite")
}

var (
	tc testutils.TestContext
	// isPrometheusManagedBySuite is true when the suite tests is installing/uninstalling the Prometheus
	isPrometheusManagedBySuite = true
	// isOLMManagedBySuite is true when the suite tests is installing/uninstalling the OLM
	isOLMManagedBySuite = true
	// kubectx stores the k8s context from where the tests are running
	kubectx string
	// projectName is the name of the test project
	projectName string
)

// BeforeSuite run before any specs are run to perform the required actions for all e2e Helm tests.
var _ = BeforeSuite(func(done Done) {
	var err error

	By("creating a new test context")
	tc, err = testutils.NewTestContext("GO111MODULE=on")
	Expect(err).NotTo(HaveOccurred())
	Expect(tc.Prepare()).To(Succeed())
	projectName = filepath.Base(tc.Dir)

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
		Expect(tc.InstallOLM()).To(Succeed())
	}

	By("initializing a Helm project")
	err = tc.Init(
		"--plugins", "helm",
		"--project-version", "3-alpha",
		"--domain", tc.Domain)
	Expect(err).NotTo(HaveOccurred())

	By("using dev image for scorecard-test")
	tc.ReplaceScorecardImagesForDev()

	By("creating an API definition")
	err = tc.CreateAPI(
		"--group", tc.Group,
		"--version", tc.Version,
		"--kind", tc.Kind)
	Expect(err).NotTo(HaveOccurred())

	By("replacing project Dockerfile to use Helm base image with the dev tag")
	testutils.ReplaceRegexInFile(filepath.Join(tc.Dir, "Dockerfile"), "quay.io/operator-framework/helm-operator:.*", "quay.io/operator-framework/helm-operator:dev")

	By("checking the kustomize setup")
	err = tc.Make("kustomize")
	Expect(err).NotTo(HaveOccurred())

	By("building the project image")
	err = tc.Make("docker-build", "IMG="+tc.ImageName)
	Expect(err).NotTo(HaveOccurred())

	if isRunningOnKind() {
		By("loading the project image into Kind cluster")
		err = tc.LoadImageToKindCluster()
		Expect(err).NotTo(HaveOccurred())
	}

	close(done)
}, 360)

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
