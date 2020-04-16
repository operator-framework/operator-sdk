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

package clusterserviceversion

import (
	"bytes"
	"io"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/blang/semver"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/yaml"

	"github.com/operator-framework/operator-sdk/internal/generate/clusterserviceversion/bases"
	"github.com/operator-framework/operator-sdk/internal/generate/collector"
	genutil "github.com/operator-framework/operator-sdk/internal/generate/internal"
	"github.com/operator-framework/operator-sdk/internal/util/k8sutil"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"
)

const (
	testProjectName = "memcached-operator"

	// Dir names/CSV versions
	newVersion      = "0.0.3"
	fromVersion     = "0.0.2"
	notExistVersion = "1.0.0"
)

var (
	testGoDataDir                = filepath.Join("..", "testdata", "go")
	testNonStandardLayoutDataDir = filepath.Join("..", "testdata", "non-standard-layout")
)

func makeNoUpdateBaseGetter(opName, apisDir string, gvks []schema.GroupVersionKind) getBaseFunc {

	b := bases.ClusterServiceVersion{
		OperatorName: opName,
		OperatorType: projutil.OperatorTypeGo,
		APIsDir:      apisDir,
		GVKs:         gvks,
	}
	return b.GetBase
}

func readFile(t *testing.T, path string) []byte {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read testdata file: %v", err)
	}
	return b
}

// TODO: Change to table driven subtests to test out different Inputs/Output for the generator
func TestGoCSVNewNewLayout(t *testing.T) {

	manifestRoot := filepath.Join(testNonStandardLayoutDataDir, "config")
	crdsDir := filepath.Join(manifestRoot, "crds")
	col := &collector.Manifests{}
	if err := col.UpdateFromDirs(manifestRoot, crdsDir); err != nil {
		t.Fatalf("Error updating collector from manifest root %s: %v", manifestRoot, err)
	}

	gvks := []schema.GroupVersionKind{
		{Group: "cache.example.com", Version: "v1alpha1", Kind: "Memcached"},
	}
	inputDir := filepath.Join(testNonStandardLayoutDataDir, "expected-catalog", OLMCatalogDir, testProjectName)
	apisDir := filepath.Join(testNonStandardLayoutDataDir, "api")
	buf := &bytes.Buffer{}
	newVersion := "0.0.1"
	g := Generator{
		OperatorName: testProjectName,
		Version:      newVersion,
		FromVersion:  "",
		Collector:    col,
		getWriter:    func() (io.Writer, error) { return buf, nil },
		getBase:      makeNoUpdateBaseGetter(testProjectName, apisDir, gvks),
	}
	err := g.GenerateLegacy()
	if err != nil {
		t.Fatalf("Failed to execute CSV generator: %v", err)
	}

	// Read expected CSV
	csvFileName := getCSVFileLegacy(g.OperatorName, g.Version)
	csvExp := string(readFile(t, filepath.Join(inputDir, csvFileName)))

	assert.Equal(t, csvExp, buf.String())
}

func TestGoCSVUpgradeNewLayout(t *testing.T) {

	manifestRoot := filepath.Join(testNonStandardLayoutDataDir, "config")
	crdsDir := filepath.Join(manifestRoot, "crds")
	col := &collector.Manifests{}
	if err := col.UpdateFromDirs(manifestRoot, crdsDir); err != nil {
		t.Fatalf("Error updating collector from manifest root %s: %v", manifestRoot, err)
	}

	fromVersion := "0.0.3"
	newVersion := "0.0.4"
	inputDir := filepath.Join(testNonStandardLayoutDataDir, "expected-catalog", OLMCatalogDir, testProjectName)
	apisDir := filepath.Join(testNonStandardLayoutDataDir, "api")
	buf := &bytes.Buffer{}
	g := Generator{
		OperatorName: testProjectName,
		Version:      newVersion,
		FromVersion:  fromVersion,
		getWriter:    func() (io.Writer, error) { return buf, nil },
	}
	err := g.GenerateLegacy(WithPackageBase(inputDir, apisDir))
	if err != nil {
		t.Fatalf("Failed to execute CSV generator: %v", err)
	}

	// Read expected CSV
	csvFileName := getCSVFileLegacy(g.OperatorName, g.Version)
	csvExp := string(readFile(t, filepath.Join(inputDir, csvFileName)))

	assert.Equal(t, csvExp, buf.String())
}

func TestGoCSVNew(t *testing.T) {

	manifestRoot := filepath.Join(testGoDataDir, "deploy")
	crdsDir := filepath.Join(manifestRoot, "crds_v1beta1")
	col := &collector.Manifests{}
	if err := col.UpdateFromDirs(manifestRoot, crdsDir); err != nil {
		t.Fatalf("Error updating collector from manifest root %s: %v", manifestRoot, err)
	}

	gvks := []schema.GroupVersionKind{
		{Group: "cache.example.com", Version: "v1alpha1", Kind: "Memcached"},
	}
	inputDir := filepath.Join(manifestRoot, OLMCatalogDir, testProjectName)
	apisDir := filepath.Join(testGoDataDir, "pkg", "apis")
	buf := &bytes.Buffer{}
	g := Generator{
		OperatorName: testProjectName,
		Version:      newVersion,
		FromVersion:  "",
		getWriter:    func() (io.Writer, error) { return buf, nil },
		getBase:      makeNoUpdateBaseGetter(testProjectName, apisDir, gvks),
	}
	err := g.GenerateLegacy()
	if err != nil {
		t.Fatalf("Failed to execute CSV generator: %v", err)
	}

	// Read expected CSV
	csvFileName := getCSVFileLegacy(g.OperatorName, g.Version)
	csvExp := string(readFile(t, filepath.Join(inputDir, csvFileName)))

	assert.Equal(t, csvExp, buf.String())
}

func TestGoCSVUpdate(t *testing.T) {

	manifestRoot := filepath.Join(testGoDataDir, "deploy")
	crdsDir := filepath.Join(manifestRoot, "crds_v1beta1")
	col := &collector.Manifests{}
	if err := col.UpdateFromDirs(manifestRoot, crdsDir); err != nil {
		t.Fatalf("Error updating collector from manifest root %s: %v", manifestRoot, err)
	}

	inputDir := filepath.Join(manifestRoot, OLMCatalogDir, testProjectName)
	apisDir := filepath.Join(testGoDataDir, "pkg", "apis")
	buf := &bytes.Buffer{}
	g := Generator{
		OperatorName: testProjectName,
		Version:      newVersion,
		FromVersion:  "",
		getWriter:    func() (io.Writer, error) { return buf, nil },
	}
	err := g.GenerateLegacy(WithPackageBase(inputDir, apisDir))
	if err != nil {
		t.Fatalf("Failed to execute CSV generator: %v", err)
	}

	// Read expected CSV
	csvFileName := getCSVFileLegacy(g.OperatorName, g.Version)
	csvExp := string(readFile(t, filepath.Join(inputDir, csvFileName)))

	assert.Equal(t, csvExp, buf.String())
}

func TestGoCSVUpgrade(t *testing.T) {

	manifestRoot := filepath.Join(testGoDataDir, "deploy")
	crdsDir := filepath.Join(manifestRoot, "crds_v1beta1")
	col := &collector.Manifests{}
	if err := col.UpdateFromDirs(manifestRoot, crdsDir); err != nil {
		t.Fatalf("Error updating collector from manifest root %s: %v", manifestRoot, err)
	}

	inputDir := filepath.Join(manifestRoot, OLMCatalogDir, testProjectName)
	apisDir := filepath.Join(testGoDataDir, "pkg", "apis")
	buf := &bytes.Buffer{}
	g := Generator{
		OperatorName: testProjectName,
		Version:      newVersion,
		FromVersion:  fromVersion,
		getWriter:    func() (io.Writer, error) { return buf, nil },
	}
	err := g.GenerateLegacy(WithPackageBase(inputDir, apisDir))
	if err != nil {
		t.Fatalf("Failed to execute CSV generator: %v", err)
	}

	// Read expected CSV
	csvFileName := getCSVFileLegacy(g.OperatorName, g.Version)
	csvExp := string(readFile(t, filepath.Join(inputDir, csvFileName)))

	assert.Equal(t, csvExp, buf.String())
}

func TestGoCSVNewWithInvalidDeployDir(t *testing.T) {

	manifestRoot := filepath.Join(testGoDataDir, "notexist")
	inputDir := filepath.Join(manifestRoot, OLMCatalogDir, testProjectName)
	apisDir := filepath.Join(testGoDataDir, "pkg", "apis")
	buf := &bytes.Buffer{}
	g := Generator{
		OperatorName: testProjectName,
		Version:      newVersion,
		FromVersion:  "",
		getWriter:    func() (io.Writer, error) { return buf, nil },
	}
	err := g.GenerateLegacy(WithPackageBase(inputDir, apisDir))
	if err == nil {
		t.Fatalf("Failed to get error for running CSV generatoron non-existent manifests directory: %s",
			manifestRoot)
	}
}

func TestGoCSVNewWithEmptyDeployDir(t *testing.T) {

	manifestRoot := filepath.Join(testGoDataDir, "emptydir")
	crdsDir := filepath.Join(manifestRoot, "crds_v1beta1")
	col := &collector.Manifests{}
	if err := col.UpdateFromDirs(manifestRoot, crdsDir); err != nil {
		t.Fatalf("Error updating collector from manifest root %s: %v", manifestRoot, err)
	}

	inputDir := filepath.Join(manifestRoot, OLMCatalogDir, testProjectName)
	apisDir := filepath.Join(testGoDataDir, "pkg", "apis")
	buf := &bytes.Buffer{}
	g := Generator{
		OperatorName: testProjectName,
		Version:      notExistVersion,
		FromVersion:  "",
		getWriter:    func() (io.Writer, error) { return buf, nil },
	}
	err := g.GenerateLegacy(WithPackageBase(inputDir, apisDir))
	if err != nil {
		t.Fatalf("Failed to execute CSV generator: %v", err)
	}

	// Create a new CSV base.
	b := bases.ClusterServiceVersion{
		OperatorName: g.OperatorName,
	}
	csv, err := b.GetBase()
	if err != nil {
		t.Fatal(err)
	}
	csv.SetName(genutil.GetCSVName(g.OperatorName, notExistVersion))
	csvExpBytes, err := k8sutil.GetObjectBytes(csv, yaml.Marshal)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, string(csvExpBytes), buf.String())
}

func TestUpdateCSVVersion(t *testing.T) {

	b := bases.ClusterServiceVersion{
		OperatorName: testProjectName,
	}
	csv, err := b.GetBase()
	if err != nil {
		t.Fatal(err)
	}
	csv.SetName(genutil.GetCSVName(testProjectName, notExistVersion))
	depSpecs := make([]operatorsv1alpha1.StrategyDeploymentSpec, 1)
	csv.Spec.InstallStrategy = operatorsv1alpha1.NamedInstallStrategy{
		StrategyName: operatorsv1alpha1.InstallStrategyNameDeployment,
		StrategySpec: operatorsv1alpha1.StrategyDetailsDeployment{
			DeploymentSpecs: depSpecs,
		},
	}
	depSpecs[0].Spec.Template.Spec.Containers = make([]corev1.Container, 1)
	depSpecs[0].Spec.Template.Spec.Containers[0].Image = "quay.io/example/memcached-operator:v" + fromVersion

	g := Generator{
		OperatorName: testProjectName,
		Version:      newVersion,
		FromVersion:  fromVersion,
	}
	if err := g.updateVersions(csv); err != nil {
		t.Fatalf("Failed to update csv with version %s: (%v)", newVersion, err)
	}

	wantedSemver, err := semver.Parse(newVersion)
	if err != nil {
		t.Errorf("Failed to parse %s: %v", newVersion, err)
	}
	if !csv.Spec.Version.Equals(wantedSemver) {
		t.Errorf("Wanted csv version %v, got %v", wantedSemver, csv.Spec.Version)
	}
	wantedName := genutil.GetCSVName(testProjectName, newVersion)
	if csv.GetName() != wantedName {
		t.Errorf("Wanted csv name %s, got %s", wantedName, csv.ObjectMeta.Name)
	}

	csvPodImage := depSpecs[0].Spec.Template.Spec.Containers[0].Image
	// updateCSVVersions should not update podspec image.
	wantedImage := "quay.io/example/memcached-operator:v" + fromVersion
	if csvPodImage != wantedImage {
		t.Errorf("Podspec image changed from %s to %s", wantedImage, csvPodImage)
	}

	wantedReplaces := genutil.GetCSVName(testProjectName, fromVersion)
	if csv.Spec.Replaces != wantedReplaces {
		t.Errorf("Wanted csv replaces %s, got %s", wantedReplaces, csv.Spec.Replaces)
	}
}
