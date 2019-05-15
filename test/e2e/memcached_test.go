// Copyright 2018 The Operator-SDK Authors
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

package e2e

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/operator-framework/operator-sdk/internal/pkg/scaffold"
	"github.com/operator-framework/operator-sdk/internal/util/fileutil"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"
	"github.com/operator-framework/operator-sdk/internal/util/yamlutil"
	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"

	"github.com/ghodss/yaml"
	"github.com/prometheus/prometheus/util/promlint"
	"github.com/rogpeppe/go-internal/modfile"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	crYAML               string = "apiVersion: \"cache.example.com/v1alpha1\"\nkind: \"Memcached\"\nmetadata:\n  name: \"example-memcached\"\nspec:\n  size: 3"
	retryInterval               = time.Second * 5
	timeout                     = time.Second * 120
	cleanupRetryInterval        = time.Second * 1
	cleanupTimeout              = time.Second * 10
	operatorName                = "memcached-operator"
)

func TestMemcached(t *testing.T) {
	// get global framework variables
	ctx := framework.NewTestCtx(t)
	defer ctx.Cleanup()
	gopath, ok := os.LookupEnv(projutil.GoPathEnv)
	if !ok || gopath == "" {
		t.Fatalf("$GOPATH not set")
	}
	cd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(cd); err != nil {
			t.Errorf("Failed to change back to original working directory: (%v)", err)
		}
	}()
	// For go commands in operator projects.
	if err = os.Setenv("GO111MODULE", "on"); err != nil {
		t.Fatal(err)
	}

	// Setup
	absProjectPath, err := ioutil.TempDir(filepath.Join(gopath, "src"), "tmp.")
	if err != nil {
		t.Fatal(err)
	}
	ctx.AddCleanupFn(func() error { return os.RemoveAll(absProjectPath) })

	if err := os.MkdirAll(absProjectPath, fileutil.DefaultDirFileMode); err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(absProjectPath); err != nil {
		t.Fatal(err)
	}

	t.Log("Creating new operator project")
	cmdOut, err := exec.Command("operator-sdk",
		"new",
		operatorName).CombinedOutput()
	if err != nil {
		t.Fatalf("Error: %v\nCommand Output: %s\n", err, string(cmdOut))
	}

	if err := os.Chdir(operatorName); err != nil {
		t.Fatalf("Failed to change to %s directory: (%v)", operatorName, err)
	}

	sdkRepo := "github.com/operator-framework/operator-sdk"
	localSDKPath := filepath.Join(gopath, "src", sdkRepo)

	replace := getGoModReplace(t, localSDKPath)
	if replace.repo != sdkRepo {
		if replace.isLocal {
			// A hacky way to get local module substitution to work is to write a
			// stub go.mod into the local SDK repo referred to in
			// memcached-operator's go.mod, which allows go to recognize
			// the local SDK repo as a module.
			sdkModPath := filepath.Join(replace.repo, "go.mod")
			err = ioutil.WriteFile(sdkModPath, []byte("module "+sdkRepo), fileutil.DefaultFileMode)
			if err != nil {
				t.Fatalf("Failed to write main repo go.mod file: %v", err)
			}
			defer func() {
				if err = os.RemoveAll(sdkModPath); err != nil {
					t.Fatalf("Failed to remove %s: %v", sdkModPath, err)
				}
			}()
		}
		writeGoModReplace(t, sdkRepo, replace.repo, replace.ref)
	}

	cmdOut, err = exec.Command("go", "mod", "vendor").CombinedOutput()
	if err != nil {
		t.Fatalf("Error after modifying go.mod: %v\nCommand Output: %s\n", err, string(cmdOut))
	}

	// Set replicas to 2 to test leader election. In production, this should
	// almost always be set to 1, because there isn't generally value in having
	// a hot spare operator process.
	opYaml, err := ioutil.ReadFile("deploy/operator.yaml")
	if err != nil {
		t.Fatalf("Could not read deploy/operator.yaml: %v", err)
	}
	newOpYaml := bytes.Replace(opYaml, []byte("replicas: 1"), []byte("replicas: 2"), 1)
	err = ioutil.WriteFile("deploy/operator.yaml", newOpYaml, 0644)
	if err != nil {
		t.Fatalf("Could not write deploy/operator.yaml: %v", err)
	}

	cmd := exec.Command("operator-sdk",
		"add",
		"api",
		"--api-version=cache.example.com/v1alpha1",
		"--kind=Memcached")
	cmd.Env = os.Environ()
	cmdOut, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Error: %v\nCommand Output: %s\n", err, string(cmdOut))
	}
	cmdOut, err = exec.Command("operator-sdk",
		"add",
		"controller",
		"--api-version=cache.example.com/v1alpha1",
		"--kind=Memcached").CombinedOutput()
	if err != nil {
		t.Fatalf("Error: %v\nCommand Output: %s\n", err, string(cmdOut))
	}

	tmplFiles := map[string]string{
		filepath.Join(localSDKPath, "example/memcached-operator/memcached_controller.go.tmpl"): "pkg/controller/memcached/memcached_controller.go",
		filepath.Join(localSDKPath, "test/e2e/incluster-test-code/main_test.go.tmpl"):          "test/e2e/main_test.go",
		filepath.Join(localSDKPath, "test/e2e/incluster-test-code/memcached_test.go.tmpl"):     "test/e2e/memcached_test.go",
	}
	for src, dst := range tmplFiles {
		if err := os.MkdirAll(filepath.Dir(dst), fileutil.DefaultDirFileMode); err != nil {
			t.Fatalf("Could not create template destination directory: %s", err)
		}
		srcTmpl, err := ioutil.ReadFile(src)
		if err != nil {
			t.Fatalf("Could not read template from %s: %s", src, err)
		}
		dstData := strings.Replace(string(srcTmpl), "github.com/example-inc", filepath.Base(absProjectPath), -1)
		if err := ioutil.WriteFile(dst, []byte(dstData), fileutil.DefaultFileMode); err != nil {
			t.Fatalf("Could not write template output to %s: %s", dst, err)
		}
	}

	memcachedTypesFile, err := ioutil.ReadFile("pkg/apis/cache/v1alpha1/memcached_types.go")
	if err != nil {
		t.Fatal(err)
	}
	memcachedTypesFileLines := bytes.Split(memcachedTypesFile, []byte("\n"))
	for lineNum, line := range memcachedTypesFileLines {
		if strings.Contains(string(line), "type MemcachedSpec struct {") {
			memcachedTypesFileLinesIntermediate := append(memcachedTypesFileLines[:lineNum+1], []byte("\tSize int32 `json:\"size\"`"))
			memcachedTypesFileLines = append(memcachedTypesFileLinesIntermediate, memcachedTypesFileLines[lineNum+3:]...)
			break
		}
	}
	for lineNum, line := range memcachedTypesFileLines {
		if strings.Contains(string(line), "type MemcachedStatus struct {") {
			memcachedTypesFileLinesIntermediate := append(memcachedTypesFileLines[:lineNum+1], []byte("\tNodes []string `json:\"nodes\"`"))
			memcachedTypesFileLines = append(memcachedTypesFileLinesIntermediate, memcachedTypesFileLines[lineNum+3:]...)
			break
		}
	}
	if err := os.Remove("pkg/apis/cache/v1alpha1/memcached_types.go"); err != nil {
		t.Fatalf("Failed to remove old memcached_type.go file: (%v)", err)
	}
	err = ioutil.WriteFile("pkg/apis/cache/v1alpha1/memcached_types.go", bytes.Join(memcachedTypesFileLines, []byte("\n")), fileutil.DefaultFileMode)
	if err != nil {
		t.Fatal(err)
	}

	t.Log("Generating k8s")
	cmdOut, err = exec.Command("operator-sdk", "generate", "k8s").CombinedOutput()
	if err != nil {
		t.Fatalf("Error: %v\nCommand Output: %s\n", err, string(cmdOut))
	}

	t.Log("Pulling new dependencies with go mod")
	cmdOut, err = exec.Command("go", "mod", "vendor").CombinedOutput()
	if err != nil {
		t.Fatalf("Command 'go mod vendor' failed: %v\nCommand Output:\n%v", err, string(cmdOut))
	}

	file, err := yamlutil.GenerateCombinedGlobalManifest(scaffold.CRDsDir)
	if err != nil {
		t.Fatal(err)
	}
	ctx.AddCleanupFn(func() error { return os.Remove(file.Name()) })

	// hacky way to use createFromYAML without exposing the method
	// create crd
	filename := file.Name()
	framework.Global.NamespacedManPath = &filename
	err = ctx.InitializeClusterResources(&framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		t.Fatal(err)
	}
	t.Log("Created global resources")

	// run subtests
	t.Run("memcached-group", func(t *testing.T) {
		t.Run("Cluster", MemcachedCluster)
		t.Run("Local", MemcachedLocal)
	})
}

type goModReplace struct {
	repo    string
	ref     string
	isLocal bool
}

// getGoModReplace returns a go.mod replacement that is appropriate based on the build's
// environment to support PR, fork/branch, and local builds.
//
//   PR:
//     1. Activate when TRAVIS_PULL_REQUEST_SLUG and TRAVIS_PULL_REQUEST_SHA are set
//     2. Modify go.mod to replace osdk import with github.com/${TRAVIS_PULL_REQUEST_SLUG} ${TRAVIS_PULL_REQUEST_SHA}
//
//   Fork/branch:
//     1. Activate when TRAVIS_REPO_SLUG and TRAVIS_COMMIT are set
//     2. Modify go.mod to replace osdk import with github.com/${TRAVIS_REPO_SLUG} ${TRAVIS_COMMIT}
//
//   Local:
//     1. Activate when none of the above TRAVIS_* variables are set.
//     2. Modify go.mod to replace osdk import with local filesystem path.
//
func getGoModReplace(t *testing.T, localSDKPath string) goModReplace {
	// PR environment
	prSlug, prSlugOk := os.LookupEnv("TRAVIS_PULL_REQUEST_SLUG")
	prSha, prShaOk := os.LookupEnv("TRAVIS_PULL_REQUEST_SHA")
	if prSlugOk && prSlug != "" && prShaOk && prSha != "" {
		return goModReplace{
			repo: fmt.Sprintf("github.com/%s", prSlug),
			ref:  prSha,
		}
	}

	// Fork/branch environment
	slug, slugOk := os.LookupEnv("TRAVIS_REPO_SLUG")
	sha, shaOk := os.LookupEnv("TRAVIS_COMMIT")
	if slugOk && slug != "" && shaOk && sha != "" {
		return goModReplace{
			repo: fmt.Sprintf("github.com/%s", slug),
			ref:  sha,
		}
	}

	// If neither of the above cases is applicable, but one of the TRAVIS_*
	// variables is nonetheless set, something unexpected is going on. Log
	// the vars and exit.
	if prSlugOk || prShaOk || slugOk || shaOk {
		t.Logf("TRAVIS_PULL_REQUEST_SLUG='%s', set: %t", prSlug, prSlugOk)
		t.Logf("TRAVIS_PULL_REQUEST_SHA='%s', set: %t", prSha, prShaOk)
		t.Logf("TRAVIS_REPO_SLUG='%s', set: %t", slug, slugOk)
		t.Logf("TRAVIS_COMMIT='%s', set: %t", sha, shaOk)
		t.Fatal("Invalid travis environment")
	}

	// Local environment
	return goModReplace{
		repo:    localSDKPath,
		isLocal: true,
	}
}

func writeGoModReplace(t *testing.T, repo, path, sha string) {
	modBytes, err := ioutil.ReadFile("go.mod")
	if err != nil {
		t.Fatalf("Failed to read go.mod: %v", err)
	}
	modFile, err := modfile.Parse("go.mod", modBytes, nil)
	if err != nil {
		t.Fatalf("Failed to parse go.mod: %v", err)
	}
	if err = modFile.AddReplace(repo, "", path, sha); err != nil {
		s := ""
		if sha != "" {
			s = " " + sha
		}
		t.Fatalf(`Failed to add "replace %s => %s%s: %v"`, repo, path, s, err)
	}
	if modBytes, err = modFile.Format(); err != nil {
		t.Fatalf("Failed to format go.mod: %v", err)
	}
	err = ioutil.WriteFile("go.mod", modBytes, fileutil.DefaultFileMode)
	if err != nil {
		t.Fatalf("Failed to write updated go.mod: %v", err)
	}
	t.Logf("go.mod: %v", string(modBytes))
}

func memcachedLeaderTest(t *testing.T, f *framework.Framework, ctx *framework.TestCtx) error {
	namespace, err := ctx.GetNamespace()
	if err != nil {
		return err
	}

	err = e2eutil.WaitForOperatorDeployment(t, f.KubeClient, namespace, operatorName, 2, retryInterval, timeout)
	if err != nil {
		return err
	}

	label := map[string]string{"name": operatorName}

	leader, err := verifyLeader(t, namespace, f, label)
	if err != nil {
		return err
	}

	// delete the leader's pod so a new leader will get elected
	err = f.Client.Delete(context.TODO(), leader)
	if err != nil {
		return err
	}

	err = e2eutil.WaitForDeletion(t, f.Client.Client, leader, retryInterval, timeout)
	if err != nil {
		return err
	}

	err = e2eutil.WaitForOperatorDeployment(t, f.KubeClient, namespace, operatorName, 2, retryInterval, timeout)
	if err != nil {
		return err
	}

	newLeader, err := verifyLeader(t, namespace, f, label)
	if err != nil {
		return err
	}
	if newLeader.Name == leader.Name {
		return fmt.Errorf("leader pod name did not change across pod delete")
	}

	return nil
}

func verifyLeader(t *testing.T, namespace string, f *framework.Framework, labels map[string]string) (*v1.Pod, error) {
	// get configmap, which is the lock
	lockName := "memcached-operator-lock"
	lock := v1.ConfigMap{}
	err := wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		err = f.Client.Get(context.TODO(), types.NamespacedName{Name: lockName, Namespace: namespace}, &lock)
		if err != nil {
			if apierrors.IsNotFound(err) {
				t.Logf("Waiting for availability of leader lock configmap %s\n", lockName)
				return false, nil
			}
			return false, err
		}
		return true, nil
	})
	if err != nil {
		return nil, fmt.Errorf("error getting leader lock configmap: %v\n", err)
	}
	t.Logf("Found leader lock configmap %s\n", lockName)

	owners := lock.GetOwnerReferences()
	if len(owners) != 1 {
		return nil, fmt.Errorf("leader lock has %d owner refs, expected 1", len(owners))
	}
	owner := owners[0]

	// get operator pods
	pods := v1.PodList{}
	opts := client.ListOptions{Namespace: namespace}
	for k, v := range labels {
		if err := opts.SetLabelSelector(fmt.Sprintf("%s=%s", k, v)); err != nil {
			return nil, fmt.Errorf("failed to set list label selector: (%v)", err)
		}
	}
	if err := opts.SetFieldSelector("status.phase=Running"); err != nil {
		t.Fatalf("Failed to set list field selector: (%v)", err)
	}
	err = f.Client.List(context.TODO(), &opts, &pods)
	if err != nil {
		return nil, err
	}
	if len(pods.Items) != 2 {
		return nil, fmt.Errorf("expected 2 pods, found %d", len(pods.Items))
	}

	// find and return the leader
	for _, pod := range pods.Items {
		if pod.Name == owner.Name {
			return &pod, nil
		}
	}
	return nil, fmt.Errorf("did not find operator pod that was referenced by configmap")
}

func memcachedScaleTest(t *testing.T, f *framework.Framework, ctx *framework.TestCtx) error {
	// create example-memcached yaml file
	filename := "deploy/cr.yaml"
	err := ioutil.WriteFile(filename,
		[]byte(crYAML),
		fileutil.DefaultFileMode)
	if err != nil {
		return err
	}

	// create memcached custom resource
	framework.Global.NamespacedManPath = &filename
	err = ctx.InitializeClusterResources(&framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		return err
	}
	t.Log("Created cr")

	namespace, err := ctx.GetNamespace()
	if err != nil {
		return err
	}
	// wait for example-memcached to reach 3 replicas
	err = e2eutil.WaitForDeployment(t, f.KubeClient, namespace, "example-memcached", 3, retryInterval, timeout)
	if err != nil {
		return err
	}

	// get fresh copy of memcached object as unstructured
	obj := unstructured.Unstructured{}
	jsonSpec, err := yaml.YAMLToJSON([]byte(crYAML))
	if err != nil {
		return fmt.Errorf("could not convert yaml file to json: %v", err)
	}
	if err := obj.UnmarshalJSON(jsonSpec); err != nil {
		t.Fatalf("Failed to unmarshal memcached CR: (%v)", err)
	}
	obj.SetNamespace(namespace)
	err = f.Client.Get(context.TODO(), types.NamespacedName{Name: obj.GetName(), Namespace: obj.GetNamespace()}, &obj)
	if err != nil {
		return fmt.Errorf("failed to get memcached object: %s", err)
	}
	// update memcached CR size to 4
	spec, ok := obj.Object["spec"].(map[string]interface{})
	if !ok {
		return errors.New("memcached object missing spec field")
	}
	spec["size"] = 4
	err = f.Client.Update(context.TODO(), &obj)
	if err != nil {
		return err
	}

	// wait for example-memcached to reach 4 replicas
	return e2eutil.WaitForDeployment(t, f.KubeClient, namespace, "example-memcached", 4, retryInterval, timeout)
}

func MemcachedLocal(t *testing.T) {
	// get global framework variables
	ctx := framework.NewTestCtx(t)
	defer ctx.Cleanup()
	namespace, err := ctx.GetNamespace()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("operator-sdk", "up", "local", "--namespace="+namespace)
	stderr, err := os.Create("stderr.txt")
	if err != nil {
		t.Fatalf("Failed to create stderr.txt: %v", err)
	}
	cmd.Stderr = stderr
	defer func() {
		if err := stderr.Close(); err != nil && !fileutil.IsClosedError(err) {
			t.Errorf("Failed to close stderr: (%v)", err)
		}
	}()

	err = cmd.Start()
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	ctx.AddCleanupFn(func() error { return cmd.Process.Signal(os.Interrupt) })

	// wait for operator to start (may take a minute to compile the command...)
	err = wait.Poll(time.Second*5, time.Second*100, func() (done bool, err error) {
		file, err := ioutil.ReadFile("stderr.txt")
		if err != nil {
			return false, err
		}
		if len(file) == 0 {
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		t.Fatalf("Local operator not ready after 100 seconds: %v\n", err)
	}

	if err = memcachedScaleTest(t, framework.Global, ctx); err != nil {
		t.Fatal(err)
	}
}

func MemcachedCluster(t *testing.T) {
	// get global framework variables
	ctx := framework.NewTestCtx(t)
	defer ctx.Cleanup()
	operatorYAML, err := ioutil.ReadFile("deploy/operator.yaml")
	if err != nil {
		t.Fatalf("Could not read deploy/operator.yaml: %v", err)
	}
	local := *e2eImageName == ""
	if local {
		*e2eImageName = "quay.io/example/memcached-operator:v0.0.1"
		if err != nil {
			t.Fatal(err)
		}
		operatorYAML = bytes.Replace(operatorYAML, []byte("imagePullPolicy: Always"), []byte("imagePullPolicy: Never"), 1)
		err = ioutil.WriteFile("deploy/operator.yaml", operatorYAML, fileutil.DefaultFileMode)
		if err != nil {
			t.Fatal(err)
		}
	}
	operatorYAML = bytes.Replace(operatorYAML, []byte("REPLACE_IMAGE"), []byte(*e2eImageName), 1)
	err = ioutil.WriteFile("deploy/operator.yaml", operatorYAML, os.FileMode(0644))
	if err != nil {
		t.Fatalf("Failed to write deploy/operator.yaml: %v", err)
	}
	t.Log("Building operator docker image")
	cmdOut, err := exec.Command("operator-sdk", "build", *e2eImageName).CombinedOutput()
	if err != nil {
		t.Fatalf("Error: %v\nCommand Output: %s\n", err, string(cmdOut))
	}

	if !local {
		t.Log("Pushing docker image to repo")
		cmdOut, err = exec.Command("docker", "push", *e2eImageName).CombinedOutput()
		if err != nil {
			t.Fatalf("Error: %v\nCommand Output: %s\n", err, string(cmdOut))
		}
	}

	file, err := yamlutil.GenerateCombinedNamespacedManifest(scaffold.DeployDir)
	if err != nil {
		t.Fatal(err)
	}
	ctx.AddCleanupFn(func() error { return os.Remove(file.Name()) })

	// create namespaced resources
	filename := file.Name()
	framework.Global.NamespacedManPath = &filename
	err = ctx.InitializeClusterResources(&framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		t.Fatal(err)
	}
	t.Log("Created namespaced resources")

	namespace, err := ctx.GetNamespace()
	if err != nil {
		t.Fatal(err)
	}
	// wait for memcached-operator to be ready
	err = e2eutil.WaitForOperatorDeployment(t, framework.Global.KubeClient, namespace, operatorName, 2, retryInterval, timeout)
	if err != nil {
		t.Fatal(err)
	}

	if err = memcachedLeaderTest(t, framework.Global, ctx); err != nil {
		t.Fatal(err)
	}

	if err = memcachedScaleTest(t, framework.Global, ctx); err != nil {
		t.Fatal(err)
	}

	if err = memcachedMetricsTest(t, framework.Global, ctx); err != nil {
		t.Fatal(err)
	}
}

func memcachedMetricsTest(t *testing.T, f *framework.Framework, ctx *framework.TestCtx) error {
	namespace, err := ctx.GetNamespace()
	if err != nil {
		return err
	}

	// Make sure metrics Service exists
	s := v1.Service{}
	err = f.Client.Get(context.TODO(), types.NamespacedName{Name: operatorName, Namespace: namespace}, &s)
	if err != nil {
		return fmt.Errorf("could not get metrics Service: (%v)", err)
	}

	// Get operator pod
	pods := v1.PodList{}
	opts := client.InNamespace(namespace)
	if len(s.Spec.Selector) == 0 {
		return fmt.Errorf("no labels found in metrics Service")
	}

	for k, v := range s.Spec.Selector {
		if err := opts.SetLabelSelector(fmt.Sprintf("%s=%s", k, v)); err != nil {
			return fmt.Errorf("failed to set list label selector: (%v)", err)
		}
	}

	if err := opts.SetFieldSelector("status.phase=Running"); err != nil {
		return fmt.Errorf("failed to set list field selector: (%v)", err)
	}
	err = f.Client.List(context.TODO(), opts, &pods)
	if err != nil {
		return fmt.Errorf("failed to get pods: (%v)", err)
	}

	podName := ""
	numPods := len(pods.Items)
	// TODO(lili): Remove below logic when we enable exposing metrics in all pods.
	if numPods == 0 {
		podName = pods.Items[0].Name
	} else if numPods > 1 {
		// If we got more than one pod, get leader pod name.
		leader, err := verifyLeader(t, namespace, f, s.Spec.Selector)
		if err != nil {
			return err
		}
		podName = leader.Name
	} else {
		return fmt.Errorf("failed to get operator pod: could not select any pods with Service selector %v", s.Spec.Selector)
	}
	// Pod name must be there, otherwise we cannot read metrics data via pod proxy.
	if podName == "" {
		return fmt.Errorf("failed to get pod name")
	}

	// Get metrics data
	request := proxyViaPod(f.KubeClient, namespace, podName, "8383", "/metrics")
	response, err := request.DoRaw()
	if err != nil {
		return fmt.Errorf("failed to get response from metrics: %v", err)
	}

	// Make sure metrics are present
	if len(response) == 0 {
		return fmt.Errorf("metrics body is empty")
	}

	// Perform prometheus metrics lint checks
	l := promlint.New(bytes.NewReader(response))
	problems, err := l.Lint()
	if err != nil {
		return fmt.Errorf("failed to lint metrics: %v", err)
	}
	// TODO(lili): Change to 0, when we upgrade to 1.14.
	// currently there is a problem with one of the metrics in upstream Kubernetes:
	// `workqueue_longest_running_processor_microseconds`.
	// This has been fixed in 1.14 release.
	if len(problems) > 1 {
		return fmt.Errorf("found problems with metrics: %#+v", problems)
	}

	return nil
}

func proxyViaPod(kubeClient kubernetes.Interface, namespace, podName, podPortName, path string) *rest.Request {
	return kubeClient.
		CoreV1().
		RESTClient().
		Get().
		Namespace(namespace).
		Resource("pods").
		SubResource("proxy").
		Name(fmt.Sprintf("%s:%s", podName, podPortName)).
		Suffix(path)
}
