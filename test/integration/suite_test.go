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
	"strconv"

	"github.com/phayes/freeport"

	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/operator-framework/operator-registry/pkg/lib/bundle"
	"sigs.k8s.io/kubebuilder/v3/test/e2e/utils"
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
	tc testutils.TestContext
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
	Expect(exec.Command("cp", "-r", "../../testdata/go/v3/memcached-operator", tc.Dir).Run()).To(Succeed())

	// By("updating the project configuration")
	// updateProjectConfigs(tc)

	By("installing OLM")
	Expect(tc.InstallOLMVersion("0.15.1")).To(Succeed())

	By("installing prometheus-operator")
	Expect(tc.InstallPrometheusOperManager()).To(Succeed())

	By("building the manager image")
	Expect(tc.Make("docker-build", "IMG="+tc.ImageName)).To(Succeed())

	if tc.IsRunningOnKind() {
		By("loading the required images into Kind cluster")
		Expect(tc.LoadImageToKindCluster()).To(Succeed())
	}

	// By("generating the operator package manifests")
	// err = tc.Make("packagemanifests", "IMG="+tc.ImageName)
	// Expect(err).NotTo(HaveOccurred())

	fmt.Printf("----------------------%+v", tc.ImageName)
	// TODO(estroz): enable when bundles can be tested locally.

	os.Setenv("LOCAL_IMAGE_REGISTRY", "1")
	PortNumber, _ := freeport.GetFreePort()
	os.Setenv("PORT_NUMBER", strconv.Itoa(PortNumber))

	ImgName := fmt.Sprintf("localhost:%d/%s:0.0.1", PortNumber, tc.ProjectName)
	BundleImgName := fmt.Sprintf("localhost:%d/%s-bundle:0.0.1", PortNumber, tc.ProjectName)

	if tc.IsRunningOnKind() {
		By("loading the required images into Kind cluster")
		Expect(tc.LoadImageToKindCluster()).To(Succeed())
	}

	cmd := exec.Command("kind", "load", "docker-image", ImgName, "--name", "Kind")
	_, err = cmd.Output()

	os.Setenv("BUNDLE_IMAGE_NAME", BundleImgName)

	fmt.Printf("-----------------hello %+v\n", PortNumber)

	cmd2 := exec.Command("docker", "run", "-d", "-p", strconv.Itoa(PortNumber)+":"+strconv.Itoa(5000), "--restart=always", "--name", "registry", "registry:2")

	stdout, _ := cmd2.Output()

	// localRegistryCmd := exec.Command("docker", "run", "-d", "-p", strconv.Itoa(PortNumber)+":"+strconv.Itoa(5000), "--restart=always", "--name", "registry", "registry:2")

	// stdout, _ := localRegistryCmd.Output()
	// if err != nil {
	// 	fmt.Println(err.Error())
	// 	return
	// }

	fmt.Printf("-----------------hello %+v\n", stdout)

	By("generating the operator bundle")
	err = tc.Make("bundle", "IMG="+ImgName)
	Expect(err).NotTo(HaveOccurred())

	By("building the operator bundle image")
	err = tc.Make("bundle-build", "BUNDLE_IMG="+BundleImgName)
	Expect(err).NotTo(HaveOccurred())

	By("push the image to a local registry")
	err = tc.Make("docker-build", "IMG="+ImgName)
	Expect(err).NotTo(HaveOccurred())

	os.Setenv("OPERATOR_IMAGE_NAME", ImgName)

	By("creating the test namespace")
	_, err = tc.Kubectl.Command("create", "namespace", tc.Kubectl.Namespace)
	Expect(err).NotTo(HaveOccurred())

	os.Setenv("KUBECTL_NAMESPACE", tc.Kubectl.Namespace)

})

var _ = AfterSuite(func() {
	By("uninstalling OLM")
	tc.UninstallOLM()

	// By("uninstalling prometheus-operator")
	// tc.UninstallPrometheusOperManager()

	// By("deleting the test namespace")
	// warn(tc.Kubectl.Delete(false, "namespace", tc.Kubectl.Namespace))

	// By("cleaning up the project")
	// tc.Destroy()
})

func warn(output string, err error) {
	if err != nil {
		fmt.Fprintf(GinkgoWriter, "warning: %s", err)
	}
}

func runPackageManifestsFor(tc *testutils.TestContext) func(...string) error {
	return func(args ...string) error {
		allArgs := []string{"run", "packagemanifests", "--timeout", "4m", "--namespace", tc.Kubectl.Namespace}
		output, err := tc.Run(exec.Command(tc.BinaryName, append(allArgs, args...)...))
		if err == nil {
			fmt.Fprintln(GinkgoWriter, string(output))
		}
		return err
	}
}

func cleanupFor(tc *testutils.TestContext) func() (string, error) {
	return func() (string, error) {
		allArgs := []string{"cleanup", tc.ProjectName, "--timeout", "4m", "--namespace", tc.Kubectl.Namespace}
		output, err := tc.Run(exec.Command(tc.BinaryName, allArgs...))
		if err == nil {
			fmt.Fprintln(GinkgoWriter, string(output))
		}
		return string(output), err
	}
}

func readCSVFor(tc *testutils.TestContext, isBundle bool) func(string) (*v1alpha1.ClusterServiceVersion, error) {
	return func(version string) (*v1alpha1.ClusterServiceVersion, error) {
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
}

func writeCSVFor(tc *testutils.TestContext, isBundle bool) func(*v1alpha1.ClusterServiceVersion) error {
	return func(csv *v1alpha1.ClusterServiceVersion) error {
		b, err := yaml.Marshal(csv)
		if err != nil {
			return err
		}
		return ioutil.WriteFile(csvPath(tc, csv.Spec.Version.String(), isBundle), b, 0644)
	}
}

func csvPath(tc *testutils.TestContext, version string, isBundle bool) string {
	fileName := fmt.Sprintf("%s.clusterserviceversion.yaml", tc.ProjectName)
	if isBundle {
		return filepath.Join(tc.Dir, "bundle", bundle.ManifestsDir, fileName)
	}
	return filepath.Join(tc.Dir, "packagemanifests", version, fileName)
}

func updateProjectConfigs(tc testutils.TestContext) {
	defaultKustomization := filepath.Join(tc.Dir, "config", "default", "kustomization.yaml")
	ExpectWithOffset(1, testutils.ReplaceInFile(defaultKustomization,
		"- ../certmanager",
		"#- ../certmanager",
	)).To(Succeed())
	ExpectWithOffset(1, testutils.ReplaceInFile(defaultKustomization,
		"- manager_webhook_patch.yaml",
		"#- manager_webhook_patch.yaml",
	)).To(Succeed())

	olmManagerWebhookPatchFile := filepath.Join(tc.Dir, "config", "manifests", "olm_manager_webhook_patch.yaml")
	ExpectWithOffset(1, ioutil.WriteFile(olmManagerWebhookPatchFile, []byte(olmManagerWebhookPatch), 0644)).To(Succeed())

	ExpectWithOffset(1, utils.InsertCode(filepath.Join(tc.Dir, "config", "manifests", "kustomization.yaml"),
		"- ../scorecard",
		defaultKustomizationOLMWebhookPatch,
	)).To(Succeed())

	ExpectWithOffset(1, tc.AddPackagemanifestsTarget(projutil.OperatorTypeGo)).To(Succeed())
}

// Exposes port 9443 for the OLM-managed webhook server.
const defaultKustomizationOLMWebhookPatch = `
patchesStrategicMerge:
- olm_manager_webhook_patch.yaml
`

const olmManagerWebhookPatch = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: controller-manager
  namespace: system
spec:
  template:
    spec:
      containers:
      - name: manager
        ports:
        - containerPort: 9443
          name: webhook-server
          protocol: TCP
`
