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
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo"
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

	// TLS configuration.
	certDir      = flag.String("cert-dir", "", "registry TLS certificates and keys")
	caSecretName string

	localImageBase     string
	inClusterImageBase string
)

var _ = BeforeSuite(func() {
	flag.Parse()

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

	By("copying sample to a temporary e2e directory")
	Expect(exec.Command("cp", "-r", "../../testdata/go/v3/memcached-operator", tc.Dir).Run()).To(Succeed())

	By("creating the test namespace")
	_, err = tc.Kubectl.Command("create", "namespace", tc.Kubectl.Namespace)
	Expect(err).NotTo(HaveOccurred())

	By("configuring the local image registry")
	configureRegistry(tc)
	localImageBase = fmt.Sprintf("%s/integration/%s", localRegistryHost, tc.ProjectName)
	inClusterImageBase = fmt.Sprintf("%s/integration/%s", inClusterRegistryHost, tc.ProjectName)

	By("configuring registry TLS")
	certFile := filepath.Join(*certDir, "cert.pem")
	caSecretName = tc.ProjectName + "-ca-secret"
	_, err = tc.Kubectl.CommandInNamespace("create", "secret", "generic", caSecretName, "--from-file", "cert.pem="+certFile)
	Expect(err).NotTo(HaveOccurred())

	By("installing OLM")
	Expect(tc.InstallOLMVersion(testutils.OlmVersionForTestSuite)).To(Succeed())

	By("installing prometheus-operator")
	Expect(tc.InstallPrometheusOperManager()).To(Succeed())

	onKind, err = tc.IsRunningOnKind()
	Expect(err).NotTo(HaveOccurred())

	By("building the operator image")
	Expect(tc.Make("docker-build", "docker-push", "IMG="+localImageBase+":v0.0.1")).To(Succeed())
	if onKind {
		inClusterOperatorImage := inClusterImageBase + ":v0.0.1"
		Expect(dockerTag(tc, localImageBase+":v0.0.1", inClusterOperatorImage)).To(Succeed())
		Expect(tc.LoadImageToKindClusterWithName(inClusterOperatorImage)).To(Succeed())
	}

	setupBundle()
	setupPackagemanifests()
})

var _ = AfterSuite(func() {
	if tc == (testutils.TestContext{}) {
		return
	}

	By("uninstalling OLM")
	tc.UninstallOLM()

	By("uninstalling prometheus-operator")
	tc.UninstallPrometheusOperManager()

	By("deleting the test namespace")
	warn(tc.Kubectl.Delete(false, "namespace", tc.Kubectl.Namespace))

	By("cleaning up the project")
	tc.Destroy()
})

func setupPackagemanifests() {
	By("adding the packagemanifests target to the project Makefile")
	ExpectWithOffset(1, tc.AddPackagemanifestsTarget(projutil.OperatorTypeGo)).To(Succeed())

	By("generating operator packagemanifests and enabling all InstallModes")
	ExpectWithOffset(1, tc.Make("packagemanifests", "IMG="+inClusterImageBase+":v0.0.1")).To(Succeed())
	csv, err := readCSV(&tc, "0.0.1", false)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	for i := range csv.Spec.InstallModes {
		csv.Spec.InstallModes[i].Supported = true
	}
	ExpectWithOffset(1, writeCSV(&tc, "0.0.1", csv, false)).To(Succeed())
}

func dockerTag(tc testutils.TestContext, old, new string) error {
	cmd := exec.Command("docker", "tag", old, new)
	_, err := tc.Run(cmd)
	return err
}

func setupBundle() {
	By("generating the operator bundle and enabling all InstallModes")
	ExpectWithOffset(1, tc.Make("bundle", "IMG="+inClusterImageBase+":v0.0.1")).To(Succeed())
	csv, err := readCSV(&tc, "", true)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	for i := range csv.Spec.InstallModes {
		csv.Spec.InstallModes[i].Supported = true
	}
	ExpectWithOffset(1, writeCSV(&tc, "", csv, true)).To(Succeed())

	By("building the operator bundle image")
	ExpectWithOffset(1, tc.Make("bundle-build", "bundle-push", "BUNDLE_IMG="+localImageBase+"-bundle:v0.0.1")).To(Succeed())
	if onKind {
		inClusterBundleImage := inClusterImageBase + "-bundle:v0.0.1"
		ExpectWithOffset(1, dockerTag(tc, localImageBase+"-bundle:v0.0.1", inClusterBundleImage)).To(Succeed())
		ExpectWithOffset(1, tc.LoadImageToKindClusterWithName(inClusterBundleImage)).To(Succeed())
	}
}

func warn(output string, err error) {
	if err != nil {
		fmt.Fprintf(GinkgoWriter, "warning: %s\n%s", err, output)
	}
}

func runCmd(tc *testutils.TestContext, cmd string, args ...string) error {
	allArgs := []string{"run", cmd, "--timeout", "4m", "--namespace", tc.Kubectl.Namespace}
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
	b, err := ioutil.ReadFile(csvPath(tc, version, isBundle))
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
