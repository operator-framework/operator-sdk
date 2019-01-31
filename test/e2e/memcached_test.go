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
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/ghodss/yaml"
	"github.com/operator-framework/operator-sdk/internal/util/fileutil"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"
	"github.com/operator-framework/operator-sdk/internal/util/yamlutil"
	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	crYAML               string = "apiVersion: \"cache.example.com/v1alpha1\"\nkind: \"Memcached\"\nmetadata:\n  name: \"example-memcached\"\nspec:\n  size: 3"
	retryInterval               = time.Second * 5
	timeout                     = time.Second * 60
	cleanupRetryInterval        = time.Second * 1
	cleanupTimeout              = time.Second * 10
)

func TestMemcached(t *testing.T) {
	// get global framework variables
	ctx := framework.NewTestCtx(t)
	defer ctx.Cleanup()
	gopath, ok := os.LookupEnv(projutil.GopathEnv)
	if !ok {
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

	// Setup
	absProjectPath := filepath.Join(gopath, "src/github.com/example-inc")
	if err := os.MkdirAll(absProjectPath, fileutil.DefaultDirFileMode); err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(absProjectPath); err != nil {
		t.Fatal(err)
	}

	t.Log("Creating new operator project")
	cmdOut, err := exec.Command("operator-sdk",
		"new",
		"memcached-operator").CombinedOutput()
	if err != nil {
		// HACK: dep cannot resolve non-master branches as the base branch for PR's,
		// so running `dep ensure` will fail when first running
		// `operator-sdk new ...`. For now we can ignore the first solve failure.
		// A permanent solution can be implemented once the following is merged:
		// https://github.com/golang/dep/pull/1658
		solveFailRe := regexp.MustCompile(`(?m)^[ \t]*Solving failure:.+github\.com/operator-framework/operator-sdk.+:$`)
		if !solveFailRe.Match(cmdOut) {
			t.Fatalf("Error: %v\nCommand Output: %s\n", err, string(cmdOut))
		}
	}
	ctx.AddCleanupFn(func() error { return os.RemoveAll(absProjectPath) })

	if err := os.Chdir("memcached-operator"); err != nil {
		t.Fatalf("Failed to change to memcached-operator directory: (%v)", err)
	}
	repo, ok := os.LookupEnv("TRAVIS_PULL_REQUEST_SLUG")
	if repo == "" {
		repo, ok = os.LookupEnv("TRAVIS_REPO_SLUG")
	}
	if ok && repo != "" && repo != "operator-framework/operator-sdk" {
		commitSha, ok := os.LookupEnv("TRAVIS_PULL_REQUEST_SHA")
		if commitSha == "" {
			commitSha, ok = os.LookupEnv("TRAVIS_COMMIT")
		}
		if ok && commitSha != "" {
			gopkg, err := ioutil.ReadFile("Gopkg.toml")
			if err != nil {
				t.Fatal(err)
			}
			// Match against the '#osdk_branch_annotation' used for version substitution
			// and comment out the current branch.
			branchRe := regexp.MustCompile("([ ]+)(.+#osdk_branch_annotation)")
			gopkg = branchRe.ReplaceAll(gopkg, []byte("$1# $2"))
			versionRe := regexp.MustCompile("([ ]+)(.+#osdk_version_annotation)")
			gopkg = versionRe.ReplaceAll(gopkg, []byte("$1# $2"))
			// Plug in the fork to test against so `dep ensure` can resolve dependencies
			// correctly.
			gopkgString := string(gopkg)
			gopkgLoc := strings.LastIndex(gopkgString, "\n  name = \"github.com/operator-framework/operator-sdk\"\n")
			gopkgString = gopkgString[:gopkgLoc] + "\n  source = \"https://github.com/" + repo + "\"\n  revision = \"" + commitSha + "\"\n" + gopkgString[gopkgLoc+1:]
			err = ioutil.WriteFile("Gopkg.toml", []byte(gopkgString), fileutil.DefaultFileMode)
			if err != nil {
				t.Fatalf("Failed to write updated Gopkg.toml: %v", err)
			}

			t.Logf("Gopkg.toml: %v", gopkgString)
		} else {
			t.Fatal("Could not find sha of PR")
		}
	}
	cmdOut, err = exec.Command("dep", "ensure").CombinedOutput()
	if err != nil {
		t.Fatalf("Error after modifying Gopkg.toml: %v\nCommand Output: %s\n", err, string(cmdOut))
	}

	// Temporarily disabling the leader election test due to GitHub issue #920 and PR #932.
	// TODO: Update this test so that it works with the changes from #932
	//
	// // Set replicas to 2 to test leader election. In production, this should
	// // almost always be set to 1, because there isn't generally value in having
	// // a hot spare operator process.
	// opYaml, err := ioutil.ReadFile("deploy/operator.yaml")
	// if err != nil {
	// 	t.Fatalf("Could not read deploy/operator.yaml: %v", err)
	// }
	// newOpYaml := bytes.Replace(opYaml, []byte("replicas: 1"), []byte("replicas: 2"), 1)
	// err = ioutil.WriteFile("deploy/operator.yaml", newOpYaml, 0644)
	// if err != nil {
	// 	t.Fatalf("Could not write deploy/operator.yaml: %v", err)
	// }

	cmd := exec.Command("operator-sdk",
		"add",
		"api",
		"--api-version=cache.example.com/v1alpha1",
		"--kind=Memcached")
	// Generators will print errors if -v is set.
	if !projutil.IsGoVerbose() {
		os.Setenv(projutil.GoFlagsEnv, os.Getenv(projutil.GoFlagsEnv)+" -v")
	}
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

	cmdOut, err = exec.Command("cp", "-a", filepath.Join(gopath, "src/github.com/operator-framework/operator-sdk/example/memcached-operator/memcached_controller.go.tmpl"),
		"pkg/controller/memcached/memcached_controller.go").CombinedOutput()
	if err != nil {
		t.Fatalf("Could not copy memcached example to to pkg/controller/memcached/memcached_controller.go: %v\nCommand Output:\n%v", err, string(cmdOut))
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

	t.Log("Copying test files to ./test")
	if err = os.MkdirAll("./test", fileutil.DefaultDirFileMode); err != nil {
		t.Fatalf("Could not create test/e2e dir: %v", err)
	}
	cmdOut, err = exec.Command("cp", "-a", filepath.Join(gopath, "src/github.com/operator-framework/operator-sdk/test/e2e/incluster-test-code"), "./test/e2e").CombinedOutput()
	if err != nil {
		t.Fatalf("Could not copy tests to test/e2e: %v\nCommand Output:\n%v", err, string(cmdOut))
	}
	// fix naming of files
	cmdOut, err = exec.Command("mv", "test/e2e/main_test.go.tmpl", "test/e2e/main_test.go").CombinedOutput()
	if err != nil {
		t.Fatalf("Could not rename test/e2e/main_test.go.tmpl: %v\nCommand Output:\n%v", err, string(cmdOut))
	}
	cmdOut, err = exec.Command("mv", "test/e2e/memcached_test.go.tmpl", "test/e2e/memcached_test.go").CombinedOutput()
	if err != nil {
		t.Fatalf("Could not rename test/e2e/memcached_test.go.tmpl: %v\nCommand Output:\n%v", err, string(cmdOut))
	}

	t.Log("Pulling new dependencies with dep ensure")
	cmdOut, err = exec.Command("dep", "ensure").CombinedOutput()
	if err != nil {
		t.Fatalf("Command 'dep ensure' failed: %v\nCommand Output:\n%v", err, string(cmdOut))
	}
	// link local sdk to vendor if not in travis
	if repo == "" {
		for _, dir := range []string{"pkg", "internal"} {
			repoDir := filepath.Join("github.com/operator-framework/operator-sdk", dir)
			vendorDir := filepath.Join("vendor", repoDir)
			if err := os.RemoveAll(vendorDir); err != nil {
				t.Fatalf("Failed to delete old vendor directory: (%v)", err)
			}
			if err := os.Symlink(filepath.Join(gopath, projutil.SrcDir, repoDir), vendorDir); err != nil {
				t.Fatalf("Failed to symlink local operator-sdk project to vendor dir: (%v)", err)
			}
		}
	}

	file, err := yamlutil.GenerateCombinedGlobalManifest()
	if err != nil {
		t.Fatal(err)
	}
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
		t.Run("ClusterTest", MemcachedClusterTest)
		t.Run("Local", MemcachedLocal)
	})
}

func memcachedLeaderTest(t *testing.T, f *framework.Framework, ctx *framework.TestCtx) error {
	namespace, err := ctx.GetNamespace()
	if err != nil {
		return err
	}

	err = e2eutil.WaitForOperatorDeployment(t, f.KubeClient, namespace, "memcached-operator", 1, retryInterval, timeout)
	if err != nil {
		return err
	}

	leader, err := verifyLeader(t, namespace, f)
	if err != nil {
		return err
	}

	// delete the leader's pod so a new leader will get elected
	err = f.Client.Delete(context.TODO(), leader)
	if err != nil {
		return err
	}

	err = e2eutil.WaitForOperatorDeployment(t, f.KubeClient, namespace, "memcached-operator", 1, retryInterval, timeout)
	if err != nil {
		return err
	}

	newLeader, err := verifyLeader(t, namespace, f)
	if err != nil {
		return err
	}
	if newLeader.Name == leader.Name {
		return fmt.Errorf("leader pod name did not change across pod delete")
	}

	return nil
}

func verifyLeader(t *testing.T, namespace string, f *framework.Framework) (*v1.Pod, error) {
	// get configmap, which is the lock
	lock := v1.ConfigMap{}
	err := f.Client.Get(context.TODO(), types.NamespacedName{Name: "memcached-operator-lock", Namespace: namespace}, &lock)
	if err != nil {
		return nil, err
	}

	owners := lock.GetOwnerReferences()
	if len(owners) != 1 {
		return nil, fmt.Errorf("leader lock has %d owner refs, expected 1", len(owners))
	}
	owner := owners[0]

	// get operator pods
	pods := v1.PodList{}
	opts := client.ListOptions{Namespace: namespace}
	if err := opts.SetLabelSelector("name=memcached-operator"); err != nil {
		t.Fatalf("Failed to set list label selector: (%v)", err)
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
	cmdOut, err := exec.Command("operator-sdk", "build", *e2eImageName,
		"--enable-tests",
		"--test-location", "./test/e2e",
		"--namespaced-manifest", "deploy/operator.yaml").CombinedOutput()
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

	file, err := yamlutil.GenerateCombinedNamespacedManifest()
	if err != nil {
		t.Fatal(err)
	}
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
	err = e2eutil.WaitForOperatorDeployment(t, framework.Global.KubeClient, namespace, "memcached-operator", 1, retryInterval, timeout)
	if err != nil {
		t.Fatal(err)
	}

	// Temporarily disabling the leader election test due to GitHub issue #920 and PR #932.
	// TODO: Update this test so that it works with the changes from #932
	// if err = memcachedLeaderTest(t, framework.Global, ctx); err != nil {
	// 	t.Fatal(err)
	// }

	if err = memcachedScaleTest(t, framework.Global, ctx); err != nil {
		t.Fatal(err)
	}
}

func MemcachedClusterTest(t *testing.T) {
	// get global framework variables
	ctx := framework.NewTestCtx(t)
	defer ctx.Cleanup()

	// create sa
	filename := "deploy/service_account.yaml"
	framework.Global.NamespacedManPath = &filename
	err := ctx.InitializeClusterResources(&framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		t.Fatal(err)
	}
	t.Log("Created sa")

	// create rbac
	filename = "deploy/role.yaml"
	framework.Global.NamespacedManPath = &filename
	err = ctx.InitializeClusterResources(&framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		t.Fatal(err)
	}
	t.Log("Created role")

	filename = "deploy/role_binding.yaml"
	framework.Global.NamespacedManPath = &filename
	err = ctx.InitializeClusterResources(&framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		t.Fatal(err)
	}
	t.Log("Created role_binding")

	namespace, err := ctx.GetNamespace()
	if err != nil {
		t.Fatalf("Could not get namespace: %v", err)
	}
	cmdOut, err := exec.Command("operator-sdk", "test", "cluster", *e2eImageName,
		"--namespace", namespace,
		"--image-pull-policy", "Never",
		"--service-account", "memcached-operator").CombinedOutput()
	if err != nil {
		t.Fatalf("In-cluster test failed: %v\nCommand Output:\n%s", err, string(cmdOut))
	}
}
