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
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/operator-framework/operator-sdk/internal/pkg/scaffold"
	"github.com/operator-framework/operator-sdk/internal/util/fileutil"
	"github.com/operator-framework/operator-sdk/internal/util/yamlutil"
	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/prometheus/util/promlint"
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
	sdkRepo                     = "github.com/operator-framework/operator-sdk"
	operatorName                = "memcached-operator"
	testRepo                    = "github.com/example-inc/" + operatorName
)

func TestMemcached(t *testing.T) {
	// get global framework variables
	ctx := framework.NewTestCtx(t)
	defer ctx.Cleanup()
	sdkTestE2EDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(sdkTestE2EDir); err != nil {
			t.Errorf("Failed to change back to original working directory: (%v)", err)
		}
	}()
	localSDKPath := *args.localRepo
	if localSDKPath == "" {
		// We're in ${sdk_repo}/test/e2e
		localSDKPath = filepath.Dir(filepath.Dir(sdkTestE2EDir))
	}
	// For go commands in operator projects.
	if err = os.Setenv("GO111MODULE", "on"); err != nil {
		t.Fatal(err)
	}

	// Setup
	absProjectPath, err := ioutil.TempDir("", "tmp.")
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
		operatorName,
		"--repo", testRepo,
		"--skip-validation").CombinedOutput()
	if err != nil {
		t.Fatalf("Error: %v\nCommand Output: %s\n", err, string(cmdOut))
	}

	if err := os.Chdir(operatorName); err != nil {
		t.Fatalf("Failed to change to %s directory: (%v)", operatorName, err)
	}

	replace := getGoModReplace(t, localSDKPath)
	if replace.repo != sdkRepo {
		if replace.isLocal {
			// A hacky way to get local module substitution to work is to write a
			// stub go.mod into the local SDK repo referred to in
			// memcached-operator's go.mod, which allows go to recognize
			// the local SDK repo as a module.
			sdkModPath := filepath.Join(filepath.FromSlash(replace.repo), "go.mod")
			if _, err = os.Stat(sdkModPath); err != nil && os.IsNotExist(err) {
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
		}
		modBytes, err := insertGoModReplace(t, sdkRepo, replace.repo, replace.ref)
		if err != nil {
			t.Fatalf("Failed to insert go.mod replace: %v", err)
		}
		t.Logf("go.mod: %v", string(modBytes))
	}

	cmdOut, err = exec.Command("go", "build", "./...").CombinedOutput()
	if err != nil {
		t.Fatalf("Command \"go build ./...\" failed after modifying go.mod: %v\nCommand Output:\n%v", err, string(cmdOut))
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
		cmdOut, err = exec.Command("cp", src, dst).CombinedOutput()
		if err != nil {
			t.Fatalf("Error: %v\nCommand Output: %s\n", err, string(cmdOut))
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
	cmdOut, err = exec.Command("go", "build", "./...").CombinedOutput()
	if err != nil {
		t.Fatalf("Command \"go build ./...\" failed: %v\nCommand Output:\n%v", err, string(cmdOut))
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

func insertGoModReplace(t *testing.T, repo, path, sha string) ([]byte, error) {
	modBytes, err := ioutil.ReadFile("go.mod")
	if err != nil {
		return nil, errors.Wrap(err, "failed to read go.mod")
	}
	sdkReplace := fmt.Sprintf("replace %s => %s", repo, path)
	if sha != "" {
		sdkReplace = fmt.Sprintf("%s %s", sdkReplace, sha)
	}
	modBytes = append(modBytes, []byte("\n"+sdkReplace)...)
	err = ioutil.WriteFile("go.mod", modBytes, fileutil.DefaultFileMode)
	if err != nil {
		return nil, errors.Wrap(err, "failed to write go.mod before replacing SDK repo")
	}
	return modBytes, nil
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
	local := *args.e2eImageName == ""
	if local {
		*args.e2eImageName = "quay.io/example/memcached-operator:v0.0.1"
		if err != nil {
			t.Fatal(err)
		}
		operatorYAML = bytes.Replace(operatorYAML, []byte("imagePullPolicy: Always"), []byte("imagePullPolicy: Never"), 1)
		err = ioutil.WriteFile("deploy/operator.yaml", operatorYAML, fileutil.DefaultFileMode)
		if err != nil {
			t.Fatal(err)
		}
	}
	operatorYAML = bytes.Replace(operatorYAML, []byte("REPLACE_IMAGE"), []byte(*args.e2eImageName), 1)
	err = ioutil.WriteFile("deploy/operator.yaml", operatorYAML, os.FileMode(0644))
	if err != nil {
		t.Fatalf("Failed to write deploy/operator.yaml: %v", err)
	}
	t.Log("Building operator docker image")
	cmdOut, err := exec.Command("operator-sdk", "build", *args.e2eImageName).CombinedOutput()
	if err != nil {
		t.Fatalf("Error: %v\nCommand Output: %s\n", err, string(cmdOut))
	}

	if !local {
		t.Log("Pushing docker image to repo")
		cmdOut, err = exec.Command("docker", "push", *args.e2eImageName).CombinedOutput()
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

	if err = memcachedOperatorMetricsTest(t, framework.Global, ctx); err != nil {
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
	if len(s.Spec.Selector) == 0 {
		return fmt.Errorf("no labels found in metrics Service")
	}

	// TODO(lili): Make port a constant in internal/scaffold/cmd.go.
	response, err := getMetrics(t, f, s.Spec.Selector, namespace, "8383")
	if err != nil {
		return fmt.Errorf("failed to get metrics: %v", err)
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

func memcachedOperatorMetricsTest(t *testing.T, f *framework.Framework, ctx *framework.TestCtx) error {
	namespace, err := ctx.GetNamespace()
	if err != nil {
		return err
	}

	// TODO(lili): Make port a constant in internal/scaffold/cmd.go.
	response, err := getMetrics(t, f, map[string]string{"name": operatorName}, namespace, "8686")
	if err != nil {
		return fmt.Errorf("failed to lint metrics: %v", err)
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
	if len(problems) > 0 {
		return fmt.Errorf("found problems with metrics: %#+v", problems)
	}

	// Make sure the metrics are the way we expect them.
	d := expfmt.NewDecoder(bytes.NewReader(response), expfmt.FmtText)
	var mf dto.MetricFamily
	for {
		if err := d.Decode(&mf); err != nil {
			if err == io.EOF {
				break
			}

			return err
		}

		/*
			Metric:
			# HELP memcached_info Information about the Memcached operator replica.
			# TYPE memcached_info gauge
			memcached_info{namespace="memcached-memcached-group-cluster-1553683239",memcached="example-memcached"} 1
		*/
		if mf.GetName() != "memcached_info" {
			return fmt.Errorf("metric name was incorrect: expected %s, got %s", "memcached_info", mf.GetName())
		}
		if mf.GetType() != dto.MetricType_GAUGE {
			return fmt.Errorf("metric type was incorrect: expected %v, got %v", dto.MetricType_GAUGE, mf.GetType())
		}

		mlabels := mf.Metric[0].GetLabel()
		if mlabels[0].GetName() != "namespace" {
			return fmt.Errorf("metric label name was incorrect: expected %s, got %s", "namespace", mlabels[0].GetName())
		}
		if mlabels[0].GetValue() != namespace {
			return fmt.Errorf("metric label value was incorrect: expected %s, got %s", namespace, mlabels[0].GetValue())
		}
		if mlabels[1].GetName() != "memcached" {
			return fmt.Errorf("metric label name was incorrect: expected %s, got %s", "memcached", mlabels[1].GetName())
		}
		if mlabels[1].GetValue() != "example-memcached" {
			return fmt.Errorf("metric label value was incorrect: expected %s, got %s", "example-memcached", mlabels[1].GetValue())
		}

		if mf.Metric[0].GetGauge().GetValue() != float64(1) {
			return fmt.Errorf("metric counter was incorrect: expected %f, got %f", float64(1), mf.Metric[0].GetGauge().GetValue())
		}
	}

	return nil
}

func getMetrics(t *testing.T, f *framework.Framework, label map[string]string, ns, port string) ([]byte, error) {
	// Get operator pod
	pods := v1.PodList{}
	opts := client.InNamespace(ns)
	for k, v := range label {
		if err := opts.SetLabelSelector(fmt.Sprintf("%s=%s", k, v)); err != nil {
			return nil, fmt.Errorf("failed to set list label selector: (%v)", err)
		}
	}
	if err := opts.SetFieldSelector("status.phase=Running"); err != nil {
		return nil, fmt.Errorf("failed to set list field selector: (%v)", err)
	}
	err := f.Client.List(context.TODO(), opts, &pods)
	if err != nil {
		return nil, fmt.Errorf("failed to get pods: (%v)", err)
	}

	podName := ""
	numPods := len(pods.Items)
	// TODO(lili): Remove below logic when we enable exposing metrics in all pods.
	if numPods == 0 {
		podName = pods.Items[0].Name
	} else if numPods > 1 {
		// If we got more than one pod, get leader pod name.
		leader, err := verifyLeader(t, ns, f, label)
		if err != nil {
			return nil, err
		}
		podName = leader.Name
	} else {
		return nil, fmt.Errorf("failed to get operator pod: could not select any pods with selector %v", label)
	}
	// Pod name must be there, otherwise we cannot read metrics data via pod proxy.
	if podName == "" {
		return nil, fmt.Errorf("failed to get pod name")
	}

	// Get metrics data
	request := proxyViaPod(f.KubeClient, ns, podName, port, "/metrics")
	response, err := request.DoRaw()
	if err != nil {
		return nil, fmt.Errorf("failed to get response from metrics: %v", err)
	}

	return response, nil

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
