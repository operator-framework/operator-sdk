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
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/operator-framework/operator-sdk/internal/generate/clusterserviceversion/bases"
	"github.com/operator-framework/operator-sdk/internal/util/fileutil"
	"github.com/operator-framework/operator-sdk/internal/util/k8sutil"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"

	"github.com/blang/semver"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/yaml"
)

const (
	testProjectName = "memcached-operator"

	// Dir names/CSV versions
	csvVersion      = "0.0.3"
	fromVersion     = "0.0.2"
	notExistVersion = "1.0.0"
	noUpdateDir     = "noupdate"
)

var (
	testGoDataDir                = filepath.Join("..", "testdata", "go")
	testNonStandardLayoutDataDir = filepath.Join("..", "testdata", "non-standard-layout")
)

func chDirWithCleanup(t *testing.T, dataDir string) func() {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dataDir); err != nil {
		t.Fatal(err)
	}
	chDirCleanupFunc := func() {
		if err := os.Chdir(wd); err != nil {
			t.Fatal(err)
		}
	}
	return chDirCleanupFunc
}

func mkTempDirWithCleanup(t *testing.T, prefix string) (dir string, f func()) {
	var err error
	if dir, err = ioutil.TempDir("", prefix); err != nil {
		t.Fatalf("Failed to create tmp dir: %v", err)
	}
	f = func() {
		if err := os.RemoveAll(dir); err != nil {
			// Not a test failure since files in /tmp will eventually get deleted
			t.Logf("Failed to remove tmp dir %s: %v", dir, err)
		}
	}
	return
}

func readFile(t *testing.T, path string) []byte {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read testdata file: %v", err)
	}
	return b
}

// TODO: Change to table driven subtests to test out different Inputs/Output for the generator
func TestGoCSVNewWithInputsToOutput(t *testing.T) {
	// Change directory to project root so the test cases can form the correct pkg imports
	cleanupFunc := chDirWithCleanup(t, testNonStandardLayoutDataDir)
	defer cleanupFunc()

	// Temporary output dir for generating catalog bundle
	outputDir, rmDirFunc := mkTempDirWithCleanup(t, t.Name()+"-output-catalog")
	defer rmDirFunc()

	csvVersion := "0.0.1"
	g := BundleGenerator{
		OperatorName:          testProjectName,
		DeployDir:             "config",
		ApisDir:               "api",
		CRDsDir:               filepath.Join("config", "crds"),
		OutputDir:             outputDir,
		CSVVersion:            csvVersion,
		FromVersion:           "",
		UpdateCRDs:            false,
		MakeManifests:         false,
		InteractivePreference: projutil.InteractiveHardOff,
	}

	g.noUpdate = true

	if err := g.Generate(); err != nil {
		t.Fatalf("Failed to execute CSV generator: %v", err)
	}

	csvFileName := getCSVFileNameLegacy(testProjectName, csvVersion)

	// Read expected CSV
	expBundleDir := filepath.Join("expected-catalog", OLMCatalogChildDir, testProjectName, csvVersion)
	csvExp := string(readFile(t, filepath.Join(expBundleDir, csvFileName)))

	// Read generated CSV from outputDir path
	outputBundleDir := filepath.Join(outputDir, OLMCatalogChildDir, testProjectName, csvVersion)
	csvOutput := string(readFile(t, filepath.Join(outputBundleDir, csvFileName)))

	assert.Equal(t, csvExp, csvOutput)
}

func TestGoCSVUpgradeWithInputsToOutput(t *testing.T) {
	// Change directory to project root so the test cases can form the correct pkg imports
	cleanupFunc := chDirWithCleanup(t, testNonStandardLayoutDataDir)
	defer cleanupFunc()

	// Temporary output dir for generating catalog bundle
	outputDir, rmDirFunc := mkTempDirWithCleanup(t, t.Name()+"-output-catalog")
	defer rmDirFunc()

	fromVersion := "0.0.3"
	csvVersion := "0.0.4"

	// Copy over expected fromVersion CSV bundle directory to the output dir
	// so the test can upgrade from it
	outputFromCSVDir := filepath.Join(outputDir, OLMCatalogChildDir, testProjectName)
	if err := os.MkdirAll(outputFromCSVDir, fileutil.DefaultDirFileMode); err != nil {
		t.Fatalf("Failed to create CSV bundle dir (%s) for fromVersion (%s): %v", outputFromCSVDir, fromVersion, err)
	}
	expCatalogDir := filepath.Join("expected-catalog", OLMCatalogChildDir)
	expFromCSVDir := filepath.Join(expCatalogDir, testProjectName, fromVersion)
	cmd := exec.Command("cp", "-r", expFromCSVDir, outputFromCSVDir)
	t.Logf("Copying expected fromVersion CSV manifest dir %#v", cmd.Args)
	if err := projutil.ExecCmd(cmd); err != nil {
		t.Fatalf("Failed to copy expected CSV bundle dir (%s) to output dir (%s): %v", expFromCSVDir, outputFromCSVDir, err)
	}

	g := BundleGenerator{
		OperatorName:          testProjectName,
		DeployDir:             "config",
		ApisDir:               "api",
		CRDsDir:               filepath.Join("config", "crds"),
		OutputDir:             outputDir,
		CSVVersion:            csvVersion,
		FromVersion:           fromVersion,
		UpdateCRDs:            false,
		MakeManifests:         false,
		InteractivePreference: projutil.InteractiveHardOff,
	}

	if err := g.Generate(); err != nil {
		t.Fatalf("Failed to execute CSV generator: %v", err)
	}
	csvFileName := getCSVFileNameLegacy(testProjectName, csvVersion)

	// Read expected CSV
	expCsvFile := filepath.Join(expCatalogDir, testProjectName, csvVersion, csvFileName)
	csvExp := string(readFile(t, expCsvFile))

	// Read generated CSV from outputDir path
	csvOutputFile := filepath.Join(outputFromCSVDir, csvVersion, csvFileName)
	csvOutput := string(readFile(t, csvOutputFile))

	assert.Equal(t, csvExp, csvOutput)
}

func TestGoCSVNew(t *testing.T) {
	cleanupFunc := chDirWithCleanup(t, testGoDataDir)
	defer cleanupFunc()

	g := BundleGenerator{
		InteractivePreference: projutil.InteractiveHardOff,
		OperatorName:          testProjectName,
		DeployDir:             "deploy",
		ApisDir:               filepath.Join("pkg", "apis"),
		CRDsDir:               filepath.Join("deploy", "crds_v1beta1"),
		OutputDir:             "deploy",
		CSVVersion:            csvVersion,
		FromVersion:           "",
		UpdateCRDs:            false,
		MakeManifests:         false,
	}
	g.noUpdate = true
	g.setDefaults()
	fileMap, err := g.generateCSV()
	if err != nil {
		t.Fatalf("Failed to execute CSV generator: %v", err)
	}

	csvExpFile := getCSVFileNameLegacy(testProjectName, csvVersion)
	csvExpBytes := readFile(t, filepath.Join(OLMCatalogDir, testProjectName, noUpdateDir, csvExpFile))
	if b, ok := fileMap[csvExpFile]; !ok {
		t.Errorf("Failed to generate CSV for version %s", csvVersion)
	} else {
		assert.Equal(t, string(csvExpBytes), string(b))
	}
}

func TestGoCSVUpdate(t *testing.T) {
	cleanupFunc := chDirWithCleanup(t, testGoDataDir)
	defer cleanupFunc()

	g := BundleGenerator{
		InteractivePreference: projutil.InteractiveHardOff,
		OperatorName:          testProjectName,
		DeployDir:             "deploy",
		ApisDir:               filepath.Join("pkg", "apis"),
		CRDsDir:               filepath.Join("deploy", "crds_v1beta1"),
		OutputDir:             "deploy",
		CSVVersion:            csvVersion,
		FromVersion:           "",
		UpdateCRDs:            false,
		MakeManifests:         false,
	}
	g.setDefaults()
	fileMap, err := g.generateCSV()
	if err != nil {
		t.Fatalf("Failed to execute CSV generator: %v", err)
	}

	csvExpFile := getCSVFileNameLegacy(testProjectName, csvVersion)
	csvExpBytes := readFile(t, filepath.Join(OLMCatalogDir, testProjectName, csvVersion, csvExpFile))
	if b, ok := fileMap[csvExpFile]; !ok {
		t.Errorf("Failed to generate CSV for version %s", csvVersion)
	} else {
		assert.Equal(t, string(csvExpBytes), string(b))
	}
}

func TestGoCSVUpgrade(t *testing.T) {
	cleanupFunc := chDirWithCleanup(t, testGoDataDir)
	defer cleanupFunc()

	g := BundleGenerator{
		InteractivePreference: projutil.InteractiveHardOff,
		OperatorName:          testProjectName,
		DeployDir:             "deploy",
		ApisDir:               filepath.Join("pkg", "apis"),
		CRDsDir:               filepath.Join("deploy", "crds_v1beta1"),
		OutputDir:             "deploy",
		CSVVersion:            csvVersion,
		FromVersion:           fromVersion,
		UpdateCRDs:            false,
		MakeManifests:         false,
	}
	g.setDefaults()
	fileMap, err := g.generateCSV()
	if err != nil {
		t.Fatalf("Failed to execute CSV generator: %v", err)
	}

	csvExpFile := getCSVFileNameLegacy(testProjectName, csvVersion)
	csvExpBytes := readFile(t, filepath.Join(OLMCatalogDir, testProjectName, csvVersion, csvExpFile))
	if b, ok := fileMap[csvExpFile]; !ok {
		t.Errorf("Failed to generate CSV for version %s", csvVersion)
	} else {
		assert.Equal(t, string(csvExpBytes), string(b))
	}
}

func TestGoCSVNewManifests(t *testing.T) {
	cleanupFunc := chDirWithCleanup(t, testGoDataDir)
	defer cleanupFunc()

	g := BundleGenerator{
		InteractivePreference: projutil.InteractiveHardOff,
		OperatorName:          testProjectName,
		DeployDir:             "deploy",
		ApisDir:               filepath.Join("pkg", "apis"),
		CRDsDir:               filepath.Join("deploy", "crds_v1beta1"),
		OutputDir:             "deploy",
		CSVVersion:            csvVersion,
		FromVersion:           "",
		UpdateCRDs:            false,
		MakeManifests:         true,
	}
	g.noUpdate = true
	g.setDefaults()
	fileMap, err := g.generateCSV()
	if err != nil {
		t.Fatalf("Failed to execute CSV generator: %v", err)
	}

	csvExpFile := getCSVFileNameLegacy(testProjectName, csvVersion)
	csvExpBytes := readFile(t, filepath.Join(OLMCatalogDir, testProjectName, noUpdateDir, csvExpFile))
	if b, ok := fileMap[getCSVFileName(testProjectName)]; !ok {
		t.Errorf("Failed to generate CSV for version %s", csvVersion)
	} else {
		assert.Equal(t, string(csvExpBytes), string(b))
	}
}

func TestGoCSVUpdateManifests(t *testing.T) {
	cleanupFunc := chDirWithCleanup(t, testGoDataDir)
	defer cleanupFunc()

	g := BundleGenerator{
		InteractivePreference: projutil.InteractiveHardOff,
		OperatorName:          testProjectName,
		DeployDir:             "deploy",
		ApisDir:               filepath.Join("pkg", "apis"),
		CRDsDir:               filepath.Join("deploy", "crds_v1beta1"),
		OutputDir:             "deploy",
		CSVVersion:            csvVersion,
		FromVersion:           "",
		UpdateCRDs:            false,
		MakeManifests:         true,
	}
	g.setDefaults()
	fileMap, err := g.generateCSV()
	if err != nil {
		t.Fatalf("Failed to execute CSV generator: %v", err)
	}

	csvExpFile := getCSVFileNameLegacy(testProjectName, csvVersion)
	csvExpBytes := readFile(t, filepath.Join(OLMCatalogDir, testProjectName, csvVersion, csvExpFile))
	if b, ok := fileMap[getCSVFileName(testProjectName)]; !ok {
		t.Errorf("Failed to generate CSV for version %s", csvVersion)
	} else {
		assert.Equal(t, string(csvExpBytes), string(b))
	}
}

func TestGoCSVNewWithInvalidDeployDir(t *testing.T) {
	cleanupFunc := chDirWithCleanup(t, testGoDataDir)
	defer cleanupFunc()

	g := BundleGenerator{
		InteractivePreference: projutil.InteractiveHardOff,
		OperatorName:          testProjectName,
		DeployDir:             "notExist",
		ApisDir:               filepath.Join("pkg", "apis"),
		CRDsDir:               "notExist",
		OutputDir:             "deploy",
		CSVVersion:            notExistVersion,
		FromVersion:           "",
		UpdateCRDs:            false,
		MakeManifests:         false,
	}

	g.setDefaults()
	_, err := g.generateCSV()
	if err == nil {
		t.Fatalf("Failed to get error for running CSV generator"+
			"on non-existent manifests directory: %s", g.DeployDir)
	}
}

func TestGoCSVNewWithEmptyDeployDir(t *testing.T) {
	cleanupFunc := chDirWithCleanup(t, testGoDataDir)
	defer cleanupFunc()

	g := BundleGenerator{
		InteractivePreference: projutil.InteractiveHardOff,
		OperatorName:          testProjectName,
		DeployDir:             "emptydir",
		ApisDir:               filepath.Join("pkg", "apis"),
		CRDsDir:               "emptydir",
		OutputDir:             "emptydir",
		CSVVersion:            notExistVersion,
		FromVersion:           "",
		UpdateCRDs:            false,
		MakeManifests:         false,
	}

	g.setDefaults()
	fileMap, err := g.generateCSV()
	if err != nil {
		t.Fatalf("Failed to execute CSV generator: %v", err)
	}

	// Create an empty CSV.
	b := bases.ClusterServiceVersion{
		OperatorName: testProjectName,
		OperatorType: projutil.OperatorTypeGo,
	}
	csv, err := b.GetBase()
	if err != nil {
		t.Fatal(err)
	}
	if err := g.updateCSVVersions(csv); err != nil {
		t.Fatal(err)
	}
	csv.Spec.InstallStrategy.StrategyName = v1alpha1.InstallStrategyNameDeployment
	csv.Spec.InstallStrategy.StrategySpec.DeploymentSpecs = []v1alpha1.StrategyDeploymentSpec{}

	csvExpBytes, err := k8sutil.GetObjectBytes(csv, yaml.Marshal)
	if err != nil {
		t.Fatal(err)
	}
	csvExpFile := getCSVFileNameLegacy(testProjectName, notExistVersion)
	if b, ok := fileMap[csvExpFile]; !ok {
		t.Errorf("Failed to generate CSV for version %s", notExistVersion)
	} else {
		assert.Equal(t, string(csvExpBytes), string(b))
	}
}

func TestUpdateCSVVersion(t *testing.T) {
	cleanupFunc := chDirWithCleanup(t, testGoDataDir)
	defer cleanupFunc()

	csv, err := getCSVFromDir(filepath.Join(OLMCatalogDir, testProjectName, fromVersion))
	if err != nil {
		t.Fatal("Failed to get new CSV")
	}

	g := BundleGenerator{
		InteractivePreference: projutil.InteractiveHardOff,
		OperatorName:          testProjectName,
		DeployDir:             "deploy",
		ApisDir:               filepath.Join("pkg", "apis"),
		CRDsDir:               filepath.Join("deploy", "crds_v1beta1"),
		OutputDir:             "deploy",
		CSVVersion:            csvVersion,
		FromVersion:           fromVersion,
		UpdateCRDs:            false,
		MakeManifests:         false,
	}
	g.setDefaults()
	if err := g.updateCSVVersions(csv); err != nil {
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
