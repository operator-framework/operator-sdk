// Copyright 2019 The Operator-SDK Authors
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

package k8sutil

import (
	"fmt"
	"io/ioutil"
	"path"
	"path/filepath"
	"regexp"

	log "github.com/sirupsen/logrus"
	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/version"
	"sigs.k8s.io/yaml"
)

// GetCustomResourceDefinitions returns all CRD manifests in the directory
// crdsDir.
func GetCustomResourceDefinitions(crdsDir string) (crds []apiextv1beta1.CustomResourceDefinition, err error) {
	infos, err := ioutil.ReadDir(crdsDir)
	if err != nil {
		return nil, err
	}
	for _, info := range infos {
		if info.IsDir() {
			continue
		}
		path := filepath.Join(crdsDir, info.Name())
		b, err := ioutil.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("error reading manifest %s: %w", path, err)
		}

		scanner := NewYAMLScanner(b)
		for scanner.Scan() {
			manifest := scanner.Bytes()
			typeMeta, err := GetTypeMetaFromBytes(manifest)
			if err != nil {
				log.Debugf("Skipping manifest in %s: %v", path, err)
				continue
			}
			if typeMeta.Kind == "CustomResourceDefinition" {
				crd := apiextv1beta1.CustomResourceDefinition{}
				if err = yaml.Unmarshal(manifest, &crd); err != nil {
					return nil, fmt.Errorf("error unmarshalling CustomResourceDefinition: %w", err)
				}
				crds = append(crds, crd)
			}
		}
		if err = scanner.Err(); err != nil {
			return nil, fmt.Errorf("error scanning %s: %w", path, err)
		}
	}
	return crds, nil
}

// ParseGroupSubpackages parses the apisDir directory tree and returns a map of
// all found API groups to subpackages.
func ParseGroupSubpackages(apisDir string) (map[string][]string, error) {
	return parseGroupSubdirs(apisDir, false)
}

// ParseGroupVersions parses the apisDir directory tree and returns a map of
// all found API groups to versions.
func ParseGroupVersions(apisDir string) (map[string][]string, error) {
	return parseGroupSubdirs(apisDir, true)
}

// versionRegexp defines a kube-like version:
// https://kubernetes.io/docs/concepts/overview/kubernetes-api/#api-versioning
var versionRegexp = regexp.MustCompile("^v[1-9][0-9]*((alpha|beta)[1-9][0-9]*)?$")

// parseGroupSubdirs searches apisDir for all groups and potential version
// subdirs directly beneath each group dir, and returns a map of each group
// dir name to all children version dir names. If strictVersionMatch is true,
// all potential version dir names must strictly match versionRegexp. If
// false, all subdir names are considered valid.
func parseGroupSubdirs(apisDir string, strictVersionMatch bool) (map[string][]string, error) {
	gvs := make(map[string][]string)
	groups, err := ioutil.ReadDir(apisDir)
	if err != nil {
		return nil, fmt.Errorf("error reading directory %q to find API groups: %v", apisDir, err)
	}

	for _, g := range groups {
		if g.IsDir() {
			groupDir := filepath.Join(apisDir, g.Name())
			versions, err := ioutil.ReadDir(groupDir)
			if err != nil {
				return nil, fmt.Errorf("error reading directory %q to find API versions: %v", groupDir, err)
			}

			gvs[g.Name()] = make([]string, 0)
			for _, v := range versions {
				if v.IsDir() {
					// Ignore directories that do not contain any files, so generators
					// do not get empty directories as arguments.
					verDir := filepath.Join(groupDir, v.Name())
					files, err := ioutil.ReadDir(verDir)
					if err != nil {
						return nil, fmt.Errorf("error reading directory %q to find API source files: %v", verDir, err)
					}
					for _, f := range files {
						if !f.IsDir() && filepath.Ext(f.Name()) == ".go" {
							// If strictVersionMatch is true, strictly check if v.Name()
							// is a Kubernetes API version.
							if !strictVersionMatch || versionRegexp.MatchString(v.Name()) {
								gvs[g.Name()] = append(gvs[g.Name()], v.Name())
							}
							break
						}
					}
				}
			}
		}
	}

	if len(gvs) == 0 {
		return nil, fmt.Errorf("no groups or versions found in %s", apisDir)
	}
	return gvs, nil
}

// CreateFQAPIs return a slice of all fully qualified pkg + groups + versions
// of pkg and gvs in the format "pkg/groupA/v1".
func CreateFQAPIs(pkg string, gvs map[string][]string) (apis []string) {
	for g, vs := range gvs {
		for _, v := range vs {
			apis = append(apis, path.Join(pkg, g, v))
		}
	}
	return apis
}

type CRDVersions []apiextv1beta1.CustomResourceDefinitionVersion

func (vs CRDVersions) Len() int { return len(vs) }
func (vs CRDVersions) Less(i, j int) bool {
	return version.CompareKubeAwareVersionStrings(vs[i].Name, vs[j].Name) > 0
}
func (vs CRDVersions) Swap(i, j int) { vs[i], vs[j] = vs[j], vs[i] }
