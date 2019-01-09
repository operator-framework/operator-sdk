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
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/operator-framework/operator-sdk/internal/util/yamlutil"
	"github.com/operator-framework/operator-sdk/pkg/scaffold"
	"github.com/operator-framework/operator-sdk/pkg/scaffold/input"

	"github.com/coreos/go-semver/semver"
	"github.com/ghodss/yaml"
	olmApi "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	log "github.com/sirupsen/logrus"
)

const (
	CSVYamlFilePrefix = ".csv.yaml"
	CSVConfigYamlFile = "csv-config.yaml"
)

type CSV struct {
	input.Input

	// DeployDir is the dir the SDK should search for deploy files, ex. *crd.yaml.
	DeployDir string
	// ConfigFilePath is the location of a configuration file path for this
	// projects' CSV file.
	ConfigFilePath string
	// CSVVersion is the CSV (and operators') current version.
	CSVVersion string
}

func (s *CSV) GetInput() (input.Input, error) {
	if s.Path == "" {
		fileName := strings.ToLower(s.ProjectName) + CSVYamlFilePrefix
		s.Path = filepath.Join(scaffold.OlmCatalogDir, fileName)
	}
	if s.ConfigFilePath == "" {
		s.ConfigFilePath = filepath.Join(scaffold.OlmCatalogDir, CSVConfigYamlFile)
	}
	if s.DeployDir == "" {
		s.DeployDir = scaffold.DeployDir
	}
	return s.Input, nil
}

// CustomRender allows a CSV to be written by marshalling
// olmApi.ClusterServiceVersion instead of writing to a template.
func (s *CSV) CustomRender() ([]byte, error) {
	// Get current CSV to update.
	currCSV, exists, err := getCurrentCSVIfExists(s.Path)
	if err != nil {
		return nil, err
	}
	if !exists {
		currCSV = new(olmApi.ClusterServiceVersion)
		s.initCSVFields(currCSV)
	}

	csvConfig, err := getCSVConfig(s.ConfigFilePath)
	if err != nil {
		return nil, err
	}

	if err = s.updateCSVVersions(currCSV); err != nil {
		return nil, err
	}
	if err = s.updateCSVFromManifestFiles(currCSV, csvConfig); err != nil {
		return nil, err
	}

	// A new csv won't have several required fields populated.
	if err = checkRequiredCSVFields(currCSV); err != nil {
		if exists {
			log.Warnf("required csv fields not filled in file %s:%s\n", s.Path, err)
		} else {
			// Report required fields to user informationally.
			log.Infof("fill in the following required fields in file %s:%s\n", s.Path, err)
		}
	}

	return yaml.Marshal(currCSV)
}

func getCurrentCSVIfExists(csvPath string) (*olmApi.ClusterServiceVersion, bool, error) {
	if _, err := os.Stat(csvPath); err != nil && os.IsNotExist(err) {
		return nil, false, nil
	}

	csvBytes, err := ioutil.ReadFile(csvPath)
	if err != nil {
		return nil, false, err
	}
	if len(csvBytes) == 0 {
		return nil, false, nil
	}

	csv := new(olmApi.ClusterServiceVersion)
	if err := yaml.Unmarshal(csvBytes, csv); err != nil {

		return nil, false, err
	}

	return csv, true, nil
}

func getCSVName(name, version string) string {
	return name + ".v" + version
}

// getDisplayName turns a project dir name in any of {snake, chain, camel}
// cases, hierarchical dot structure, or space-delimited into a
// space-delimited, title'd display name.
// Ex. "another-_AppOperator_againTwiceThrice More"
// ->  "Another App Operator Again Twice Thrice More"
func getDisplayName(name string) string {
	for _, sep := range ".-_ " {
		splitName := strings.Split(name, string(sep))
		for i := 0; i < len(splitName); i++ {
			if splitName[i] == "" {
				splitName = append(splitName[:i], splitName[i+1:]...)
				i--
			} else {
				splitName[i] = strings.TrimSpace(splitName[i])
			}
		}
		name = strings.Join(splitName, " ")
	}
	splitName := strings.Split(name, " ")
	for i, word := range splitName {
		temp := word
		o := 0
		for j, r := range word {
			if unicode.IsUpper(r) {
				if j > 0 && !unicode.IsUpper(rune(word[j-1])) {
					temp = temp[0:j+o] + " " + temp[j+o:len(temp)]
					o++
				}
			}
		}
		splitName[i] = temp
	}
	return strings.TrimSpace(strings.Title(strings.Join(splitName, " ")))
}

// initCSVFields initializes all csv fields that should be populated by a user
// with sane defaults. initCSVFields should only be called for new csv's.
func (s *CSV) initCSVFields(csv *olmApi.ClusterServiceVersion) {
	// Metadata
	csv.TypeMeta.APIVersion = olmApi.ClusterServiceVersionAPIVersion
	csv.TypeMeta.Kind = olmApi.ClusterServiceVersionKind
	csv.SetName(getCSVName(strings.ToLower(s.ProjectName), s.CSVVersion))
	csv.SetNamespace("placeholder")

	// Spec fields
	csv.Spec.Version = *semver.New(s.CSVVersion)
	csv.Spec.DisplayName = getDisplayName(s.ProjectName)
	csv.Spec.Description = "Placeholder description"
	csv.Spec.Maturity = "alpha"
	csv.Spec.Provider = olmApi.AppLink{}
	csv.Spec.Maintainers = make([]olmApi.Maintainer, 0)
	csv.Spec.Links = make([]olmApi.AppLink, 0)
	csv.SetLabels(make(map[string]string))
}

// TODO: validate that all fields from files are populated as expected
// ex. add `resources` to a CRD

func checkRequiredCSVFields(csv *olmApi.ClusterServiceVersion) error {
	errsb := &strings.Builder{}

	// Metadata
	if csv.TypeMeta.APIVersion != olmApi.ClusterServiceVersionAPIVersion {
		errsb.WriteString("\n\tapiVersion")
	}
	if csv.TypeMeta.Kind != olmApi.ClusterServiceVersionKind {
		errsb.WriteString("\n\tkind")
	}
	if csv.ObjectMeta.Name == "" {
		errsb.WriteString("\n\tmetadata.name")
	}
	// Spec fields
	if csv.Spec.Version.String() == "" {
		errsb.WriteString("\n\tspec.version")
	}
	if csv.Spec.DisplayName == "" {
		errsb.WriteString("\n\tspec.displayName")
	}
	if csv.Spec.Description == "" {
		errsb.WriteString("\n\tspec.description")
	}
	if len(csv.Spec.Keywords) == 0 {
		errsb.WriteString("\n\tspec.keywords")
	}
	if len(csv.Spec.Maintainers) == 0 {
		errsb.WriteString("\n\tspec.maintainers")
	}
	if csv.Spec.Provider == (olmApi.AppLink{}) {
		errsb.WriteString("\n\tspec.provider")
	}
	if len(csv.Spec.Labels) == 0 {
		errsb.WriteString("\n\tspec.labels")
	}

	if len(errsb.String()) == 0 {
		return nil
	}
	return errors.New(errsb.String())
}

// updateCSVVersions updates csv's version and data involving the version,
// ex. ObjectMeta.Name, and place the old version in the `replaces` object,
// if there is an old version to replace.
func (s *CSV) updateCSVVersions(csv *olmApi.ClusterServiceVersion) error {

	// Old csv version to replace, and updated csv version.
	oldVer, newVer := csv.Spec.Version.String(), s.CSVVersion
	if oldVer == newVer {
		return nil
	}

	// We do not want to update versions in most fields, as these versions are
	// independent of global csv version and will be updated elsewhere.
	fieldsToUpdate := []interface{}{
		&csv.ObjectMeta,
		&csv.Spec.Labels,
		&csv.Spec.Selector,
	}
	for _, v := range fieldsToUpdate {
		err := replaceAllBytes(v, []byte(oldVer), []byte(newVer))
		if err != nil {
			return err
		}
	}

	// Now replace all references to the old operator name.
	lowerProjName := strings.ToLower(s.ProjectName)
	oldCSVName := getCSVName(lowerProjName, oldVer)
	newCSVName := getCSVName(lowerProjName, newVer)
	err := replaceAllBytes(csv, []byte(oldCSVName), []byte(newCSVName))
	if err != nil {
		return err
	}

	newSemVer, err := semver.NewVersion(newVer)
	if err != nil {
		return err
	}
	csv.Spec.Version = *newSemVer
	csv.Spec.Replaces = oldCSVName
	return nil
}

func replaceAllBytes(v interface{}, old, new []byte) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	b = bytes.Replace(b, old, new, -1)
	if err = json.Unmarshal(b, v); err != nil {
		return err
	}
	return nil
}

// updateCSVFromManifestFiles gathers relevant data from generated and user-defined manifests
// and updates csv.
func (s *CSV) updateCSVFromManifestFiles(csv *olmApi.ClusterServiceVersion, csvConfig *CSVConfig) error {
	for _, f := range append(csvConfig.CrdCrPaths, csvConfig.OperatorPath, csvConfig.RolePath) {
		yamlData, err := ioutil.ReadFile(f)
		if err != nil {
			return err
		}

		scanner := yamlutil.NewYAMLScanner(yamlData)
		for scanner.Scan() {
			yamlSpec := scanner.Bytes()

			k, err := getKindfromYAML(yamlSpec)
			if err != nil {
				return err
			}

			updateFunc, ok := updateDispTable[k]
			if ok {
				updateFunc(yamlSpec)
			}
		}
	}

	updaters := &CSVUpdateSet{}
	updaters.Populate()
	return updaters.Apply(csv)
}

func getKindfromYAML(yamlData []byte) (string, error) {
	// Get Kind for inital categorization.
	var temp struct {
		Kind string
	}
	if err := yaml.Unmarshal(yamlData, &temp); err != nil {
		return "", err
	}
	return temp.Kind, nil
}
