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

	"github.com/operator-framework/operator-sdk/internal/util/diffutil"
	"github.com/operator-framework/operator-sdk/pkg/scaffold"
	"github.com/operator-framework/operator-sdk/pkg/scaffold/input"

	"github.com/coreos/go-semver/semver"
	"github.com/ghodss/yaml"
	olmapiv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	olminstall "github.com/operator-framework/operator-lifecycle-manager/pkg/controller/install"
)

const testDataDir = "testdata"

var (
	testOLMDir = filepath.Join(testDataDir, scaffold.OLMCatalogDir)
)

func TestCSV(t *testing.T) {
	buf := &bytes.Buffer{}
	s := &scaffold.Scaffold{
		GetWriter: func(_ string, _ os.FileMode) (io.Writer, error) {
			return buf, nil
		},
	}
	csvVer := "0.1.0"
	projectName := "app-operator"

	err := s.Execute(&input.Config{ProjectName: projectName},
		&CSV{
			CSVVersion:     csvVer,
			ConfigFilePath: filepath.Join(testOLMDir, CSVConfigYamlFile),
		},
	)
	if err != nil {
		t.Fatalf("Failed to execute the scaffold: (%v)", err)
	}

	// Get the expected CSV manifest from test data dir.
	csvFilePath := projectName + CSVYamlFileExt
	csvExpBytes, err := ioutil.ReadFile(filepath.Join(testOLMDir, csvFilePath))
	if err != nil {
		t.Fatal(err)
	}
	csvExp := string(csvExpBytes)

	if csvExp != buf.String() {
		diffs := diffutil.Diff(csvExp, buf.String())
		t.Fatalf("Expected vs actual differs.\n%v", diffs)
	}
}

func TestUpdateVersion(t *testing.T) {
	projectName := "app-operator"
	csvFilePath := projectName + CSVYamlFileExt
	csvExpBytes, err := ioutil.ReadFile(filepath.Join(testOLMDir, csvFilePath))
	if err != nil {
		t.Fatal(err)
	}
	csv := &olmapiv1alpha1.ClusterServiceVersion{}
	if err := yaml.Unmarshal(csvExpBytes, csv); err != nil {
		t.Fatal(err)
	}

	newCSVVer := "0.2.0"
	c := &CSV{
		Input: input.Input{
			ProjectName: projectName,
		},
		CSVVersion: newCSVVer,
	}
	if err := c.updateCSVVersions(csv); err != nil {
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
