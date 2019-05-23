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
	"os"
	"path/filepath"
	"strings"

	yaml "github.com/ghodss/yaml"
	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
)

func GetCRDs(crdsDir string) ([]*apiextv1beta1.CustomResourceDefinition, error) {
	manifests, err := GetCRDManifestPaths(crdsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get CRD's from %s: (%v)", crdsDir, err)
	}
	var crds []*apiextv1beta1.CustomResourceDefinition
	for _, m := range manifests {
		b, err := ioutil.ReadFile(m)
		if err != nil {
			return nil, err
		}
		crd := &apiextv1beta1.CustomResourceDefinition{}
		if err = yaml.Unmarshal(b, crd); err != nil {
			return nil, err
		}
		crds = append(crds, crd)
	}
	return crds, nil
}

func GetCRDManifestPaths(crdsDir string) (crdPaths []string, err error) {
	err = filepath.Walk(crdsDir, func(path string, info os.FileInfo, werr error) error {
		if werr != nil {
			return werr
		}
		if info == nil {
			return nil
		}
		if !info.IsDir() && strings.HasSuffix(path, "_crd.yaml") {
			crdPaths = append(crdPaths, path)
		}
		return nil
	})
	return crdPaths, err
}
