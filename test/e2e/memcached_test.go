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
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path"
	"testing"
	"time"

	"github.com/operator-framework/operator-sdk/test/e2e/e2eutil"
	framework "github.com/operator-framework/operator-sdk/test/e2e/framework"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	filemode = int(0664)
	// amount of lines to remove from end of types file to allow us to fill in the
	// blank structs
	typesFileTrimAmount = 7
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
	os.Chdir(path.Join(gopath, "/src/github.com/example-inc"))
	t.Log("Creating new operator project")
	cmdOut, err := exec.Command("operator-sdk",
		"new",
		"memcached-operator",
		"--api-version=cache.example.com/v1alpha1",
		"--kind=Memcached").CombinedOutput()
	if err != nil {
		t.Fatalf("error: %v\nCommand Output: %s\n", err, string(cmdOut))
	}
	ctx.AddFinalizerFn(func() error { return os.RemoveAll(path.Join(gopath, "/src/github.com/example-inc/memcached-operator")) })

	os.Chdir("memcached-operator")
	os.RemoveAll("vendor/github.com/operator-framework/operator-sdk/pkg")
	os.Symlink(path.Join(gopath, "/src/github.com/operator-framework/operator-sdk/pkg"),
		"vendor/github.com/operator-framework/operator-sdk/pkg")
	handlerFile, err := os.Create("pkg/stub/handler.go")
	if err != nil {
		t.Fatal(err)
	}
	ctx.AddFinalizerFn(func() error { return handlerFile.Close() })
	handlerTemplate, err := http.Get("https://raw.githubusercontent.com/operator-framework/operator-sdk/master/example/memcached-operator/handler.go.tmpl")
	if err != nil {
		t.Fatal(err)
	}
	ctx.AddFinalizerFn(func() error { return handlerTemplate.Body.Close() })
	_, err = io.Copy(handlerFile, handlerTemplate.Body)
	if err != nil {
		t.Fatal(err)
	}
	memcachedTypesFile, err := ioutil.ReadFile("pkg/apis/cache/v1alpha1/types.go")
	if err != nil {
		t.Fatal(err)
	}
	memcachedTypesFileLines := bytes.Split(memcachedTypesFile, []byte("\n"))
	memcachedTypesFileLines = memcachedTypesFileLines[:len(memcachedTypesFileLines)-typesFileTrimAmount]
	memcachedTypesFileLines = append(memcachedTypesFileLines, []byte("type MemcachedSpec struct {	Size int32 `json:\"size\"`}"))
	memcachedTypesFileLines = append(memcachedTypesFileLines, []byte("type MemcachedStatus struct {Nodes []string `json:\"nodes\"`}\n"))
	os.Remove("pkg/apis/cache/v1alpha1/types.go")
	err = ioutil.WriteFile("pkg/apis/cache/v1alpha1/types.go", bytes.Join(memcachedTypesFileLines, []byte("\n")), os.FileMode(filemode))
	if err != nil {
		t.Fatal(err)
	}

	t.Log("Generating k8s")
	cmdOut, err = exec.Command("operator-sdk", "generate", "k8s").CombinedOutput()
	if err != nil {
		t.Fatalf("error: %v\nCommand Output: %s\n", err, string(cmdOut))
	}

	// create crd
	crdYAML, err := ioutil.ReadFile("deploy/crd.yaml")
	err = ctx.CreateFromYAML(crdYAML)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("Created crd")

	// run both subtests
	t.Run("memcached-group", func(t *testing.T) {
		t.Run("Cluster", MemcachedCluster)
		t.Run("Local", MemcachedLocal)
	})
}

func memcachedScaleTest(t *testing.T, f *framework.Framework, ctx framework.TestCtx) error {
	// create example-memcached yaml file
	err := ioutil.WriteFile("deploy/cr.yaml",
		[]byte("apiVersion: \"cache.example.com/v1alpha1\"\nkind: \"Memcached\"\nmetadata:\n  name: \"example-memcached\"\nspec:\n  size: 3"),
		os.FileMode(filemode))
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
	local := *f.ImageName == ""
	if local {
		*f.ImageName = "quay.io/example/memcached-operator:v0.0.1"
	}
	t.Log("Building operator docker image")
	cmdOut, err := exec.Command("operator-sdk", "build", *f.ImageName).CombinedOutput()
	if err != nil {
		t.Fatalf("error: %v\nCommand Output: %s\n", err, string(cmdOut))
	}

	operatorYAML, err := ioutil.ReadFile("deploy/operator.yaml")
	operatorYAML = bytes.Replace(operatorYAML, []byte("REPLACE_IMAGE"), []byte(*f.ImageName), 1)

	if local {
		if err != nil {
			t.Fatal(err)
		}
		operatorYAML = bytes.Replace(operatorYAML, []byte("imagePullPolicy: Always"), []byte("imagePullPolicy: Never"), 1)
		err = ioutil.WriteFile("deploy/operator.yaml", operatorYAML, os.FileMode(filemode))
		if err != nil {
			t.Fatal(err)
		}
	} else {
		t.Log("Pushing docker image to repo")
		cmdOut, err = exec.Command("docker", "push", *f.ImageName).CombinedOutput()
		if err != nil {
			t.Fatalf("error: %v\nCommand Output: %s\n", err, string(cmdOut))
		}
	}

	// create rbac
	rbacYAML, err := ioutil.ReadFile("deploy/rbac.yaml")
	rbacYAMLSplit := bytes.Split(rbacYAML, []byte("\n---\n"))
	for _, rbacSpec := range rbacYAMLSplit {
		err = ctx.CreateFromYAML(rbacSpec)
		if err != nil {
			t.Fatal(err)
		}
	}
	t.Log("Created rbac")

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
