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
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"text/template"

	"helm.sh/helm/v3/pkg/chartutil"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/yaml"

	"github.com/operator-framework/operator-sdk/internal/scaffold"
	"github.com/operator-framework/operator-sdk/internal/scaffold/helm"
	"github.com/operator-framework/operator-sdk/internal/util/fileutil"
)

const WatchesFile = "watches.yaml"

// Watch defines options for configuring a watch for a Helm-based
// custom resource.
type Watch struct {
	schema.GroupVersionKind `json:",inline"`
	ChartDir                string            `json:"chart"`
	WatchDependentResources *bool             `json:"watchDependentResources,omitempty"`
	OverrideValues          map[string]string `json:"overrideValues,omitempty"`
}

// UnmarshalYAML unmarshals an individual watch from the Helm watches.yaml file
// into a Watch struct.
//
// Deprecated: This function is no longer used internally to unmarshal
// watches.yaml data. To ensure the correct defaults are applied when loading
// watches.yaml, use Load() or LoadReader() instead of this function and/or
// yaml.Unmarshal().
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
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("could not open watches file: %w", err)
	}
	w, err := LoadReader(f)

	// Make sure to close the file, regardless of the error returned by
	// LoadReader.
	if err := f.Close(); err != nil {
		return nil, fmt.Errorf("could not close watches file: %w", err)
	}
	return w, err
}

// LoadReader loads a slice of Watches from the provided reader. For each entry
// in the watches file, it verifies the configuration. If an error is
// encountered reading or verifying the configuration, it will be returned.
func LoadReader(reader io.Reader) ([]Watch, error) {
	b, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	watches := []Watch{}
	err = yaml.Unmarshal(b, &watches)
	if err != nil {
		return nil, err
	}

	watchesMap := make(map[schema.GroupVersionKind]struct{})
	for i, w := range watches {
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
		watchesMap[gvk] = struct{}{}
		if w.WatchDependentResources == nil {
			trueVal := true
			w.WatchDependentResources = &trueVal
		}
		w.OverrideValues = expandOverrideEnvs(w.OverrideValues)
		watches[i] = w
	}
	return watches, nil
}

// Append reads watches.yaml data from the provided reader, verifies that the
// provided watch is valid and unique, and returns a buffer containing the new
// watch appended to the end of the existing watches.yaml data. If an error
// occurs, it will be returned.
func Append(r io.Reader, watch Watch) ([]byte, error) {
	if err := verifyGVK(watch.GroupVersionKind); err != nil {
		return nil, fmt.Errorf("invalid GVK %s: %w", watch.GroupVersionKind, err)
	}

	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read watches data: %w", err)
	}

	watches := []Watch{}
	err = yaml.Unmarshal(b, &watches)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal watches: %w", err)
	}

	for _, w := range watches {
		if w.GroupVersionKind == watch.GroupVersionKind {
			return nil, fmt.Errorf("duplicate GVK: %s", watch.GroupVersionKind)
		}
	}

	out := &bytes.Buffer{}
	if len(b) == 0 {
		if _, err := out.Write([]byte("---\n")); err != nil {
			return nil, fmt.Errorf("write error: %w", err)
		}
	} else if _, err := out.Write(b); err != nil {
		return nil, fmt.Errorf("write error: %w", err)
	}

	t := template.Must(template.New("watches.yaml").Parse(watchTemplate))
	if err := t.Execute(out, watch); err != nil {
		return nil, fmt.Errorf("failed to template new watch: %w", err)
	}
	return out.Bytes(), nil
}

// UpdateForResource appends a new watch to the provided watches.yaml file
// based on the provided resource. The watch is validated and must be unique.
// If an error occurs, it is returned.
func UpdateForResource(path string, r *scaffold.Resource, chartName string) (err error) {
	gvk := schema.GroupVersionKind{
		Group:   r.FullGroup,
		Version: r.Version,
		Kind:    r.Kind,
	}
	watch := Watch{
		GroupVersionKind: gvk,
		ChartDir:         filepath.Join(helm.HelmChartsDir, chartName),
	}

	f, err := ioutil.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("failed to read watches.yaml file: %w", err)
	}

	data, err := Append(bytes.NewReader(f), watch)
	if err != nil {
		return fmt.Errorf("failed to generate new watches.yaml file: %w", err)
	}

	return ioutil.WriteFile(path, data, fileutil.DefaultFileMode)
}

func expandOverrideEnvs(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
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

const watchTemplate = `- group: {{.Group}}
  version: {{.Version}}
  kind: {{.Kind}}
  chart: {{.ChartDir}}
{{- with .WatchDependentResources }}
  watchDependentResources: {{ . }}
{{- end}}{{ with .OverrideValues }}
  overrideValues:
{{- range $key, $value := . }}
    "{{ $key }}": "{{ $value }}"{{end}}{{end}}
`
