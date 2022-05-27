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

package integration

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/operator-framework/operator-registry/pkg/lib/bundle"
	"sigs.k8s.io/yaml"

	golang "github.com/operator-framework/operator-sdk/hack/generate/samples/go"
	"github.com/operator-framework/operator-sdk/internal/testutils"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"
	"github.com/operator-framework/operator-sdk/testutils/command"
	"github.com/operator-framework/operator-sdk/testutils/e2e/certmanager"
	"github.com/operator-framework/operator-sdk/testutils/e2e/kind"
	"github.com/operator-framework/operator-sdk/testutils/e2e/olm"
	"github.com/operator-framework/operator-sdk/testutils/e2e/operator"
	"github.com/operator-framework/operator-sdk/testutils/e2e/prometheus"
	"github.com/operator-framework/operator-sdk/testutils/kubernetes"
	"github.com/operator-framework/operator-sdk/testutils/sample"
)

//TODO: update to use the new PoC api

// TestIntegration tests operator-sdk projects with OLM.
func TestIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping Integration Suite in short mode")
	}
	RegisterFailHandler(Fail)
	RunSpecs(t, "Integration")
}

var (
	tc                         testutils.TestContext
	kctl                       kubernetes.Kubectl
	isPrometheusManagedBySuite = true
	isOLMManagedBySuite        = true
	goSample                   sample.Sample
	testdir                    = "e2e-test-integration"
	image                      = "e2e-test-integration:temp"
)

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

	By("updating the project configuration")
	Expect(olm.AddPackagemanifestsTarget(goSample, projutil.OperatorTypeGo)).To(Succeed())

	By("building the manager image")
	err = operator.BuildOperatorImage(goSample, image)
	Expect(err).NotTo(HaveOccurred())

	onKind, err := kind.IsRunningOnKind(kctl)
	Expect(err).NotTo(HaveOccurred())
	if onKind {
		By("loading the required images into Kind cluster")
		Expect(kind.LoadImageToKindCluster(goSample.CommandContext(), image)).To(Succeed())
	}

	By("generating the operator package manifests and enabling all InstallModes")
	Expect(olm.GeneratePackageManifests(goSample, image)).To(Succeed())
	csv, err := readCSV(goSample, "0.0.1", false)
	Expect(err).NotTo(HaveOccurred())
	for i := range csv.Spec.InstallModes {
		csv.Spec.InstallModes[i].Supported = true
	}
	Expect(writeCSV(goSample, "0.0.1", csv, false)).To(Succeed())

	// TODO(estroz): enable when bundles can be tested locally.
	//
	// By("generating the operator bundle")
	// err = tc.Make("bundle", "IMG="+tc.ImageName)
	// Expect(err).NotTo(HaveOccurred())
	//
	// By("building the operator bundle image")
	// err = tc.Make("bundle-build", "BUNDLE_IMG="+tc.BundleImageName)
	// Expect(err).NotTo(HaveOccurred())

	By("creating the test namespace")
	_, err = kctl.Command("create", "namespace", kctl.Namespace())
	Expect(err).NotTo(HaveOccurred())
})

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

	By("deleting the test namespace")
	warn(kctl.Delete(false, "namespace", kctl.Namespace()))

	By("destroying container image and work dir")
	cmd := exec.Command("docker", "rmi", "-f", image)
	if _, err := goSample.CommandContext().Run(cmd); err != nil {
		Expect(err).To(BeNil())
	}
	if err := os.RemoveAll(testdir); err != nil {
		Expect(err).To(BeNil())
	}
})

func warn(output string, err error) {
	if err != nil {
		fmt.Fprintf(GinkgoWriter, "warning: %s\n%s", err, output)
	}
}

func runPackageManifests(sample sample.Sample, args ...string) error {
	allArgs := []string{"run", "packagemanifests", "--timeout", "6m", "--namespace", kctl.Namespace()}
	output, err := sample.CommandContext().Run(exec.Command(sample.Binary(), append(allArgs, args...)...), sample.Name())
	if err == nil {
		fmt.Fprintln(GinkgoWriter, string(output))
	}
	return err
}

func cleanup(sample sample.Sample) (string, error) {
	allArgs := []string{"cleanup", sample.Name(), "--timeout", "4m", "--namespace", kctl.Namespace()}
	output, err := sample.CommandContext().Run(exec.Command(sample.Binary(), allArgs...), sample.Name())
	if err == nil {
		fmt.Fprintln(GinkgoWriter, string(output))
	}
	return string(output), err
}

func readCSV(sample sample.Sample, version string, isBundle bool) (*v1alpha1.ClusterServiceVersion, error) {
	b, err := ioutil.ReadFile(csvPath(sample, version, isBundle))
	if err != nil {
		return nil, err
	}
	csv := &v1alpha1.ClusterServiceVersion{}
	if err := yaml.Unmarshal(b, csv); err != nil {
		return nil, err
	}
	return csv, nil
}

func writeCSV(sample sample.Sample, version string, csv *v1alpha1.ClusterServiceVersion, isBundle bool) error {
	b, err := yaml.Marshal(csv)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(csvPath(sample, version, isBundle), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}
	if _, err := f.Write(b); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}
	return f.Close()
}

func csvPath(sample sample.Sample, version string, isBundle bool) string {
	fileName := fmt.Sprintf("%s.clusterserviceversion.yaml", sample.Name())
	if isBundle {
		return filepath.Join(sample.Dir(), "bundle", bundle.ManifestsDir, fileName)
	}
	return filepath.Join(sample.Dir(), "packagemanifests", version, fileName)
}
