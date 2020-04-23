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

package scorecardutil

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/operator-framework/operator-registry/pkg/registry"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
)

// TODO(joelanford): This function should probably be upstreamed into
//     operator-framework/operator-registry and improved. Much of the code
//     upstream already exists, but not in a consumable way. See:
//     https://github.com/operator-framework/operator-registry/blob/2bdcd07e6f9ea23a35f1fdbb6a7c6584ec53a0a1/pkg/registry/imageinput.go#L25-L81
//
// LoadBundleDirectory parses a Bundle from a given on-disk path.
func LoadBundleDirectory(bundlePath string) (*registry.Bundle, error) {
	metadataDir := filepath.Join(bundlePath, "metadata")
	manifestsDir := filepath.Join(bundlePath, "manifests")

	annotationsPath := filepath.Join(metadataDir, "annotations.yaml")
	annotations, err := decodeAnnotationsFile(annotationsPath)
	if err != nil {
		return nil, err
	}

	files, err := ioutil.ReadDir(manifestsDir)
	if err != nil {
		return nil, err
	}

	fileStrings := []string{}
	for _, f := range files {
		if f.IsDir() {
			continue
		}

		fStr, err := ioutil.ReadFile(filepath.Join(manifestsDir, f.Name()))
		if err != nil {
			return nil, err
		}
		fileStrings = append(fileStrings, string(fStr))
	}
	return registry.NewBundleFromStrings(annotations.GetName(), annotations.GetName(), annotations.GetDefaultChannelName(), fileStrings)
}

func decodeAnnotationsFile(path string) (*registry.AnnotationsFile, error) {
	annotationsData, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	annotations := registry.AnnotationsFile{}
	if err := yaml.Unmarshal(annotationsData, &annotations); err != nil {
		return nil, err
	}
	return &annotations, nil
}

// TODO(joelanford): bump to Kubernetes v1.18 and latest operator-registry
//  to get access to dependencies file definitions.
//func decodeDependenciesFile(path string) (*registry.DependenciesFile, error) {
//	dependenciesData, err := ioutil.ReadFile(path)
//	if err != nil {
//		return nil, err
//	}
//	dependencies := registry.DependenciesFile{}
//	if err := yaml.Unmarshal(dependenciesData, &dependencies); err != nil {
//		return nil, err
//	}
//	return &dependencies, nil
//}

// GetALMExamples parses the bundle CSV to extract, parse, and return the
// example resources.
func GetALMExamples(bundle registry.Bundle) ([]unstructured.Unstructured, error) {
	csv, err := bundle.ClusterServiceVersion()
	if err != nil {
		return nil, fmt.Errorf("error in csv retrieval %s", err.Error())
	}

	if csv.GetAnnotations() == nil {
		return nil, nil
	}

	almExamplesJSON, ok := csv.GetAnnotations()["alm-examples"]
	if !ok {
		return nil, nil
	}
	return parseExamples(almExamplesJSON)
}

func parseExamples(in string) ([]unstructured.Unstructured, error) {
	// get CRs from CSV's alm-examples annotation, assume single bundle
	almExamples := make([]unstructured.Unstructured, 0)
	if err := json.Unmarshal([]byte(in), &almExamples); err != nil {
		return nil, err
	}
	return almExamples, nil
}
