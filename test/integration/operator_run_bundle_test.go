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

// Modified from https://github.com/kubernetes-sigs/kubebuilder/tree/39224f0/test/e2e/v3

package e2e

import (
	"context"
	"fmt"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/operator-framework/operator-sdk/internal/olm/operator"
	"github.com/operator-framework/operator-sdk/internal/olm/operator/bundle"
	testutils "github.com/operator-framework/operator-sdk/test/internal"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	wait "k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	kbtestutils "sigs.k8s.io/kubebuilder/test/e2e/utils"
)

// TODO: After run bundle is complete, remove the boolean variable and the if statements.
var runBundleImplemented = false

var (
	tc  testutils.TestContext
	err error
)

var _ = BeforeSuite(func() {
	By("creating a new context")
	// For this test suite, the bundle image name is equal to tc.ImageName
	tc, err = testutils.NewTestContext("GO111MODULE=on")
	Expect(err).NotTo(HaveOccurred())
	Expect(tc.Prepare()).To(Succeed())

	createOperator(tc, defaultOperatorName)
})

var _ = AfterSuite(func() {
	By("removing container image and working dir")
	tc.Destroy()
})

var _ = Describe("run bundle", func() {
	It("Run Bundle Basic", func() {
		if !runBundleImplemented {
			Skip("Run bundle not yet implemented")
		}

		cfg := &operator.Configuration{KubeconfigPath: kubeconfigPath}
		err := cfg.Load()
		Expect(err).NotTo(HaveOccurred())

		i := bundle.NewInstall(cfg)
		i.BundleImage = tc.ImageName

		err = doInstall(i)
		Expect(err).NotTo(HaveOccurred())

		uninstall(kubeconfigPath, tc)
	})

	It("Run bundle SingleNamespace", func() {
		if !runBundleImplemented {
			Skip("Run bundle not yet implemented")
		}

		cfg := &operator.Configuration{KubeconfigPath: kubeconfigPath, Namespace: tc.Kubectl.Namespace}
		err := cfg.Load()
		Expect(err).NotTo(HaveOccurred())

		i := bundle.NewInstall(cfg)
		i.BundleImage = tc.ImageName
		i.InstallMode.InstallModeType = v1alpha1.InstallModeTypeSingleNamespace
		i.InstallMode.TargetNamespaces = []string{tc.Kubectl.Namespace}

		err = doInstall(i)
		Expect(err).NotTo(HaveOccurred())

		uninstall(kubeconfigPath, tc)
	})

	It("Run bundle OwnNamespace", func() {
		if !runBundleImplemented {
			Skip("Run bundle not yet implemented")
		}

		cfg := &operator.Configuration{KubeconfigPath: kubeconfigPath}
		err := cfg.Load()
		Expect(err).NotTo(HaveOccurred())

		i := bundle.NewInstall(cfg)
		i.BundleImage = tc.ImageName
		i.InstallMode.InstallModeType = v1alpha1.InstallModeTypeOwnNamespace
		i.InstallMode.TargetNamespaces = []string{tc.Kubectl.Namespace}

		err = doInstall(i)
		Expect(err).NotTo(HaveOccurred())

		uninstall(kubeconfigPath, tc)
	})

	It("Run Bundle - watching the specified namesapce", func() {
		if !runBundleImplemented {
			Skip("Run bundle not yet implemented")
		}

		cfg := &operator.Configuration{KubeconfigPath: kubeconfigPath, Namespace: tc.Kubectl.Namespace}
		err := cfg.Load()
		Expect(err).NotTo(HaveOccurred())

		By("create a namespace to watch")
		_, err = tc.Kubectl.Command("create", "namespace", "bar")
		Expect(err).NotTo(HaveOccurred())

		i := bundle.NewInstall(cfg)
		i.BundleImage = tc.ImageName
		i.InstallMode.InstallModeType = v1alpha1.InstallModeTypeSingleNamespace
		i.InstallMode.TargetNamespaces = []string{tc.Kubectl.Namespace, "bar"}

		err = doInstall(i)
		Expect(err).NotTo(HaveOccurred())

		uninstall(kubeconfigPath, tc)
	})

})

// createOperator scaffolds a new test operator, generates bundle and pushes it to a remote
// repository. The generated csv by default supports OwnNamespace, SingleNamespace and AllNamespaces.
func createOperator(tc testutils.TestContext, projectName string) {
	By("initializing a project")
	err := tc.Init(
		"--project-version", "3-alpha",
		"--repo", path.Join("github.com", "example", projectName),
		"--domain", tc.Domain,
		"--fetch-deps=false")
	Expect(err).Should(Succeed())

	By("creating an API definition")
	err = tc.CreateAPI(
		"--group", tc.Group,
		"--version", tc.Version,
		"--kind", tc.Kind,
		"--namespaced",
		"--resource",
		"--controller",
		"--make=false")
	Expect(err).Should(Succeed())

	By("implementing the API")
	Expect(kbtestutils.InsertCode(
		filepath.Join(tc.Dir, "api", tc.Version, fmt.Sprintf("%s_types.go", strings.ToLower(tc.Kind))),
		fmt.Sprintf(`type %sSpec struct {
`, tc.Kind),
		`	// +optional
	Count int `+"`"+`json:"count,omitempty"`+"`"+`
`)).Should(Succeed())

	By("generating the operator bundle")
	// Turn off interactive prompts for all generation tasks.
	replace := "operator-sdk generate kustomize manifests"
	testutils.ReplaceInFile(filepath.Join(tc.Dir, "Makefile"), replace, replace+" --interactive=false")
	err = tc.Make("bundle", "IMG="+tc.ImageName)
	Expect(err).NotTo(HaveOccurred())

	By("building the operator bundle image")
	// Use the existing image tag but with a "-bundle" suffix.
	err = tc.Make("bundle-build", "BUNDLE_IMG="+tc.ImageName)
	Expect(err).NotTo(HaveOccurred())

	By("push the bundle to a remote repository")
	err = tc.Make("docker-push", "IMG="+tc.ImageName)
	Expect(err).NotTo(HaveOccurred())
}

func uninstall(kubeconfig string, tc testutils.TestContext) {
	cfg := &operator.Configuration{KubeconfigPath: kubeconfig}
	err := cfg.Load()
	Expect(err).NotTo(HaveOccurred())

	uninstallConfig := operator.NewUninstall(cfg)
	uninstallConfig.DeleteAll = true
	uninstallConfig.DeleteOperatorGroupNames = []string{tc.Group}
	uninstallConfig.Package = defaultOperatorName
	uninstallConfig.Logf = logrus.Infof

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	err = uninstallConfig.Run(ctx)
	Expect(err).NotTo(HaveOccurred())

	waitForConfigMapDeletion(ctx, cfg, defaultOperatorName)
}

func waitForConfigMapDeletion(ctx context.Context, cfg *operator.Configuration, packageName string) {
	cfgmaps := corev1.ConfigMapList{}
	opts := []client.ListOption{
		client.InNamespace(cfg.Namespace),
		client.MatchingLabels{"owner": "operator-sdk", "package-name": packageName},
	}
	err := wait.PollImmediateUntil(250*time.Millisecond, func() (bool, error) {
		if err := cfg.Client.List(ctx, &cfgmaps, opts...); err != nil {
			return false, err
		}
		return len(cfgmaps.Items) == 0, nil
	}, ctx.Done())
	Expect(err).NotTo(HaveOccurred())
}

func TestRunBundle(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Operator SDK run bundle test suite")
}
