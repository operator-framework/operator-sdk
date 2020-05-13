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
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/operator-framework/operator-sdk/internal/util/k8sutil"

	olmapiv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	log "github.com/sirupsen/logrus"
	"sigs.k8s.io/yaml"
)

// isBundleDirExist returns true if "parentDir/version" exists on disk.
func isBundleDirExist(parentDir, version string) bool {
	// Ensure full path is constructed.
	return version != "" && isExist(filepath.Join(parentDir, version))
}

func isNotExist(path string) bool {
	_, err := os.Stat(path)
	return err != nil && errors.Is(err, os.ErrNotExist)
}

func isExist(path string) bool {
	_, err := os.Stat(path)
	return err == nil || errors.Is(err, os.ErrExist)
}

// addCustomResourceDefinitionsToFileSet adds all CustomResourceDefinition
// manifests in dir to fileMap with file name keys.
func addCustomResourceDefinitionsToFileSet(dir string, fileMap map[string][]byte) error {
	infos, err := ioutil.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, info := range infos {
		if info.IsDir() {
			continue
		}

		fromPath := filepath.Join(dir, info.Name())
		b, err := ioutil.ReadFile(fromPath)
		if err != nil {
			return fmt.Errorf("error reading manifest %s: %v", fromPath, err)
		}

		scanner := k8sutil.NewYAMLScanner(bytes.NewBuffer(b))
		manifests := []byte{}
		for scanner.Scan() {
			manifest := scanner.Bytes()
			typeMeta, err := k8sutil.GetTypeMetaFromBytes(manifest)
			if err != nil {
				log.Debugf("Skipping non-Object manifest %s: %v", fromPath, err)
				continue
			}
			if typeMeta.Kind == "CustomResourceDefinition" {
				manifests = k8sutil.CombineManifests(manifests, b)
			}
		}
		if err = scanner.Err(); err != nil {
			return err
		}

		if len(manifests) != 0 {
			fileMap[info.Name()] = manifests
		}
	}

	return nil
}

// getCSVFromDir returns the ClusterServiceVersion manifest in dir. If no
// manifest is found, getCSVFromDir returns an error.
func getCSVFromDir(dir string) (*olmapiv1alpha1.ClusterServiceVersion, error) {
	infos, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, info := range infos {
		// Only read manifest from files, not directories
		if info.IsDir() {
			continue
		}

		path := filepath.Join(dir, info.Name())
		b, err := ioutil.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("error reading manifest %s: %v", path, err)
		}

		scanner := k8sutil.NewYAMLScanner(bytes.NewBuffer(b))
		for scanner.Scan() {
			manifest := scanner.Bytes()
			typeMeta, err := k8sutil.GetTypeMetaFromBytes(manifest)
			if err != nil {
				log.Debugf("Skipping non-Object manifest %s: %v", path, err)
				continue
			}
			if typeMeta.Kind == olmapiv1alpha1.ClusterServiceVersionKind {
				csv := &olmapiv1alpha1.ClusterServiceVersion{}
				if err := yaml.Unmarshal(manifest, csv); err != nil {
					return nil, fmt.Errorf("error unmarshalling ClusterServiceVersion from manifest %s: %v", path, err)
				}
				return csv, nil
			}
		}
		if err = scanner.Err(); err != nil {
			return nil, fmt.Errorf("error scanning manifest %s: %v", path, err)
		}
	}

	return nil, fmt.Errorf("no CSV manifest in %s", dir)
}

func joinFields(fields []string) string {
	sb := &strings.Builder{}
	for _, f := range fields {
		sb.WriteString("\n\t" + f)
	}
	return sb.String()
}
