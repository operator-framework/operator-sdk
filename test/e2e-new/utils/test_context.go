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

package utils

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo" //nolint:golint
)

// TODO: remove this file once kubernetes-sigs/kubebuilder#1520 is merged

// TestContext specified to run e2e tests
type TestContext struct {
	*CmdContext
	TestSuffix string
	Domain     string
	Group      string
	Version    string
	Kind       string
	Resources  string
	ImageName  string
	BinName    string
	Kubectl    *Kubectl
}

// NewTestContext init with a random suffix for test TestContext stuff,
// to avoid conflict when running tests synchronously.
func NewTestContext(binName string, env ...string) (*TestContext, error) {
	testSuffix, err := randomSuffix()
	if err != nil {
		return nil, err
	}

	testGroup := "bar" + testSuffix
	path, err := filepath.Abs("e2e-" + testSuffix)
	if err != nil {
		return nil, err
	}

	cc := &CmdContext{
		Env: env,
		Dir: path,
	}

	return &TestContext{
		TestSuffix: testSuffix,
		Domain:     "example.com" + testSuffix,
		Group:      testGroup,
		Version:    "v1alpha1",
		Kind:       "Foo" + testSuffix,
		Resources:  "foo" + testSuffix + "s",
		ImageName:  "e2e-test/controller-manager:" + testSuffix,
		BinName:    binName,
		CmdContext: cc,
		Kubectl: &Kubectl{
			Namespace:  fmt.Sprintf("e2e-%s-system", testSuffix),
			CmdContext: cc,
		},
	}, nil
}

// Prepare prepare a work directory for testing
func (tc *TestContext) Prepare() error {
	fmt.Fprintf(GinkgoWriter, "preparing testing directory: %s\n", tc.Dir)
	return os.MkdirAll(tc.Dir, 0755)
}

// CleanupManifests is a helper func to run kustomize build and pipe the output to kubectl delete -f -
func (tc *TestContext) CleanupManifests(dir string) {
	cmd := exec.Command("kustomize", "build", dir)
	output, err := tc.Run(cmd)
	if err != nil {
		fmt.Fprintf(GinkgoWriter, "warning: error when running kustomize build: %v\n", err)
	}
	if _, err := tc.Kubectl.WithInput(string(output)).Command("delete", "-f", "-"); err != nil {
		fmt.Fprintf(GinkgoWriter, "warning: error when running kubectl delete -f -: %v\n", err)
	}
}

// Init is for running `<tc.BinName> init`
func (tc *TestContext) Init(initOptions ...string) error {
	initOptions = append([]string{"init"}, initOptions...)
	cmd := exec.Command(tc.BinName, initOptions...)
	_, err := tc.Run(cmd)
	return err
}

// CreateAPI is for running `<tc.BinName> create api`
func (tc *TestContext) CreateAPI(resourceOptions ...string) error {
	resourceOptions = append([]string{"create", "api"}, resourceOptions...)
	cmd := exec.Command(tc.BinName, resourceOptions...)
	_, err := tc.Run(cmd)
	return err
}

// CreateWebhook is for running `<tc.BinName> create webhook`
func (tc *TestContext) CreateWebhook(resourceOptions ...string) error {
	resourceOptions = append([]string{"create", "webhook"}, resourceOptions...)
	cmd := exec.Command(tc.BinName, resourceOptions...)
	_, err := tc.Run(cmd)
	return err
}

// Make is for running `make` with various targets
func (tc *TestContext) Make(makeOptions ...string) error {
	cmd := exec.Command("make", makeOptions...)
	_, err := tc.Run(cmd)
	return err
}

// Destroy is for cleaning up the docker images for testing
func (tc *TestContext) Destroy() {
	//nolint:gosec
	cmd := exec.Command("docker", "rmi", "-f", tc.ImageName)
	if _, err := tc.Run(cmd); err != nil {
		fmt.Fprintf(GinkgoWriter, "warning: error when removing the local image: %v\n", err)
	}
	if err := os.RemoveAll(tc.Dir); err != nil {
		fmt.Fprintf(GinkgoWriter, "warning: error when removing the word dir: %v\n", err)
	}
}

// LoadImageToKindCluster loads a local docker image to the kind cluster
func (tc *TestContext) LoadImageToKindCluster() error {
	kindOptions := []string{"load", "docker-image", tc.ImageName}
	cmd := exec.Command("kind", kindOptions...)
	_, err := tc.Run(cmd)
	return err
}

// CmdContext provides context for command execution
type CmdContext struct {
	// environment variables in k=v format.
	Env   []string
	Dir   string
	Stdin io.Reader
}

// Run executes the provided command within this context
func (cc *CmdContext) Run(cmd *exec.Cmd) ([]byte, error) {
	cmd.Dir = cc.Dir
	cmd.Env = append(os.Environ(), cc.Env...)
	cmd.Stdin = cc.Stdin
	command := strings.Join(cmd.Args, " ")
	fmt.Fprintf(GinkgoWriter, "running: %s\n", command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return output, fmt.Errorf("%s failed with error: %s", command, string(output))
	}

	return output, nil
}
