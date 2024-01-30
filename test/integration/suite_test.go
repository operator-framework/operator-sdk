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
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/operator-framework/operator-registry/pkg/lib/bundle"
	"sigs.k8s.io/yaml"

	"github.com/operator-framework/operator-sdk/internal/testutils"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"
)

// TestIntegration tests operator-sdk projects with OLM.
func TestIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping Integration Suite in short mode")
	}
	RegisterFailHandler(Fail)
	RunSpecs(t, "Integration")
}

var (
	tc     testutils.TestContext
	onKind bool
)

var _ = BeforeSuite(func() {
	var err error

	By("creating a new test context")
	tc, err = testutils.NewTestContext(testutils.BinaryName, "GO111MODULE=on")
	Expect(err).NotTo(HaveOccurred())

	tc.Domain = "example.com"
	tc.Group = "cache"
	tc.Version = "v1alpha1"
	tc.Kind = "Memcached"
	tc.Resources = "memcacheds"
	tc.ProjectName = "memcached-operator"
	tc.ImageName = fmt.Sprintf("quay.io/integration/%s:0.0.1", tc.ProjectName)
	tc.BundleImageName = fmt.Sprintf("quay.io/integration/%s-bundle:0.0.1", tc.ProjectName)

	By("copying sample to a temporary e2e directory")
	Expect(exec.Command("cp", "-r", "../../testdata/go/v4/memcached-operator", tc.Dir).Run()).To(Succeed())

	By("updating the project configuration")
	Expect(tc.AddPackagemanifestsTarget(projutil.OperatorTypeGo)).To(Succeed())

	By("installing OLM")
	Expect(tc.InstallOLMVersion(testutils.OlmVersionForTestSuite)).To(Succeed())

	By("installing prometheus-operator")
	Expect(tc.InstallPrometheusOperManager()).To(Succeed())

	By("building the manager image")
	Expect(tc.Make("docker-build", "IMG="+tc.ImageName)).To(Succeed())

	onKind, err = tc.IsRunningOnKind()
	Expect(err).NotTo(HaveOccurred())
	if onKind {
		By("loading the required images into Kind cluster")
		Expect(tc.LoadImageToKindCluster()).To(Succeed())
	}

	By("generating the operator package manifests and enabling AllNamespaces InstallMode")
	Expect(tc.Make("packagemanifests", "IMG="+tc.ImageName)).To(Succeed())
	csv, err := readCSV(&tc, "0.0.1", false)
	Expect(err).NotTo(HaveOccurred())
	for i := range csv.Spec.InstallModes {
		if csv.Spec.InstallModes[i].Type == "AllNamespaces" {
			csv.Spec.InstallModes[i].Supported = true
		}
	}
	Expect(writeCSV(&tc, "0.0.1", csv, false)).To(Succeed())

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
	_, err = tc.Kubectl.Command("create", "namespace", tc.Kubectl.Namespace)
	Expect(err).NotTo(HaveOccurred())
})

var _ = AfterSuite(func() {
	By("uninstalling OLM")
	tc.UninstallOLM()

	By("uninstalling prometheus-operator")
	tc.UninstallPrometheusOperManager()

	By("deleting the test namespace")
	warn(tc.Kubectl.Delete(false, "namespace", tc.Kubectl.Namespace))

	By("cleaning up the project")
	tc.Destroy()
})

func warn(output string, err error) {
	if err != nil {
		fmt.Fprintf(GinkgoWriter, "warning: %s\n%s", err, output)
	}
}

func runPackageManifests(tc *testutils.TestContext, args ...string) error {
	allArgs := []string{"run", "packagemanifests", "--timeout", "6m", "--namespace", tc.Kubectl.Namespace}
	output, err := tc.Run(exec.Command(tc.BinaryName, append(allArgs, args...)...))
	if err == nil {
		fmt.Fprintln(GinkgoWriter, string(output))
	}
	return err
}

func cleanup(tc *testutils.TestContext) (string, error) {
	allArgs := []string{"cleanup", tc.ProjectName, "--timeout", "4m", "--namespace", tc.Kubectl.Namespace}
	output, err := tc.Run(exec.Command(tc.BinaryName, allArgs...))
	if err == nil {
		fmt.Fprintln(GinkgoWriter, string(output))
	}
	return string(output), err
}

func readCSV(tc *testutils.TestContext, version string, isBundle bool) (*v1alpha1.ClusterServiceVersion, error) {
	b, err := os.ReadFile(csvPath(tc, version, isBundle))
	if err != nil {
		return nil, err
	}
	csv := &v1alpha1.ClusterServiceVersion{}
	if err := yaml.Unmarshal(b, csv); err != nil {
		return nil, err
	}
	return csv, nil
}

func writeCSV(tc *testutils.TestContext, version string, csv *v1alpha1.ClusterServiceVersion, isBundle bool) error {
	b, err := yaml.Marshal(csv)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(csvPath(tc, version, isBundle), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
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

func csvPath(tc *testutils.TestContext, version string, isBundle bool) string {
	fileName := fmt.Sprintf("%s.clusterserviceversion.yaml", tc.ProjectName)
	if isBundle {
		return filepath.Join(tc.Dir, "bundle", bundle.ManifestsDir, fileName)
	}
	return filepath.Join(tc.Dir, "packagemanifests", version, fileName)
}
