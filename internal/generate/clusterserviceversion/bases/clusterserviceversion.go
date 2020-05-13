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

	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/yaml"

	"github.com/operator-framework/operator-sdk/internal/util/k8sutil"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"
)

// ClusterServiceVersion configures the v1alpha1.ClusterServiceVersion
// that GetBase() returns.
type ClusterServiceVersion struct {
	// BasePath is the path to the base being read. If empty, GetBase() returns
	// a default base.
	BasePath string
	// OperatorName is the operator's name, ex. app-operator
	OperatorName string
	// OperatorType
	OperatorType projutil.OperatorType
	// APIsDir contains project API definition files.
	APIsDir string
	// GVKs are all GroupVersionKinds in the project.
	GVKs []schema.GroupVersionKind
	// Interactive turns on an interactive prompt.
	Interactive bool

	// Fields for input to the base.
	DisplayName  string
	Description  string
	Maturity     string
	Capabilities string
	Keywords     []string
	Provider     v1alpha1.AppLink
	Links        []v1alpha1.AppLink
	Maintainers  []v1alpha1.Maintainer
	Icon         []v1alpha1.Icon // TODO(estroz): read icon bytes from files.
}

// GetBase returns a base v1alpha1.ClusterServiceVersion, populated
// either with default values or, if b.BasePath is set, bytes from disk.
func (b ClusterServiceVersion) GetBase() (base *v1alpha1.ClusterServiceVersion, err error) {
	if b.BasePath != "" {
		if base, err = readClusterServiceVersionBase(b.BasePath); err != nil {
			return nil, fmt.Errorf("error reading existing ClusterServiceVersion base %s: %v", b.BasePath, err)
		}
	} else {
		b.setDefaults()
		base = b.makeNewBase()
	}

	// Interactively fill in UI metadata.
	if b.Interactive {
		meta := &uiMetadata{}
		meta.runInteractivePrompt()
		meta.apply(base)
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

// setDefaults sets default values in b using b's existing values.
func (b *ClusterServiceVersion) setDefaults() {
	if b.DisplayName == "" {
		b.DisplayName = k8sutil.GetDisplayName(b.OperatorName)
	}
	if b.Description == "" {
		b.Description = b.DisplayName + " description. TODO."
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
	if len(b.Links) == 0 || b.Links[0] == (v1alpha1.AppLink{}) {
		b.Links = []v1alpha1.AppLink{
			{
				Name: b.DisplayName,
				URL:  fmt.Sprintf("https://%s.domain", b.OperatorName),
			},
		}
	}
	if len(b.Icon) == 0 {
		b.Icon = make([]v1alpha1.Icon, 1)
	}
	if b.Provider == (v1alpha1.AppLink{}) {
		b.Provider = v1alpha1.AppLink{
			Name: "Provider Name",
			URL:  "https://your.domain",
		}
	}
	if len(b.Maintainers) == 0 || b.Maintainers[0] == (v1alpha1.Maintainer{}) {
		b.Maintainers = []v1alpha1.Maintainer{
			{
				Name:  "Maintainer Name",
				Email: "your@email.com",
			},
		}
	}
}

// makeNewBase returns a base v1alpha1.ClusterServiceVersion to modify.
func (b ClusterServiceVersion) makeNewBase() *v1alpha1.ClusterServiceVersion {
	return &v1alpha1.ClusterServiceVersion{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.ClusterServiceVersionAPIVersion,
			Kind:       v1alpha1.ClusterServiceVersionKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      b.OperatorName + ".vX.Y.Z",
			Namespace: "placeholder",
			Annotations: map[string]string{
				"capabilities": b.Capabilities,
				"alm-examples": "[]",
			},
		},
		Spec: v1alpha1.ClusterServiceVersionSpec{
			DisplayName: b.DisplayName,
			Description: b.Description,
			Provider:    b.Provider,
			Maintainers: b.Maintainers,
			Links:       b.Links,
			Maturity:    b.Maturity,
			Keywords:    b.Keywords,
			Icon:        b.Icon,
			InstallModes: []v1alpha1.InstallMode{
				{Type: v1alpha1.InstallModeTypeOwnNamespace, Supported: true},
				{Type: v1alpha1.InstallModeTypeSingleNamespace, Supported: true},
				{Type: v1alpha1.InstallModeTypeMultiNamespace, Supported: false},
				{Type: v1alpha1.InstallModeTypeAllNamespaces, Supported: true},
			},
		},
	}
}

// readClusterServiceVersionBase returns the ClusterServiceVersion base at path.
// If no base is found, readClusterServiceVersionBase returns an error.
func readClusterServiceVersionBase(path string) (*v1alpha1.ClusterServiceVersion, error) {
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
		if typeMeta.Kind == v1alpha1.ClusterServiceVersionKind {
			csv := &v1alpha1.ClusterServiceVersion{}
			if err := yaml.Unmarshal(manifest, csv); err != nil {
				return nil, fmt.Errorf("error unmarshalling ClusterServiceVersion from manifest %s: %v", path, err)
			}
			return csv, nil
		}
	}
	if err = scanner.Err(); err != nil {
		return nil, fmt.Errorf("error scanning manifest %s: %v", path, err)
	}

	return nil, fmt.Errorf("no ClusterServiceVersion manifest in %s", path)
}
