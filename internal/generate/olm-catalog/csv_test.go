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

package olmcatalog

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/operator-framework/operator-sdk/internal/generate/gen"
	"github.com/operator-framework/operator-sdk/internal/scaffold"
	internalk8sutil "github.com/operator-framework/operator-sdk/internal/util/k8sutil"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"

	"github.com/blang/semver"
	"github.com/ghodss/yaml"
	olminstall "github.com/operator-framework/operator-lifecycle-manager/pkg/controller/install"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
)

const (
	testProjectName  = "memcached-operator"
	csvVersion       = "0.0.3"
	fromVersion      = "0.0.2"
	notExistVersion  = "1.0.0"
	scratchBundleDir = "scratch"
	testGroup        = "cache.example.com"
	testKind1        = "Memcached"
	testVersion1     = "v1alpha1"
)

var (
	testDataDir     = filepath.Join("..", "testdata")
	testGoDataDir   = filepath.Join(testDataDir, "go")
	testHelmDataDir = filepath.Join(testDataDir, "helm")
)

func setupTestEnvWithCleanup(t *testing.T, dataDir string) (cleanupFuncs []func()) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dataDir); err != nil {
		t.Fatal(err)
	}
	cleanupFuncs = append(cleanupFuncs, func() {
		if err := os.Chdir(wd); err != nil {
			t.Fatal(err)
		}
	})
	return cleanupFuncs
}

func TestGoCSVGoNew(t *testing.T) {
	for _, cleanupFunc := range setupTestEnvWithCleanup(t, testGoDataDir) {
		defer cleanupFunc()
	}

	cfg := gen.Config{
		OperatorName: testProjectName,
		Filters:      gen.MakeFilters(scaffold.DeployDir),
	}
	g := NewCSV(cfg, csvVersion, "")
	fileMap, err := g.(csvGenerator).generate()
	if err != nil {
		t.Fatalf("Failed to execute CSV generator: %v", err)
	}

	csvExpFile := getCSVFileName(testProjectName, csvVersion)
	csvExpBytes, err := ioutil.ReadFile(filepath.Join(OLMCatalogDir, testProjectName, csvVersion, csvExpFile))
	if err != nil {
		t.Fatalf("Failed to read expected CSV file: %v", err)
	}
	csvExp := string(csvExpBytes)
	// Replace image tag, which is retrieved from the deployment and is
	// different than that in the expected CSV, but doesn't matter for this test.
	csvExp = strings.Replace(csvExp,
		"image: quay.io/example/memcached-operator:v0.0.2",
		"image: quay.io/example/memcached-operator:v0.0.3",
		-1)
	if b, ok := fileMap[csvExpFile]; !ok {
		t.Errorf("Failed to generate CSV for version %s", csvVersion)
	} else {
		assert.Equal(t, csvExp, string(b))
	}
}

func TestGoCSVFromOld(t *testing.T) {
	for _, cleanupFunc := range setupTestEnvWithCleanup(t, testGoDataDir) {
		defer cleanupFunc()
	}

	cfg := gen.Config{
		OperatorName: testProjectName,
		Filters:      gen.MakeFilters(scaffold.DeployDir),
	}
	g := NewCSV(cfg, csvVersion, fromVersion)
	fileMap, err := g.(csvGenerator).generate()
	if err != nil {
		t.Fatalf("Failed to execute CSV generator: %v", err)
	}

	csvExpFile := getCSVFileName(testProjectName, csvVersion)
	csvExpBytes, err := ioutil.ReadFile(filepath.Join(OLMCatalogDir, testProjectName, csvVersion, csvExpFile))
	if err != nil {
		t.Fatalf("Failed to read expected CSV file: %v", err)
	}
	csvExp := string(csvExpBytes)
	if b, ok := fileMap[csvExpFile]; !ok {
		t.Errorf("Failed to generate CSV for version %s", csvVersion)
	} else {
		assert.Equal(t, csvExp, string(b))
	}
}

func TestGoCSVIncludeAll(t *testing.T) {
	cfg := gen.Config{OperatorName: testProjectName}
	g := NewCSV(cfg, notExistVersion, "")
	fileMap, err := g.(csvGenerator).generate()
	if err != nil {
		t.Fatalf("Failed to execute CSV generator: %v", err)
	}

	// Create an empty CSV.
	csv, err := newCSV(testProjectName, notExistVersion)
	if err != nil {
		t.Fatal(err)
	}
	csvExpBytes, err := internalk8sutil.GetObjectBytes(csv, yaml.Marshal)
	if err != nil {
		t.Fatal(err)
	}
	csvExpFile := getCSVFileName(testProjectName, notExistVersion)
	if b, ok := fileMap[csvExpFile]; !ok {
		t.Errorf("Failed to generate CSV for version %s", notExistVersion)
	} else {
		assert.Equal(t, string(csvExpBytes), string(b))
	}
}

func getTestCRDFile(g, k string) string {
	return fmt.Sprintf("%s_%s_crd.yaml", g, strings.ToLower(k)+"s")
}

func getTestCRFile(g, v, k string) string {
	return fmt.Sprintf("%s_%s_%s_cr.yaml", g, v, strings.ToLower(k))
}

func TestHelmCSVNew(t *testing.T) {
	for _, cleanupFunc := range setupTestEnvWithCleanup(t, testHelmDataDir) {
		defer cleanupFunc()
	}

	cfg := gen.Config{
		OperatorName: testProjectName,
		// Only include one CRD and all deploy manifests in this run.
		Filters: gen.MakeFilters(
			filepath.Join(scaffold.CRDsDir, getTestCRDFile(testGroup, testKind1)),
			filepath.Join(scaffold.CRDsDir, getTestCRFile(testGroup, testVersion1, testKind1)),
			filepath.Join(scaffold.DeployDir, "operator.yaml"),
			filepath.Join(scaffold.DeployDir, "role_binding.yaml"),
			filepath.Join(scaffold.DeployDir, "role.yaml"),
			filepath.Join(scaffold.DeployDir, "service_account.yaml"),
		),
	}
	// Create new CSV from scratch and compare against an existing CSV.
	g := NewCSV(cfg, notExistVersion, "")
	fileMap, err := g.(csvGenerator).generate()
	if err != nil {
		t.Fatalf("Failed to execute CSV generator: %v", err)
	}
	// Get an existing CSV created from scratch.
	csvExpFile := getCSVFileName(testProjectName, notExistVersion)
	csvExpBytes, err := ioutil.ReadFile(filepath.Join(OLMCatalogDir, testProjectName, scratchBundleDir, csvExpFile))
	if err != nil {
		t.Fatalf("Failed to read expected CSV file: %v", err)
	}
	// Compare scratch CSV to existing CSV.
	csvExp := string(csvExpBytes)
	genCSVFile := getCSVFileName(testProjectName, notExistVersion)
	if b, ok := fileMap[genCSVFile]; !ok {
		t.Errorf("Failed to generate CSV for version %s: file %s not found", notExistVersion, genCSVFile)
	} else {
		assert.Equal(t, csvExp, string(b))
	}

	// Include all CRDs.
	cfg.Filters = gen.MakeFilters(scaffold.DeployDir)
	g = NewCSV(cfg, csvVersion, "")
	fileMap, err = g.(csvGenerator).generate()
	if err != nil {
		t.Fatalf("Failed to execute CSV generator: %v", err)
	}
	csvExpFile = getCSVFileName(testProjectName, csvVersion)
	csvExpBytes, err = ioutil.ReadFile(filepath.Join(OLMCatalogDir, testProjectName, csvVersion, csvExpFile))
	if err != nil {
		t.Fatalf("Failed to read expected CSV file: %v", err)
	}
	csvExp = string(csvExpBytes)
	if b, ok := fileMap[csvExpFile]; !ok {
		t.Errorf("Failed to generate CSV for version %s", csvVersion)
	} else {
		assert.Equal(t, csvExp, string(b))
	}
}

func TestUpdateVersion(t *testing.T) {
	csv, err := getCSVFromDir(filepath.Join(testGoDataDir, OLMCatalogDir, testProjectName, fromVersion))
	if err != nil {
		t.Fatal("Failed to get new CSV")
	}

	cfg := gen.Config{OperatorName: testProjectName}
	g := NewCSV(cfg, csvVersion, fromVersion)
	if err := g.(csvGenerator).updateCSVVersions(csv); err != nil {
		t.Fatalf("Failed to update csv with version %s: (%v)", csvVersion, err)
	}

	wantedSemver, err := semver.Parse(csvVersion)
	if err != nil {
		t.Errorf("Failed to parse %s: %v", csvVersion, err)
	}
	if !csv.Spec.Version.Equals(wantedSemver) {
		t.Errorf("Wanted csv version %v, got %v", wantedSemver, csv.Spec.Version)
	}
	wantedName := getCSVName(testProjectName, csvVersion)
	if csv.ObjectMeta.Name != wantedName {
		t.Errorf("Wanted csv name %s, got %s", wantedName, csv.ObjectMeta.Name)
	}

	var resolver *olminstall.StrategyResolver
	strategyInterface, err := resolver.UnmarshalStrategy(csv.Spec.InstallStrategy)
	if err != nil {
		t.Fatal(err)
	}
	strategy, ok := strategyInterface.(*olminstall.StrategyDetailsDeployment)
	if !ok {
		t.Fatalf("Strategy of type %T was not StrategyDetailsDeployment", strategyInterface)
	}
	csvPodImage := strategy.DeploymentSpecs[0].Spec.Template.Spec.Containers[0].Image
	// updateCSVVersions should not update podspec image.
	wantedImage := "quay.io/example/memcached-operator:v0.0.2"
	if csvPodImage != wantedImage {
		t.Errorf("Podspec image changed from %s to %s", wantedImage, csvPodImage)
	}

	wantedReplaces := getCSVName(testProjectName, fromVersion)
	if csv.Spec.Replaces != wantedReplaces {
		t.Errorf("Wanted csv replaces %s, got %s", wantedReplaces, csv.Spec.Replaces)
	}
}

func TestSetAndCheckOLMNamespaces(t *testing.T) {
	depBytes, err := ioutil.ReadFile(filepath.Join(testGoDataDir, scaffold.DeployDir, "operator.yaml"))
	if err != nil {
		t.Fatalf("Failed to read Deployment bytes: %v", err)
	}

	// The test operator.yaml doesn't have "olm.targetNamespaces", so first
	// check that depHasOLMNamespaces() returns false.
	dep := appsv1.Deployment{}
	if err := yaml.Unmarshal(depBytes, &dep); err != nil {
		t.Fatalf("Failed to unmarshal Deployment bytes: %v", err)
	}
	if depHasOLMNamespaces(dep) {
		t.Error("Expected depHasOLMNamespaces to return false, got true")
	}

	// Insert "olm.targetNamespaces" into WATCH_NAMESPACE and check that
	// depHasOLMNamespaces() returns true.
	setWatchNamespacesEnv(&dep)
	if !depHasOLMNamespaces(dep) {
		t.Error("Expected depHasOLMNamespaces to return true, got false")
	}

	// Overwrite WATCH_NAMESPACE and check that depHasOLMNamespaces() returns
	// false.
	overwriteContainerEnvVar(&dep, k8sutil.WatchNamespaceEnvVar, newEnvVar("FOO", "bar"))
	if depHasOLMNamespaces(dep) {
		t.Error("Expected depHasOLMNamespaces to return false, got true")
	}

	// Insert "olm.targetNamespaces" elsewhere in the deployment pod spec
	// and check that depHasOLMNamespaces() returns true.
	dep = appsv1.Deployment{}
	if err := yaml.Unmarshal(depBytes, &dep); err != nil {
		t.Fatalf("Failed to unmarshal Deployment bytes: %v", err)
	}
	dep.Spec.Template.ObjectMeta.Labels["namespace"] = olmTNMeta
	if !depHasOLMNamespaces(dep) {
		t.Error("Expected depHasOLMNamespaces to return true, got false")
	}
}
