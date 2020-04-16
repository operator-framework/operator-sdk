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

package bases

import (
	"bytes"
	"fmt"
	"io/ioutil"

	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/yaml"

	"github.com/operator-framework/operator-sdk/internal/util/k8sutil"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"
)

type ClusterServiceVersion struct {
	// OperatorName is the operator's name, ex. app-operator
	OperatorName string
	// OperatorType
	OperatorType projutil.OperatorType
	// APIsDir contains project API definition files.
	APIsDir string
	// GVKs are all GroupVersionKinds in the project.
	GVKs []schema.GroupVersionKind
	// BasePath is the path to the base being read.
	BasePath string

	// Fields for input to the base.
	DisplayName  string
	Description  string
	Maturity     string
	Capabilities string
	Keywords     []string
	Provider     operatorsv1alpha1.AppLink
	Links        []operatorsv1alpha1.AppLink
	Maintainers  []operatorsv1alpha1.Maintainer
	// TODO(estroz): read icon bytes from files.
	Icon []operatorsv1alpha1.Icon
}

func (b ClusterServiceVersion) GetBase() (base *operatorsv1alpha1.ClusterServiceVersion, err error) {
	if b.BasePath != "" {
		if base, err = readClusterServiceVersionBase(b.BasePath); err != nil {
			return nil, fmt.Errorf("error reading existing ClusterServiceVersion base %s: %v", b.BasePath, err)
		}
	} else {
		b.setDefaults()
		base = b.makeNewBase()
	}

	if b.APIsDir != "" {
		switch b.OperatorType {
		case projutil.OperatorTypeGo:
			if err := updateDescriptionsForGVKs(base, b.APIsDir, b.GVKs); err != nil {
				return nil, fmt.Errorf("error generating ClusterServiceVersion base metadata: %w", err)
			}
		}
	}

	return base, nil
}

func (b *ClusterServiceVersion) setDefaults() {
	if b.DisplayName == "" {
		b.DisplayName = k8sutil.GetDisplayName(b.OperatorName)
	}
	if b.Description == "" {
		b.Description = b.DisplayName + " description. Fill me in."
	}
	if b.Maturity == "" {
		b.Maturity = "alpha"
	}
	if b.Capabilities == "" {
		b.Capabilities = "Basic Install"
	}
	if len(b.Keywords) == 0 || b.Keywords[0] == "" {
		b.Keywords = []string{b.OperatorName}
	}
	if len(b.Links) == 0 || b.Links[0] == (operatorsv1alpha1.AppLink{}) {
		b.Links = []operatorsv1alpha1.AppLink{
			{
				Name: b.DisplayName,
				URL:  fmt.Sprintf("https://%s.domain", b.OperatorName),
			},
		}
	}
	if len(b.Maintainers) == 0 || b.Maintainers[0] == (operatorsv1alpha1.Maintainer{}) {
		b.Maintainers = []operatorsv1alpha1.Maintainer{
			{
				Name:  "Maintainer Name",
				Email: "your@email.com",
			},
		}
	}
	if b.Provider == (operatorsv1alpha1.AppLink{}) {
		b.Provider = operatorsv1alpha1.AppLink{
			Name: "Provider Name",
			URL:  "https://your.domain",
		}
	}
	if len(b.Icon) == 0 || b.Icon[0] == (operatorsv1alpha1.Icon{}) {
		b.Icon = make([]operatorsv1alpha1.Icon, 1)
	}
}

func (b ClusterServiceVersion) makeNewBase() *operatorsv1alpha1.ClusterServiceVersion {
	return &operatorsv1alpha1.ClusterServiceVersion{
		TypeMeta: metav1.TypeMeta{
			APIVersion: operatorsv1alpha1.ClusterServiceVersionAPIVersion,
			Kind:       operatorsv1alpha1.ClusterServiceVersionKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      b.OperatorName + ".vX.Y.Z",
			Namespace: "placeholder",
			Annotations: map[string]string{
				"capabilities": b.Capabilities,
				"alm-examples": "[]",
			},
		},
		Spec: operatorsv1alpha1.ClusterServiceVersionSpec{
			DisplayName: b.DisplayName,
			Description: b.Description,
			Provider:    b.Provider,
			Maintainers: b.Maintainers,
			Links:       b.Links,
			Maturity:    b.Maturity,
			Keywords:    b.Keywords,
			Icon:        b.Icon,
			InstallModes: []operatorsv1alpha1.InstallMode{
				{Type: operatorsv1alpha1.InstallModeTypeOwnNamespace, Supported: true},
				{Type: operatorsv1alpha1.InstallModeTypeSingleNamespace, Supported: true},
				{Type: operatorsv1alpha1.InstallModeTypeMultiNamespace, Supported: false},
				{Type: operatorsv1alpha1.InstallModeTypeAllNamespaces, Supported: false},
			},
		},
	}
}

// readClusterServiceVersionBase returns the ClusterServiceVersion base at path.
// If no base is found, readClusterServiceVersionBase returns an error.
func readClusterServiceVersionBase(path string) (*operatorsv1alpha1.ClusterServiceVersion, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	scanner := k8sutil.NewYAMLScanner(bytes.NewBuffer(b))
	for scanner.Scan() {
		manifest := scanner.Bytes()
		typeMeta, err := k8sutil.GetTypeMetaFromBytes(manifest)
		if err != nil {
			log.Debugf("Skipping non-Object manifest %s: %v", path, err)
			continue
		}
		if typeMeta.Kind == operatorsv1alpha1.ClusterServiceVersionKind {
			csv := &operatorsv1alpha1.ClusterServiceVersion{}
			if err := yaml.Unmarshal(manifest, csv); err != nil {
				return nil, fmt.Errorf("error unmarshalling ClusterServiceVersion from manifest %s: %w", path, err)
			}
			return csv, nil
		}
	}
	if err = scanner.Err(); err != nil {
		return nil, fmt.Errorf("error scanning manifest %s: %w", path, err)
	}

	return nil, fmt.Errorf("no ClusterServiceVersion manifest in %s", path)
}
