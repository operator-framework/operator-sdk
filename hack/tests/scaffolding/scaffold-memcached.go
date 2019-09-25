// Copyright 2019 The Operator-SDK Authors
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

package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/operator-framework/operator-sdk/internal/util/fileutil"
	"github.com/operator-framework/operator-sdk/internal/util/k8sutil"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
)

// TODO: Migrate most/all of the cli commands to the bash script instead of keeping them here

const (
	sdkRepo      = "github.com/operator-framework/operator-sdk"
	operatorName = "memcached-operator"
	testRepo     = "github.com/example-inc/" + operatorName
)

func main() {
	localRepo := flag.String("local-repo", "", "Path to local SDK repository being tested. Only use when running e2e tests locally")
	imageName := flag.String("image-name", "", "Name of image being used for tests")
	noPull := flag.Bool("local-image", false, "Disable pulling images as image is local")
	flag.Parse()
	// get global framework variables
	sdkTestE2EDir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(sdkTestE2EDir); err != nil {
			log.Errorf("Failed to change back to original working directory: (%v)", err)
		}
	}()
	localSDKPath := *localRepo
	if localSDKPath == "" {
		localSDKPath = sdkTestE2EDir
	}
	// For go commands in operator projects.
	if err = os.Setenv("GO111MODULE", "on"); err != nil {
		log.Fatal(err)
	}

	log.Print("Creating new operator project")
	cmdOut, err := exec.Command("operator-sdk",
		"new",
		operatorName,
		"--repo", testRepo,
		"--skip-validation").CombinedOutput()
	if err != nil {
		log.Fatalf("Error: %v\nCommand Output: %s\n", err, string(cmdOut))
	}

	if err := os.Chdir(operatorName); err != nil {
		log.Fatalf("Failed to change to %s directory: (%v)", operatorName, err)
	}

	// Always use the local SDK path in the go.mod "replace" line.
	modBytes, err := insertGoModReplaceDir(sdkRepo, localSDKPath)
	if err != nil {
		log.Fatalf("Failed to insert go.mod replace: %v", err)
	}
	log.Printf("go.mod: %v", string(modBytes))
	cmdOut, err = exec.Command("go", "build", "./...").CombinedOutput()
	if err != nil {
		log.Fatalf("Command \"go build ./...\" failed after modifying go.mod: %v\nCommand Output:\n%v", err, string(cmdOut))
	}

	// Set replicas to 2 to test leader election. In production, this should
	// almost always be set to 1, because there isn't generally value in having
	// a hot spare operator process.
	opYaml, err := ioutil.ReadFile("deploy/operator.yaml")
	if err != nil {
		log.Fatalf("Could not read deploy/operator.yaml: %v", err)
	}
	newOpYaml := bytes.Replace(opYaml, []byte("replicas: 1"), []byte("replicas: 2"), 1)
	err = ioutil.WriteFile("deploy/operator.yaml", newOpYaml, 0644)
	if err != nil {
		log.Fatalf("Could not write deploy/operator.yaml: %v", err)
	}

	cmd := exec.Command("operator-sdk",
		"add",
		"api",
		"--api-version=cache.example.com/v1alpha1",
		"--kind=Memcached")
	cmd.Env = os.Environ()
	cmdOut, err = cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("Error: %v\nCommand Output: %s\n", err, string(cmdOut))
	}
	cmdOut, err = exec.Command("operator-sdk",
		"add",
		"controller",
		"--api-version=cache.example.com/v1alpha1",
		"--kind=Memcached").CombinedOutput()
	if err != nil {
		log.Fatalf("Error: %v\nCommand Output: %s\n", err, string(cmdOut))
	}

	tmplFiles := map[string]string{
		filepath.Join(localSDKPath, "example/memcached-operator/memcached_controller.go.tmpl"): "pkg/controller/memcached/memcached_controller.go",
		filepath.Join(localSDKPath, "test/e2e/_incluster-test-code/main_test.go"):              "test/e2e/main_test.go",
		filepath.Join(localSDKPath, "test/e2e/_incluster-test-code/memcached_test.go"):         "test/e2e/memcached_test.go",
	}
	for src, dst := range tmplFiles {
		if err := os.MkdirAll(filepath.Dir(dst), fileutil.DefaultDirFileMode); err != nil {
			log.Fatalf("Could not create template destination directory: %s", err)
		}
		cmdOut, err = exec.Command("cp", src, dst).CombinedOutput()
		if err != nil {
			log.Fatalf("Error: %v\nCommand Output: %s\n", err, string(cmdOut))
		}
	}

	memcachedTypesFile, err := ioutil.ReadFile("pkg/apis/cache/v1alpha1/memcached_types.go")
	if err != nil {
		log.Fatalf("Could not read pkg/apis/cache/v1alpha1/memcached_types.go: %v", err)
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
		log.Fatalf("Failed to remove old memcached_type.go file: (%v)", err)
	}
	err = ioutil.WriteFile("pkg/apis/cache/v1alpha1/memcached_types.go", bytes.Join(memcachedTypesFileLines, []byte("\n")), fileutil.DefaultFileMode)
	if err != nil {
		log.Fatalf("Could not write to pkg/apis/cache/v1alpha1/memcached_types.go: %v", err)
	}

	log.Print("Generating k8s")
	cmdOut, err = exec.Command("operator-sdk", "generate", "k8s").CombinedOutput()
	if err != nil {
		log.Fatalf("Error: %v\nCommand Output: %s\n", err, string(cmdOut))
	}

	log.Print("Generating openapi")
	cmdOut, err = exec.Command("operator-sdk", "generate", "openapi").CombinedOutput()
	if err != nil {
		log.Fatalf("Error: %v\nCommand Output: %s\n", err, string(cmdOut))
	}

	// TODO(haseeb): Remove this when this test no longer runs on a k8s v1.11 cluster in CI
	removeTypeFromCRDValidation()

	log.Print("Pulling new dependencies with go mod")
	cmdOut, err = exec.Command("go", "build", "./...").CombinedOutput()
	if err != nil {
		log.Fatalf("Command \"go build ./...\" failed: %v\nCommand Output:\n%v", err, string(cmdOut))
	}

	operatorYAML, err := ioutil.ReadFile("deploy/operator.yaml")
	if err != nil {
		log.Fatalf("Could not read deploy/operator.yaml: %v", err)
	}
	if *imageName == "" {
		*imageName = "quay.io/example/memcached-operator:v0.0.1"
	}
	if *noPull {
		operatorYAML = bytes.Replace(operatorYAML, []byte("imagePullPolicy: Always"), []byte("imagePullPolicy: Never"), 1)
	}
	operatorYAML = bytes.Replace(operatorYAML, []byte("REPLACE_IMAGE"), []byte(*imageName), 1)
	err = ioutil.WriteFile("deploy/operator.yaml", operatorYAML, fileutil.DefaultFileMode)
	if err != nil {
		log.Fatalf("Failed to write to deploy/operator.yaml: %v", err)
	}
}

// TODO(haseeb): Remove this when this test no longer runs on a k8s v1.11 cluster in CI
// removeTypeFromCRDValidation will edit the memcached CRD manifest to remove
// the "type: object" field from root of the CRD validation block.
// This serves as a workaround for the following bug on a k8s 1.11 cluster:
// https://github.com/kubernetes/kubernetes/issues/65293
func removeTypeFromCRDValidation() {
	crdPath := "deploy/cache.example.com_memcachedrs_crd.yaml"
	b, err := ioutil.ReadFile(crdPath)
	if err != nil {
		log.Fatalf("Failed to read CRD manifest %s: %v", crdPath, err)
	}

	crd := &apiextv1beta1.CustomResourceDefinition{}
	if err = yaml.Unmarshal(b, crd); err != nil {
		log.Fatalf("Failed to unmarshal CRD: %v", err)
	}

	crd.Spec.Validation.OpenAPIV3Schema.Type = ""

	crdYaml, err := k8sutil.GetObjectBytes(crd, yaml.Marshal)
	if err != nil {
		log.Fatalf("Failed to marshal CRD: %v", err)
	}

	err = ioutil.WriteFile(crdPath, crdYaml, fileutil.DefaultFileMode)
	if err != nil {
		log.Fatalf("Failed to write out CRD manifest %s: %v", crdPath, err)
	}
}

func insertGoModReplaceDir(repo, path string) ([]byte, error) {
	modBytes, err := ioutil.ReadFile("go.mod")
	if err != nil {
		return nil, errors.Wrap(err, "failed to read go.mod")
	}
	// Remove all replace lines in go.mod.
	replaceRe := regexp.MustCompile(fmt.Sprintf("(replace )?%s =>.+", repo))
	modBytes = replaceRe.ReplaceAll(modBytes, nil)
	// Append the desired replace to the end of go.mod's bytes.
	sdkReplace := fmt.Sprintf("replace %s => %s", repo, path)
	modBytes = append(modBytes, []byte("\n"+sdkReplace)...)
	err = ioutil.WriteFile("go.mod", modBytes, fileutil.DefaultFileMode)
	if err != nil {
		return nil, errors.Wrap(err, "failed to write go.mod before replacing SDK repo")
	}
	return modBytes, nil
}
