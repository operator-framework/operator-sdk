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

package olmcatalog

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	gen "github.com/operator-framework/operator-sdk/internal/generate/gen"
	"github.com/operator-framework/operator-sdk/internal/scaffold"
	internalk8sutil "github.com/operator-framework/operator-sdk/internal/util/k8sutil"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"

	"github.com/blang/semver"
	"github.com/ghodss/yaml"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
)

const (
	testProjectName = "memcached-operator"
	csvVersion      = "0.0.3"
	fromVersion     = "0.0.2"
	notExistVersion = "1.0.0"
)

var (
	testDataDir   = filepath.Join("..", "testdata")
	testGoDataDir = filepath.Join(testDataDir, "go")
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

func TestGoCSVWithInputsAndOutput(t *testing.T) {
	// Move to testdata/non-standard to test on the non-standard project layout
	// TODO: Refactor to make the chdir logic more readable
	nonStandardTestDataDir := filepath.Join(testDataDir, "non-standard-layout")
	for _, cleanupFunc := range setupTestEnvWithCleanup(t, nonStandardTestDataDir) {
		defer cleanupFunc()
	}

	// Temporary output dir for generating catalog bundle
	outputDir, err := ioutil.TempDir("", t.Name()+"-output-catalog")
	if err != nil {
		log.Fatal(err)
	}
	// Clean up output catalog dir
	defer func() {
		if err := os.RemoveAll(outputDir); err != nil && !os.IsNotExist(err) {
			// Not a test failure since files in /tmp will eventually get deleted
			t.Logf("Failed to remove tmp generated catalog directory (%s): %v", outputDir, err)
		}
	}()

	cfg := gen.Config{
		OperatorName: testProjectName,
		Inputs: map[string]string{
			DeployDirKey: "config",
			APIsDirKey:   "api",
		},
		OutputDir: outputDir,
	}
	g := NewCSV(cfg, csvVersion, "")

	if err := g.Generate(); err != nil {
		t.Fatalf("Failed to execute CSV generator: %v", err)
	}

	csvFileName := getCSVFileName(testProjectName, csvVersion)

	// Read expected CSV
	expCatalogDir := filepath.Join("expected-catalog", OLMCatalogChildDir)
	csvExpBytes, err := ioutil.ReadFile(filepath.Join(expCatalogDir, testProjectName, csvVersion, csvFileName))
	if err != nil {
		t.Fatalf("Failed to read expected CSV file: %v", err)
	}
	csvExp := string(csvExpBytes)

	// Read generated CSV from OutputDir/olm-catalog
	outputCatalogDir := filepath.Join(cfg.OutputDir, OLMCatalogChildDir)
	csvOutputBytes, err := ioutil.ReadFile(filepath.Join(outputCatalogDir, testProjectName, csvVersion, csvFileName))
	if err != nil {
		t.Fatalf("Failed to read output CSV file: %v", err)
	}
	csvOutput := string(csvOutputBytes)

	assert.Equal(t, csvExp, csvOutput)
}

// TODO: This test is only updating the existing CSV
// deploy/olm-catalog/memcached-operator/0.0.3/memcached-operator.v0.0.3.clusterserviceversion.yaml
// present in testdata/go
// Fix to generate a new CSV rather than only update an existing one
func TestGoCSVFromNew(t *testing.T) {
	for _, cleanupFunc := range setupTestEnvWithCleanup(t, testGoDataDir) {
		defer cleanupFunc()
	}

	cfg := gen.Config{
		OperatorName: testProjectName,
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

	csvDepSpecs := csv.Spec.InstallStrategy.StrategySpec.DeploymentSpecs
	if len(csvDepSpecs) != 1 {
		t.Fatal("No deployment specs in CSV")
	}
	csvPodImage := csvDepSpecs[0].Spec.Template.Spec.Containers[0].Image
	if len(csvDepSpecs[0].Spec.Template.Spec.Containers) != 1 {
		t.Fatal("No containers in CSV deployment spec")
	}
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
