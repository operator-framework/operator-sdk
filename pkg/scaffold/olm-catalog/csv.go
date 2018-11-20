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
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/operator-framework/operator-sdk/pkg/scaffold"
	"github.com/operator-framework/operator-sdk/pkg/scaffold/input"

	"github.com/coreos/go-semver/semver"
	"github.com/ghodss/yaml"
	olmApi "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	log "github.com/sirupsen/logrus"
	k8syaml "k8s.io/apimachinery/pkg/util/yaml"
)

const (
	CsvYamlFilePrefix = ".csv.yaml"
	CsvConfigYamlFile = "csv-config.yaml"
)

type Csv struct {
	input.Input

	// DeployDir is the dir the SDK should search for deploy files, ex. *crd.yaml.
	DeployDir string
	// ConfigFilePath is the location of a configuration file path for this
	// projects' CSV file.
	ConfigFilePath string
	// OperatorVersion is the operators' current version.
	OperatorVersion string
}

func (s *Csv) GetInput() (input.Input, error) {
	if s.Path == "" {
		fileName := strings.ToLower(s.ProjectName) + CsvYamlFilePrefix
		s.Path = filepath.Join(scaffold.OlmCatalogDir, fileName)
	}
	if s.ConfigFilePath == "" {
		s.ConfigFilePath = filepath.Join(scaffold.OlmCatalogDir, CsvConfigYamlFile)
	}
	if s.DeployDir == "" {
		s.DeployDir = scaffold.DeployDir
	}
	return s.Input, nil
}

// CustomRender allows a Csv to be written by marshalling
// olmApi.ClusterServiceVersion instead of writing to a template.
func (s *Csv) CustomRender() ([]byte, error) {
	// Get current CSV to update.
	currCSV, exists, err := getCurrentCSVIfExists(s.Path)
	if err != nil {
		return nil, err
	}
	if !exists {
		currCSV = new(olmApi.ClusterServiceVersion)
		if err = s.initCSVFields(currCSV); err != nil {
			return nil, err
		}
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
	if exists {
		if err = checkRequiredCSVFields(currCSV); err != nil {
			log.Warnf("%s\nFill in these fields in file %s\n", err, s.Path)
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
		log.Infof("getCurrentCSVIfExists ReadFile: (%v)", err)
		return nil, false, err
	}
	if len(csvBytes) == 0 {
		return nil, false, nil
	}

	csv := new(olmApi.ClusterServiceVersion)
	if err := yaml.Unmarshal(csvBytes, csv); err != nil {
		log.Infof("getCurrentCSVIfExists Unmarshal: (%v)", err)
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
			splitName[i] = strings.TrimSpace(splitName[i])
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
func (s *Csv) initCSVFields(csv *olmApi.ClusterServiceVersion) error {
	// Metadata
	csv.TypeMeta.APIVersion = olmApi.ClusterServiceVersionAPIVersion
	csv.TypeMeta.Kind = olmApi.ClusterServiceVersionKind
	csv.ObjectMeta.Name = getCSVName(strings.ToLower(s.ProjectName), s.OperatorVersion)
	csv.ObjectMeta.Namespace = "placeholder"

	// Spec fields
	csv.Spec.Version = *semver.New(s.OperatorVersion)
	csv.Spec.DisplayName = getDisplayName(s.ProjectName)
	csv.Spec.Description = "Placeholder description"
	csv.Spec.Maturity = "alpha"
	csv.Spec.Provider = olmApi.AppLink{}
	csv.Spec.Maintainers = make([]olmApi.Maintainer, 0)
	csv.Spec.Links = make([]olmApi.AppLink, 0)
	csv.Spec.Labels = make(map[string]string)

	return nil
}

// TODO: validate that all fields from files are populated as expected
// ex. add `resources` to a CRD

func checkRequiredCSVFields(csv *olmApi.ClusterServiceVersion) error {
	incorrectFields := make([]string, 0)

	// Metadata
	if csv.TypeMeta.APIVersion != olmApi.ClusterServiceVersionAPIVersion {
		incorrectFields = append(incorrectFields, "apiVersion")
	}
	if csv.TypeMeta.Kind != olmApi.ClusterServiceVersionKind {
		incorrectFields = append(incorrectFields, "kind")
	}
	if csv.ObjectMeta.Name == "" {
		incorrectFields = append(incorrectFields, "metadata.name")
	}

	// Spec fields
	if csv.Spec.Version.String() == "" {
		incorrectFields = append(incorrectFields, "spec.version")
	}
	if csv.Spec.DisplayName == "" {
		incorrectFields = append(incorrectFields, "spec.displayName")
	}
	if csv.Spec.Description == "" {
		incorrectFields = append(incorrectFields, "spec.description")
	}
	if len(csv.Spec.Keywords) == 0 {
		incorrectFields = append(incorrectFields, "spec.keywords")
	}
	if len(csv.Spec.Maintainers) == 0 {
		incorrectFields = append(incorrectFields, "spec.maintainers")
	}
	if csv.Spec.Provider == (olmApi.AppLink{}) {
		incorrectFields = append(incorrectFields, "spec.provider")
	}
	if len(csv.Spec.Labels) == 0 {
		incorrectFields = append(incorrectFields, "spec.labels")
	}

	if len(incorrectFields) == 0 {
		return nil
	}
	errStr := "required csv fields not filled:"
	for _, field := range incorrectFields {
		errStr += "\n\t" + field
	}

	return errors.New(errStr)
}

// updateCSVVersions updates csv's version and data involving the version,
// ex. ObjectMeta.Name, and place the old version in the `replaces` object,
// if there is an old version to replace.
func (s *Csv) updateCSVVersions(csv *olmApi.ClusterServiceVersion) error {

	// Old csv version to replace, and updated csv version.
	oldVer, newVer := csv.Spec.Version.String(), s.OperatorVersion
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
		err := replaceAllBytes(v, v, []byte(oldVer), []byte(newVer))
		if err != nil {
			return err
		}
	}

	// Now replace all references to the old operator name.
	lowerProjName := strings.ToLower(s.ProjectName)
	oldCSVName := getCSVName(lowerProjName, oldVer)
	newCSVName := getCSVName(lowerProjName, newVer)
	err := replaceAllBytes(csv, csv, []byte(oldCSVName), []byte(newCSVName))
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

func replaceAllBytes(src, dst interface{}, old, new []byte) error {
	b, err := json.Marshal(src)
	if err != nil {
		log.Infof("replaceAllBytes: (%v)", err)
		return err
	}
	b = bytes.Replace(b, old, new, -1)
	if err = json.Unmarshal(b, dst); err != nil {
		log.Infof("replaceAllBytes: (%v)", err)
		return err
	}
	return nil
}

// updateCSVFromManifestFiles gathers relevant data from generated and user-defined manifests
// and updates csv.
func (s *Csv) updateCSVFromManifestFiles(csv *olmApi.ClusterServiceVersion, csvConfig *CsvConfig) error {
	for _, f := range append(csvConfig.CrdCrPaths, csvConfig.OperatorPath, csvConfig.RolePath) {
		yamlData, err := ioutil.ReadFile(f)
		if err != nil {
			log.Infof("updateCSVFromManifestFiles ReadFile %v: (%v)", f, err)
			return err
		}

		// Individual k8s YAML documents can contain delimited YAML manifests
		// that should be processed inidividually. We parse each file assuming
		// there are multiple manifests in one document.
		readBuf := bytes.NewBuffer(yamlData)
		reader := k8syaml.NewYAMLReader(bufio.NewReader(readBuf))
		for {
			yamlDoc, rerr := reader.Read()
			if rerr != nil && rerr != io.EOF {
				log.Infof("updateCSVFromManifestFiles Read: (%v)", rerr)
				return rerr
			}
			// No more separator-delimited YAML documents within yamlData.
			if rerr == io.EOF {
				break
			}

			k, err := getKindfromYAML(yamlDoc)
			if err != nil {
				return err
			}

			updateFunc, ok := updateDispTable[k]
			if ok {
				updateFunc(yamlDoc)
			}
		}
	}

	updaters := &CSVUpdateSet{}
	updaters.Populate()
	return updaters.ApplyAll(csv)
}

func getKindfromYAML(yamlData []byte) (string, error) {
	// Get Kind for inital categorization.
	var temp struct {
		Kind string
	}
	if err := yaml.Unmarshal(yamlData, &temp); err != nil {
		log.Infof("getKindfromYAML Unmarshal: (%v)", err)
		return "", err
	}
	return temp.Kind, nil
}
