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

package e2e_ansible_test

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/operator-framework/operator-sdk/internal/samples/ansible"
	"github.com/operator-framework/operator-sdk/internal/samples/pkg"
	"github.com/operator-framework/operator-sdk/internal/testutils"
)

// TestE2EAnsible ensures the ansible projects built with the SDK tool by using its binary.
func TestE2EAnsible(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping Operator SDK E2E Ansible Suite testing in short mode")
	}
	RegisterFailHandler(Fail)
	RunSpecs(t, "E2EAnsible Suite")
}

var (
	tc testutils.TestContext
)

// BeforeSuite run before any specs are run to perform the required actions for all e2e ansible tests.
var _ = BeforeSuite(func() {
	var err error

	By("creating a new test context")
	tc, err = testutils.NewTestContext(testutils.BinaryName, "GO111MODULE=on")
	Expect(err).NotTo(HaveOccurred())

	tc.Domain = "example.com"
	tc.Version = "v1alpha1"
	tc.Group = "cache"
	tc.Kind = "Memcached"
	tc.ProjectName = "memcached-operator"
	tc.Kubectl.Namespace = fmt.Sprintf("%s-system", tc.ProjectName)

	By("copying sample to a temporary e2e directory")
	Expect(exec.Command("cp", "-r", "../../testdata/ansible/memcached-operator", tc.Dir).Run()).To(Succeed())

	By("adding molecule sample steps")
	ctx, err := pkg.NewSampleContextWithTestContext(&tc)
	Expect(err).Should(Succeed())
	molecule := ansible.NewMoleculeAnsible(&ctx)
	molecule.Run()

	By("fetching the current-context")
	tc.Kubectx, err = tc.Kubectl.Command("config", "current-context")
	Expect(err).NotTo(HaveOccurred())

	By("preparing the prerequisites on cluster")
	tc.InstallPrerequisites()

	By("using dev image for scorecard-test")
	err = tc.ReplaceScorecardImagesForDev()
	Expect(err).NotTo(HaveOccurred())

	By("replacing project Dockerfile to use ansible base image with the dev tag")
	err = testutils.ReplaceRegexInFile(filepath.Join(tc.Dir, "Dockerfile"), "quay.io/operator-framework/ansible-operator:.*", "quay.io/operator-framework/ansible-operator:dev")
	Expect(err).Should(Succeed())

	By("building the project image")
	err = tc.Make("docker-build", "IMG="+tc.ImageName)
	Expect(err).NotTo(HaveOccurred())

	if tc.IsRunningOnKind() {
		By("loading the required images into Kind cluster")
		Expect(tc.LoadImageToKindCluster()).To(Succeed())
		Expect(tc.LoadImageToKindClusterWithName("quay.io/operator-framework/scorecard-test:dev")).To(Succeed())
	}

	By("running OLM integration steps")
	err = tc.RunOlmIntegration()
	Expect(err).NotTo(HaveOccurred())
})

// AfterSuite run after all the specs have run, regardless of whether any tests have failed to ensures that
// all be cleaned up
var _ = AfterSuite(func() {
	By("uninstalling prerequisites")
	tc.UninstallPrerequisites()

	By("destroying container image and work dir")
	tc.Destroy()
})
