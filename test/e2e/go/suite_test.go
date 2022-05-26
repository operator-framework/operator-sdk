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
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	golang "github.com/operator-framework/operator-sdk/hack/generate/samples/go"
	"github.com/operator-framework/operator-sdk/internal/testutils"
	"github.com/operator-framework/operator-sdk/testutils/command"
	"github.com/operator-framework/operator-sdk/testutils/e2e/certmanager"
	"github.com/operator-framework/operator-sdk/testutils/e2e/kind"
	"github.com/operator-framework/operator-sdk/testutils/e2e/olm"
	"github.com/operator-framework/operator-sdk/testutils/e2e/operator"
	"github.com/operator-framework/operator-sdk/testutils/e2e/prometheus"
	"github.com/operator-framework/operator-sdk/testutils/e2e/scorecard"
	"github.com/operator-framework/operator-sdk/testutils/kubernetes"
	"github.com/operator-framework/operator-sdk/testutils/sample"
)

//TODO: update this to use the new PoC api

// TestE2EGo ensures the Go projects built with the SDK tool by using its binary.
func TestE2EGo(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping Operator SDK E2E Go Suite testing in short mode")
	}
	RegisterFailHandler(Fail)
	RunSpecs(t, "E2EGo Suite")
}

var (
	tc                         testutils.TestContext
	kctl                       kubernetes.Kubectl
	isPrometheusManagedBySuite = true
	isOLMManagedBySuite        = true
	goSample                   sample.Sample
	testdir                    = "e2e-test-go"
	image                      = "e2e-test:go"
)

// BeforeSuite run before any specs are run to perform the required actions for all e2e Go tests.
var _ = BeforeSuite(func() {
	var err error

	By("creating Go test samples")
	samples := golang.GenerateMemcachedSamples(testutils.BinaryName, testdir)
	goSample = samples[0]

	kctl = kubernetes.NewKubectlUtil(
		kubernetes.WithCommandContext(
			command.NewGenericCommandContext(
				command.WithDir(goSample.Dir()),
			),
		),
		kubernetes.WithNamespace(goSample.Name()+"-system"),
		kubernetes.WithServiceAccount(goSample.Name()+"-controller-manager"),
	)

	By("preparing the prerequisites on cluster")
	By("checking API resources applied on Cluster")
	output, err := kctl.Command("api-resources")
	Expect(err).NotTo(HaveOccurred())
	if strings.Contains(output, "servicemonitors") {
		isPrometheusManagedBySuite = false
	}
	if strings.Contains(output, "clusterserviceversions") {
		isOLMManagedBySuite = false
	}

	if isPrometheusManagedBySuite {
		By("installing Prometheus")
		Expect(prometheus.InstallPrometheusOperator(kctl)).To(Succeed())

		By("ensuring provisioned Prometheus Manager Service")
		Eventually(func() error {
			_, err := kctl.Get(
				false,
				"Service", "prometheus-operator")
			return err
		}, 3*time.Minute, time.Second).Should(Succeed())
	}

	if isOLMManagedBySuite {
		By("installing OLM")
		Expect(olm.InstallOLMVersion(goSample, olm.OlmVersionForTestSuite)).To(Succeed())
	}

	By("by adding scorecard custom patch file")
	err = scorecard.AddScorecardCustomPatchFile(goSample)
	Expect(err).NotTo(HaveOccurred())

	By("using dev image for scorecard-test")
	err = scorecard.ReplaceScorecardImagesForDev(goSample)
	Expect(err).NotTo(HaveOccurred())

	By("building the project image")
	err = operator.BuildOperatorImage(goSample, image)
	Expect(err).NotTo(HaveOccurred())

	onKind, err := kind.IsRunningOnKind(kctl)
	Expect(err).NotTo(HaveOccurred())
	if onKind {
		By("loading the required images into Kind cluster")
		Expect(kind.LoadImageToKindCluster(goSample.CommandContext(), image)).To(Succeed())
		Expect(kind.LoadImageToKindCluster(goSample.CommandContext(), "quay.io/operator-framework/scorecard-test:dev")).To(Succeed())
		Expect(kind.LoadImageToKindCluster(goSample.CommandContext(), "quay.io/operator-framework/custom-scorecard-tests:dev")).To(Succeed())
	}

	By("generating bundle")
	Expect(olm.GenerateBundle(goSample, "bundle-"+image)).To(Succeed())

	By("installing cert manager bundle")
	Expect(certmanager.InstallCertManagerBundle(false, kctl)).To(Succeed())
})

// AfterSuite run after all the specs have run, regardless of whether any tests have failed to ensures that
// all be cleaned up
var _ = AfterSuite(func() {
	By("uninstall cert manager bundle")
	certmanager.UninstallCertManagerBundle(false, kctl)

	By("uninstalling prerequisites")
	if isPrometheusManagedBySuite {
		By("uninstalling Prometheus")
		prometheus.UninstallPrometheusOperator(kctl)
	}
	if isOLMManagedBySuite {
		By("uninstalling OLM")
		olm.UninstallOLM(goSample)
	}

	By("destroying container image and work dir")
	cmd := exec.Command("docker", "rmi", "-f", image)
	if _, err := goSample.CommandContext().Run(cmd); err != nil {
		Expect(err).To(BeNil())
	}
	if err := os.RemoveAll(testdir); err != nil {
		Expect(err).To(BeNil())
	}
})
