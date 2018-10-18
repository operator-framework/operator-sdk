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
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/operator-framework/operator-sdk/test/e2e/e2eutil"
	framework "github.com/operator-framework/operator-sdk/test/e2e/framework"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	filemode os.FileMode = 0664
	dirmode  os.FileMode = 0750
)

func TestMemcached(t *testing.T) {
	// get global framework variables
	f := framework.Global
	ctx := f.NewTestCtx(t)
	defer ctx.Cleanup(t)
	gopath, ok := os.LookupEnv("GOPATH")
	if !ok {
		t.Fatalf("$GOPATH not set")
	}
	cd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		os.Chdir(cd)
	}()

	// Setup
	absProjectPath := filepath.Join(gopath, "src/github.com/example-inc")
	if err := os.MkdirAll(absProjectPath, dirmode); err != nil {
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
		t.Fatalf("error: %v\nCommand Output: %s\n", err, string(cmdOut))
	}
	ctx.AddFinalizerFn(func() error { return os.RemoveAll(absProjectPath) })

	os.Chdir("memcached-operator")
	cmdOut, err = exec.Command("operator-sdk",
		"add",
		"api",
		"--api-version=cache.example.com/v1alpha1",
		"--kind=Memcached").CombinedOutput()
	if err != nil {
		t.Fatalf("error: %v\nCommand Output: %s\n", err, string(cmdOut))
	}
	cmdOut, err = exec.Command("operator-sdk",
		"add",
		"controller",
		"--api-version=cache.example.com/v1alpha1",
		"--kind=Memcached").CombinedOutput()
	if err != nil {
		t.Fatalf("error: %v\nCommand Output: %s\n", err, string(cmdOut))
	}

	cmdOut, err = exec.Command("cp", "-a", filepath.Join(gopath, "src/github.com/operator-framework/operator-sdk/example/memcached-operator/memcached_controller.go.tmpl"),
		"pkg/controller/memcached/memcached_controller.go").CombinedOutput()
	if err != nil {
		t.Fatalf("could not copy memcached example to to pkg/controller/memcached/memcached_controller.go: %v\nCommand Output:\n%v", err, string(cmdOut))
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
	os.Remove("pkg/apis/cache/v1alpha1/memcached_types.go")
	err = ioutil.WriteFile("pkg/apis/cache/v1alpha1/memcached_types.go", bytes.Join(memcachedTypesFileLines, []byte("\n")), filemode)
	if err != nil {
		t.Fatal(err)
	}

	t.Log("Generating k8s")
	cmdOut, err = exec.Command("operator-sdk", "generate", "k8s").CombinedOutput()
	if err != nil {
		t.Fatalf("error: %v\nCommand Output: %s\n", err, string(cmdOut))
	}

	t.Log("Copying test files to ./test")
	if err = os.MkdirAll("./test", dirmode); err != nil {
		t.Fatalf("could not create test/e2e dir: %v", err)
	}
	cmdOut, err = exec.Command("cp", "-a", filepath.Join(gopath, "src/github.com/operator-framework/operator-sdk/test/e2e/incluster-test-code"), "./test/e2e").CombinedOutput()
	if err != nil {
		t.Fatalf("could not copy tests to test/e2e: %v\nCommand Output:\n%v", err, string(cmdOut))
	}
	// fix naming of files
	cmdOut, err = exec.Command("mv", "test/e2e/main_test.go.tmpl", "test/e2e/main_test.go").CombinedOutput()
	if err != nil {
		t.Fatalf("could not rename test/e2e/main_test.go.tmpl: %v\nCommand Output:\n%v", err, string(cmdOut))
	}
	cmdOut, err = exec.Command("mv", "test/e2e/memcached_test.go.tmpl", "test/e2e/memcached_test.go").CombinedOutput()
	if err != nil {
		t.Fatalf("could not rename test/e2e/memcached_test.go.tmpl: %v\nCommand Output:\n%v", err, string(cmdOut))
	}
	t.Log("Pulling new dependencies with dep ensure")
	prSlug, ok := os.LookupEnv("TRAVIS_PULL_REQUEST_SLUG")
	if ok && prSlug != "" {
		prSha, ok := os.LookupEnv("TRAVIS_PULL_REQUEST_SHA")
		if ok && prSha != "" {
			gopkg, err := ioutil.ReadFile("Gopkg.toml")
			if err != nil {
				t.Fatal(err)
			}
			// Match against the '#osdk_branch_annotation' used for version substitution
			// and comment out the current branch.
			branchRe := regexp.MustCompile("([ ]+)(.+#osdk_branch_annotation)")
			gopkg = branchRe.ReplaceAll(gopkg, []byte("$1# $2"))
			// Plug in the fork to test against so `dep ensure` can resolve dependencies
			// correctly.
			gopkgString := string(gopkg)
			gopkgLoc := strings.LastIndex(gopkgString, "\n  name = \"github.com/operator-framework/operator-sdk\"\n")
			gopkgString = gopkgString[:gopkgLoc] + "\n  source = \"https://github.com/" + prSlug + "\"\n  revision = \"" + prSha + "\"\n" + gopkgString[gopkgLoc+1:]
			err = ioutil.WriteFile("Gopkg.toml", []byte(gopkgString), filemode)
			if err != nil {
				t.Fatalf("failed to write updated Gopkg.toml: %v", err)
			}
			t.Logf("Gopkg.toml: %v", gopkgString)
		} else {
			t.Fatal("could not find sha of PR")
		}
	}
	cmdOut, err = exec.Command("dep", "ensure").CombinedOutput()
	if err != nil {
		t.Fatalf("dep ensure failed: %v\nCommand Output:\n%v", err, string(cmdOut))
	}
	// link local sdk to vendor if not in travis
	if prSlug == "" {
		os.RemoveAll("vendor/github.com/operator-framework/operator-sdk/pkg")
		os.Symlink(filepath.Join(gopath, "src/github.com/operator-framework/operator-sdk/pkg"),
			"vendor/github.com/operator-framework/operator-sdk/pkg")
	}

	// create crd
	crdYAML, err := ioutil.ReadFile("deploy/crds/cache_v1alpha1_memcached_crd.yaml")
	if err != nil {
		t.Fatalf("could not read crd file: %v", err)
	}
	err = ctx.CreateFromYAML(crdYAML)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("Created crd")

	// run subtests
	t.Run("memcached-group", func(t *testing.T) {
		t.Run("Cluster", MemcachedCluster)
		t.Run("ClusterTest", MemcachedClusterTest)
		t.Run("Local", MemcachedLocal)
	})
}

func memcachedScaleTest(t *testing.T, f *framework.Framework, ctx framework.TestCtx) error {
	// create example-memcached yaml file
	err := ioutil.WriteFile("deploy/cr.yaml",
		[]byte("apiVersion: \"cache.example.com/v1alpha1\"\nkind: \"Memcached\"\nmetadata:\n  name: \"example-memcached\"\nspec:\n  size: 3"),
		filemode)
	if err != nil {
		return err
	}

	// create memcached custom resource
	crYAML, err := ioutil.ReadFile("deploy/cr.yaml")
	if err != nil {
		return err
	}
	err = ctx.CreateFromYAML(crYAML)
	if err != nil {
		return err
	}
	namespace, err := ctx.GetNamespace()
	if err != nil {
		return err
	}
	// wait for example-memcached to reach 3 replicas
	err = e2eutil.DeploymentReplicaCheck(t, f.KubeClient, namespace, "example-memcached", 3, 6)
	if err != nil {
		return err
	}

	// update memcached CR size to 4
	memcachedClient, err := ctx.GetCRClient(crYAML)
	if err != nil {
		return err
	}
	err = memcachedClient.Patch(types.JSONPatchType).
		Namespace(namespace).
		Resource("memcacheds").
		Name("example-memcached").
		Body([]byte("[{\"op\": \"replace\", \"path\": \"/spec/size\", \"value\": 4}]")).
		Do().
		Error()
	if err != nil {
		return err
	}

	// wait for example-memcached to reach 4 replicas
	return e2eutil.DeploymentReplicaCheck(t, f.KubeClient, namespace, "example-memcached", 4, 6)
}

func MemcachedLocal(t *testing.T) {
	// get global framework variables
	f := framework.Global
	ctx := f.NewTestCtx(t)
	defer ctx.Cleanup(t)
	namespace, err := ctx.GetNamespace()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("operator-sdk", "up", "local", "--namespace="+namespace)
	stderr, err := os.Create("stderr.txt")
	if err != nil {
		t.Fatalf("failed to create stderr.txt: %v", err)
	}
	cmd.Stderr = stderr
	defer stderr.Close()

	err = cmd.Start()
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	ctx.AddFinalizerFn(func() error { return cmd.Process.Signal(os.Interrupt) })

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
		t.Fatalf("local operator not ready after 100 seconds: %v\n", err)
	}

	if err = memcachedScaleTest(t, f, ctx); err != nil {
		t.Fatal(err)
	}
}

func MemcachedCluster(t *testing.T) {
	// get global framework variables
	f := framework.Global
	ctx := f.NewTestCtx(t)
	defer ctx.Cleanup(t)
	operatorYAML, err := ioutil.ReadFile("deploy/operator.yaml")
	if err != nil {
		t.Fatalf("could not read deploy/operator.yaml: %v", err)
	}
	local := *f.ImageName == ""
	if local {
		*f.ImageName = "quay.io/example/memcached-operator:v0.0.1"
		if err != nil {
			t.Fatal(err)
		}
		operatorYAML = bytes.Replace(operatorYAML, []byte("imagePullPolicy: Always"), []byte("imagePullPolicy: Never"), 1)
		err = ioutil.WriteFile("deploy/operator.yaml", operatorYAML, filemode)
		if err != nil {
			t.Fatal(err)
		}
	}
	operatorYAML = bytes.Replace(operatorYAML, []byte("REPLACE_IMAGE"), []byte(*f.ImageName), 1)
	err = ioutil.WriteFile("deploy/operator.yaml", operatorYAML, os.FileMode(0644))
	if err != nil {
		t.Fatalf("failed to write deploy/operator.yaml: %v", err)
	}
	t.Log("Building operator docker image")
	cmdOut, err := exec.Command("operator-sdk", "build", *f.ImageName,
		"--enable-tests",
		"--test-location", "./test/e2e",
		"--namespaced-manifest", "deploy/operator.yaml").CombinedOutput()
	if err != nil {
		t.Fatalf("error: %v\nCommand Output: %s\n", err, string(cmdOut))
	}

	if !local {
		t.Log("Pushing docker image to repo")
		cmdOut, err = exec.Command("docker", "push", *f.ImageName).CombinedOutput()
		if err != nil {
			t.Fatalf("error: %v\nCommand Output: %s\n", err, string(cmdOut))
		}
	}

	// create sa
	saYAML, err := ioutil.ReadFile("deploy/service_account.yaml")
	if err != nil {
		t.Fatal(err)
	}
	err = ctx.CreateFromYAML(saYAML)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("Created sa")

	// create rbac
	roleYAML, err := ioutil.ReadFile("deploy/role.yaml")
	if err != nil {
		t.Fatalf("could not read role file: %v", err)
	}
	err = ctx.CreateFromYAML(roleYAML)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("Created role")
	roleBindingYAML, err := ioutil.ReadFile("deploy/role_binding.yaml")
	if err != nil {
		t.Fatalf("could not read role_binding file: %v", err)
	}
	err = ctx.CreateFromYAML(roleBindingYAML)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("Created role_binding")

	// create operator
	operatorYAML, err = ioutil.ReadFile("deploy/operator.yaml")
	if err != nil {
		t.Fatal(err)
	}
	err = ctx.CreateFromYAML(operatorYAML)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("Created operator")

	namespace, err := ctx.GetNamespace()
	if err != nil {
		t.Fatal(err)
	}
	// wait for memcached-operator to be ready
	err = e2eutil.DeploymentReplicaCheck(t, f.KubeClient, namespace, "memcached-operator", 1, 6)
	if err != nil {
		t.Fatal(err)
	}

	if err = memcachedScaleTest(t, f, ctx); err != nil {
		t.Fatal(err)
	}
}

func MemcachedClusterTest(t *testing.T) {
	// get global framework variables
	f := framework.Global
	ctx := f.NewTestCtx(t)
	defer ctx.Cleanup(t)

	// create sa
	saYAML, err := ioutil.ReadFile("deploy/service_account.yaml")
	if err != nil {
		t.Fatal(err)
	}
	err = ctx.CreateFromYAML(saYAML)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("Created sa")

	// create rbac
	roleYAML, err := ioutil.ReadFile("deploy/role.yaml")
	if err != nil {
		t.Fatalf("could not read role file: %v", err)
	}
	err = ctx.CreateFromYAML(roleYAML)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("Created role")
	roleBindingYAML, err := ioutil.ReadFile("deploy/role_binding.yaml")
	if err != nil {
		t.Fatalf("could not read role_binding file: %v", err)
	}
	err = ctx.CreateFromYAML(roleBindingYAML)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("Created role_binding")

	namespace, err := ctx.GetNamespace()
	if err != nil {
		t.Fatalf("could not get namespace: %v", err)
	}
	cmdOut, err := exec.Command("operator-sdk", "test", "cluster", *f.ImageName,
		"--namespace", namespace,
		"--image-pull-policy", "Never",
		"--service-account", "memcached-operator").CombinedOutput()
	if err != nil {
		t.Fatalf("in-cluster test failed: %v\nCommand Output:\n%s", err, string(cmdOut))
	}

}
