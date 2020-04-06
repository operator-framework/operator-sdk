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

package watches

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	// todo(camila): replace it for yaml "sigs.k8s.io/yaml"
	// See that the unmarshaling JSON will be affected
	yaml "gopkg.in/yaml.v3"

	"helm.sh/helm/v3/pkg/chartutil"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Watch defines options for configuring a watch for a Helm-based
// custom resource.
type Watch struct {
	GroupVersionKind        schema.GroupVersionKind `yaml:",inline"`
	ChartDir                string                  `yaml:"chart"`
	WatchDependentResources *bool                   `yaml:"watchDependentResources,omitempty"`
	OverrideValues          map[string]string       `yaml:"overrideValues,omitempty"`
}

func (w *Watch) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// by default, the operator will watch dependent resources
	trueVal := true
	w.WatchDependentResources = &trueVal

	// hide watch data in plain struct to prevent unmarshal from calling
	// UnmarshalYAML again
	type plain Watch

	return unmarshal((*plain)(w))
}

// Load loads a slice of Watches from the watch file at `path`. For each entry
// in the watches file, it verifies the configuration. If an error is
// encountered loading the file or verifying the configuration, it will be
// returned.
func Load(path string) ([]Watch, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	yamlWatches := []Watch{}
	err = yaml.Unmarshal(b, &yamlWatches)
	if err != nil {
		return nil, err
	}

	watches := []Watch{}
	watchesMap := make(map[schema.GroupVersionKind]Watch)
	for _, w := range yamlWatches {
		gvk := w.GroupVersionKind

		if err := verifyGVK(gvk); err != nil {
			return nil, fmt.Errorf("invalid GVK: %s: %w", gvk, err)
		}

		if _, err := chartutil.IsChartDir(w.ChartDir); err != nil {
			return nil, fmt.Errorf("invalid chart directory %s: %w", w.ChartDir, err)
		}

		if _, ok := watchesMap[gvk]; ok {
			return nil, fmt.Errorf("duplicate GVK: %s", gvk)
		}

		watch := Watch{
			GroupVersionKind:        gvk,
			ChartDir:                w.ChartDir,
			WatchDependentResources: w.WatchDependentResources,
			OverrideValues:          expandOverrideEnvs(w.OverrideValues),
		}
		watchesMap[gvk] = watch
		watches = append(watches, watch)
	}
	return watches, nil
}

func expandOverrideEnvs(in map[string]string) map[string]string {
	out := make(map[string]string)
	for k, v := range in {
		out[k] = os.ExpandEnv(v)
	}
	return out
}

func verifyGVK(gvk schema.GroupVersionKind) error {
	// A GVK without a group is valid. Certain scenarios may cause a GVK
	// without a group to fail in other ways later in the initialization
	// process.
	if gvk.Version == "" {
		return errors.New("version must not be empty")
	}
	if gvk.Kind == "" {
		return errors.New("kind must not be empty")
	}
	return nil
}
