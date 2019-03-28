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

package catalog

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/operator-framework/operator-sdk/internal/pkg/scaffold"
	"github.com/operator-framework/operator-sdk/internal/pkg/scaffold/input"
	"github.com/operator-framework/operator-sdk/internal/util/diffutil"
	"github.com/operator-framework/operator-sdk/internal/util/fileutil"

	"github.com/coreos/go-semver/semver"
	"github.com/ghodss/yaml"
	olmapiv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	olminstall "github.com/operator-framework/operator-lifecycle-manager/pkg/controller/install"
	"github.com/spf13/afero"
)

const testDataDir = "testdata"

var testDeployDir = filepath.Join(testDataDir, scaffold.DeployDir)

func TestCSVNew(t *testing.T) {
	buf := &bytes.Buffer{}
	s := &scaffold.Scaffold{
		GetWriter: func(_ string, _ os.FileMode) (io.Writer, error) {
			return buf, nil
		},
	}
	csvVer := "0.1.0"
	projectName := "app-operator"

	sc := &CSV{CSVVersion: csvVer, pathPrefix: testDataDir}
	err := s.Execute(&input.Config{ProjectName: projectName}, sc)
	if err != nil {
		t.Fatalf("Failed to execute the scaffold: (%v)", err)
	}

	// Get the expected CSV manifest from test data dir.
	csvExpBytes, err := afero.ReadFile(s.Fs, sc.getCSVPath(csvVer))
	if err != nil {
		t.Fatal(err)
	}
	csvExp := string(csvExpBytes)
	if csvExp != buf.String() {
		diffs := diffutil.Diff(csvExp, buf.String())
		t.Errorf("Expected vs actual differs.\n%v", diffs)
	}
}

func writeOSPathToFS(fromFs, toFs afero.Fs, root string) error {
	if _, err := fromFs.Stat(root); err != nil {
		return err
	}
	return afero.Walk(fromFs, root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil {
			return err
		}
		if !info.IsDir() {
			b, err := afero.ReadFile(fromFs, path)
			if err != nil {
				return err
			}
			return afero.WriteFile(toFs, path, b, fileutil.DefaultFileMode)
		}
		return nil
	})
}

func TestCSVFromOld(t *testing.T) {
	s := &scaffold.Scaffold{Fs: afero.NewMemMapFs()}
	projectName := "app-operator"
	oldCSVVer, newCSVVer := "0.1.0", "0.2.0"

	// Write all files in testdata/deploy to fs so manifests are present when
	// writing a new CSV.
	if err := writeOSPathToFS(afero.NewOsFs(), s.Fs, testDeployDir); err != nil {
		t.Fatalf("Failed to write %s to in-memory test fs: (%v)", testDeployDir, err)
	}

	sc := &CSV{
		CSVVersion:  newCSVVer,
		FromVersion: oldCSVVer,
		pathPrefix:  testDataDir,
	}
	err := s.Execute(&input.Config{ProjectName: projectName}, sc)
	if err != nil {
		t.Fatalf("Failed to execute the scaffold: (%v)", err)
	}

	// Check if a new file was written at the expected path.
	newCSVPath := sc.getCSVPath(newCSVVer)
	newCSV, newExists, err := getCSVFromFSIfExists(s.Fs, newCSVPath)
	if err != nil {
		t.Fatalf("Failed to get new CSV %s: (%v)", newCSVPath, err)
	}
	if !newExists {
		t.Fatalf("New CSV does not exist at %s", newCSVPath)
	}

	expName := getCSVName(projectName, newCSVVer)
	if newCSV.ObjectMeta.Name != expName {
		t.Errorf("Expected CSV metadata.name %s, got %s", expName, newCSV.ObjectMeta.Name)
	}
	expReplaces := getCSVName(projectName, oldCSVVer)
	if newCSV.Spec.Replaces != expReplaces {
		t.Errorf("Expected CSV spec.replaces %s, got %s", expReplaces, newCSV.Spec.Replaces)
	}
}

func TestUpdateVersion(t *testing.T) {
	projectName := "app-operator"
	oldCSVVer, newCSVVer := "0.1.0", "0.2.0"
	sc := &CSV{
		Input:      input.Input{ProjectName: projectName},
		CSVVersion: newCSVVer,
		pathPrefix: testDataDir,
	}
	csvExpBytes, err := ioutil.ReadFile(sc.getCSVPath(oldCSVVer))
	if err != nil {
		t.Fatal(err)
	}
	csv := &olmapiv1alpha1.ClusterServiceVersion{}
	if err := yaml.Unmarshal(csvExpBytes, csv); err != nil {
		t.Fatal(err)
	}

	if err := sc.updateCSVVersions(csv); err != nil {
		t.Fatalf("Failed to update csv with version %s: (%v)", newCSVVer, err)
	}

	wantedSemver := semver.New(newCSVVer)
	if !csv.Spec.Version.Equal(*wantedSemver) {
		t.Errorf("Wanted csv version %v, got %v", *wantedSemver, csv.Spec.Version)
	}
	wantedName := getCSVName(projectName, newCSVVer)
	if csv.ObjectMeta.Name != wantedName {
		t.Errorf("Wanted csv name %s, got %s", wantedName, csv.ObjectMeta.Name)
	}

	var resolver *olminstall.StrategyResolver
	stratInterface, err := resolver.UnmarshalStrategy(csv.Spec.InstallStrategy)
	if err != nil {
		t.Fatal(err)
	}
	strat, ok := stratInterface.(*olminstall.StrategyDetailsDeployment)
	if !ok {
		t.Fatalf("Strategy of type %T was not StrategyDetailsDeployment", stratInterface)
	}
	csvPodImage := strat.DeploymentSpecs[0].Spec.Template.Spec.Containers[0].Image
	// updateCSVVersions should not update podspec image.
	wantedImage := "quay.io/example-inc/operator:v0.1.0"
	if csvPodImage != wantedImage {
		t.Errorf("Podspec image changed from %s to %s", wantedImage, csvPodImage)
	}

	wantedReplaces := getCSVName(projectName, "0.1.0")
	if csv.Spec.Replaces != wantedReplaces {
		t.Errorf("Wanted csv replaces %s, got %s", wantedReplaces, csv.Spec.Replaces)
	}
}

func TestGetDisplayName(t *testing.T) {
	cases := []struct {
		input, wanted string
	}{
		{"Appoperator", "Appoperator"},
		{"appoperator", "Appoperator"},
		{"appoperatoR", "Appoperato R"},
		{"AppOperator", "App Operator"},
		{"appOperator", "App Operator"},
		{"app-operator", "App Operator"},
		{"app-_operator", "App Operator"},
		{"App-operator", "App Operator"},
		{"app-_Operator", "App Operator"},
		{"app--Operator", "App Operator"},
		{"app--_Operator", "App Operator"},
		{"APP", "APP"},
		{"another-AppOperator_againTwiceThrice More", "Another App Operator Again Twice Thrice More"},
	}

	for _, c := range cases {
		dn := getDisplayName(c.input)
		if dn != c.wanted {
			t.Errorf("Wanted %s, got %s", c.wanted, dn)
		}
	}
}
