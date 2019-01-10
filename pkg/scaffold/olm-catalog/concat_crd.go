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
	"io/ioutil"
	"path/filepath"

	"github.com/operator-framework/operator-sdk/internal/util/yamlutil"
	"github.com/operator-framework/operator-sdk/pkg/scaffold"
	"github.com/operator-framework/operator-sdk/pkg/scaffold/input"
)

const (
	ConcatCRDYamlFile = "_generated.concat_crd.yaml"
)

type ConcatCRD struct {
	input.Input

	// ConfigFilePath is the location of a configuration file path for this
	// projects' CSV file.
	ConfigFilePath string
}

func (s *ConcatCRD) GetInput() (input.Input, error) {
	if s.Path == "" {
		s.Path = filepath.Join(scaffold.OlmCatalogDir, ConcatCRDYamlFile)
	}
	if s.ConfigFilePath == "" {
		s.ConfigFilePath = filepath.Join(scaffold.OlmCatalogDir, CSVConfigYamlFile)
	}
	return s.Input, nil
}

// CustomRender returns the bytes of all CRD manifests concatenated into one file.
func (s *ConcatCRD) CustomRender() ([]byte, error) {
	csvConfig, err := getCSVConfig(s.ConfigFilePath)
	if err != nil {
		return nil, err
	}

	return s.concatCSVsInPaths(csvConfig.CRDCRPaths)
}

// concatCSVsInCRDsDir concatenates CRD manifests found at crdPaths into one
// file, delimited by `---`.
func (s *ConcatCRD) concatCSVsInPaths(crdPaths []string) ([]byte, error) {
	var concatCRD []byte
	for _, f := range crdPaths {
		yamlData, err := ioutil.ReadFile(f)
		if err != nil {
			return nil, err
		}

		scanner := yamlutil.NewYAMLScanner(yamlData)
		for scanner.Scan() {
			yamlSpec := scanner.Bytes()

			k, err := getKindfromYAML(yamlSpec)
			if err != nil {
				return nil, err
			}
			if k == "CustomResourceDefinition" {
				concatCRD = yamlutil.CombineManifests(concatCRD, yamlSpec)
			}
		}
	}

	return concatCRD, nil
}
