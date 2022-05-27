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
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/operator-framework/operator-sdk/hack/generate/samples/helm"
	"github.com/operator-framework/operator-sdk/internal/testutils"
	"github.com/operator-framework/operator-sdk/internal/util"
	"github.com/operator-framework/operator-sdk/testutils/command"
	"github.com/operator-framework/operator-sdk/testutils/e2e/kind"
	"github.com/operator-framework/operator-sdk/testutils/e2e/olm"
	"github.com/operator-framework/operator-sdk/testutils/e2e/operator"
	"github.com/operator-framework/operator-sdk/testutils/e2e/prometheus"
	"github.com/operator-framework/operator-sdk/testutils/e2e/scorecard"
	"github.com/operator-framework/operator-sdk/testutils/kubernetes"
	"github.com/operator-framework/operator-sdk/testutils/sample"
)

//TODO: update this to use the PoC api

// TestE2EHelm ensures the Helm projects built with the SDK tool by using its binary.
func TestE2EHelm(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping Operator SDK E2E Helm Suite testing in short mode")
	}
	RegisterFailHandler(Fail)
	RunSpecs(t, "E2EHelm Suite")
}

var (
	tc                         testutils.TestContext
	kctl                       kubernetes.Kubectl
	isPrometheusManagedBySuite = true
	isOLMManagedBySuite        = true
	helmSample                 sample.Sample
	testdir                    = "e2e-test-helm"
	image                      = "e2e-test-helm:temp"
	helmSampleValidKubeConfig  sample.Sample
)

// BeforeSuite run before any specs are run to perform the required actions for all e2e Helm tests.
var _ = BeforeSuite(func() {
	var err error

	By("creating helm test samples")
	helmChartPath, err := filepath.Abs("../../../hack/generate/samples/helm/testdata/memcached-0.0.1.tgz")
	Expect(err).NotTo(HaveOccurred())
	samples := helm.GenerateMemcachedSamples(testutils.BinaryName, testdir, helmChartPath)
	helmSample = samples[0]

	// need to create a new sample for anything that requires a valid KUBECONFIG
	helmSampleValidKubeConfig = sample.NewGenericSample(
		sample.WithBinary(helmSample.Binary()),
		sample.WithGvk(helmSample.GVKs()...),
		sample.WithDomain(helmSample.Domain()),
		sample.WithName(helmSample.Name()),
		sample.WithCommandContext(command.NewGenericCommandContext(
			command.WithDir(helmSample.CommandContext().Dir()),
		)),
	)

	kctl = kubernetes.NewKubectlUtil(
		kubernetes.WithCommandContext(
			command.NewGenericCommandContext(
				command.WithDir(helmSample.Dir()),
			),
		),
		kubernetes.WithNamespace(helmSample.Name()+"-system"),
		kubernetes.WithServiceAccount(helmSample.Name()+"-controller-manager"),
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
		Expect(olm.InstallOLMVersion(helmSampleValidKubeConfig, olm.OlmVersionForTestSuite)).To(Succeed())
	}

	By("using dev image for scorecard-test")
	err = scorecard.ReplaceScorecardImagesForDev(helmSample)
	Expect(err).NotTo(HaveOccurred())

	By("replacing project Dockerfile to use Helm base image with the dev tag")
	err = util.ReplaceRegexInFile(filepath.Join(helmSample.Dir(), "Dockerfile"), "quay.io/operator-framework/helm-operator:.*", "quay.io/operator-framework/helm-operator:dev")
	Expect(err).Should(Succeed())

	By("checking the kustomize setup")
	cmd := exec.Command("make", "kustomize")
	_, err = helmSample.CommandContext().Run(cmd, helmSample.Name())
	Expect(err).NotTo(HaveOccurred())

	By("building the project image")
	err = operator.BuildOperatorImage(helmSample, image)
	Expect(err).NotTo(HaveOccurred())

	onKind, err := kind.IsRunningOnKind(kctl)
	Expect(err).NotTo(HaveOccurred())
	if onKind {
		By("loading the required images into Kind cluster")
		Expect(kind.LoadImageToKindCluster(helmSample.CommandContext(), image)).To(Succeed())
		Expect(kind.LoadImageToKindCluster(helmSample.CommandContext(), "quay.io/operator-framework/scorecard-test:dev")).To(Succeed())
	}

	By("generating bundle")
	Expect(olm.GenerateBundle(helmSample, "bundle-"+image)).To(Succeed())
})

// AfterSuite run after all the specs have run, regardless of whether any tests have failed to ensures that
// all be cleaned up
var _ = AfterSuite(func() {
	By("uninstalling prerequisites")
	if isPrometheusManagedBySuite {
		By("uninstalling Prometheus")
		prometheus.UninstallPrometheusOperator(kctl)
	}
	if isOLMManagedBySuite {
		By("uninstalling OLM")
		olm.UninstallOLM(helmSample)
	}

	By("destroying container image and work dir")
	cmd := exec.Command("docker", "rmi", "-f", image)
	if _, err := helmSample.CommandContext().Run(cmd); err != nil {
		Expect(err).To(BeNil())
	}
	if err := os.RemoveAll(testdir); err != nil {
		Expect(err).To(BeNil())
	}
})
