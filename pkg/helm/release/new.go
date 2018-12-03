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

package release

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	yaml "gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/kube"
	"k8s.io/helm/pkg/storage"
)

const (
	// HelmChartWatchesEnvVar is the environment variable for a YAML
	// configuration file containing mappings of GVKs to helm charts. Use of
	// this environment variable overrides the watch configuration provided
	// by API_VERSION, KIND, and HELM_CHART, and it allows users to configure
	// multiple watches, each with a different chart.
	HelmChartWatchesEnvVar = "HELM_CHART_WATCHES"

	// APIVersionEnvVar is the environment variable for the group and version
	// to be watched using the format `<group>/<version>`
	// (e.g. "example.com/v1alpha1").
	APIVersionEnvVar = "API_VERSION"

	// KindEnvVar is the environment variable for the kind to be watched. The
	// value is typically singular and should be CamelCased (e.g. "MyApp").
	KindEnvVar = "KIND"

	// HelmChartEnvVar is the environment variable for the directory location
	// of the helm chart to be installed for CRs that match the values for the
	// API_VERSION and KIND environment variables.
	HelmChartEnvVar = "HELM_CHART"

	defaultHelmChartWatchesFile = "/opt/helm/watches.yaml"
)

type watch struct {
	Group   string `yaml:"group"`
	Version string `yaml:"version"`
	Kind    string `yaml:"kind"`
	Chart   string `yaml:"chart"`
}

// NewManagerFactory returns a new Helm manager factory capable of installing and uninstalling releases.
func NewManagerFactory(storageBackend *storage.Storage, tillerKubeClient *kube.Client, chartDir string) ManagerFactory {
	return &managerFactory{storageBackend, tillerKubeClient, chartDir}
}

// newManagerFactoryFromEnv returns a GVK and manager factory based on configuration provided
// in the environment.
func newManagerFactoryFromEnv(storageBackend *storage.Storage, tillerKubeClient *kube.Client) (schema.GroupVersionKind, ManagerFactory, error) {
	apiVersion := os.Getenv(APIVersionEnvVar)
	kind := os.Getenv(KindEnvVar)
	chartDir := os.Getenv(HelmChartEnvVar)

	var gvk schema.GroupVersionKind
	gv, err := schema.ParseGroupVersion(apiVersion)
	if err != nil {
		return gvk, nil, err
	}
	gvk = gv.WithKind(kind)

	if err := verifyGVK(gvk); err != nil {
		return gvk, nil, fmt.Errorf("invalid GVK: %s: %s", gvk, err)
	}

	if _, err := chartutil.IsChartDir(chartDir); err != nil {
		return gvk, nil, fmt.Errorf("invalid chart directory %s: %s", chartDir, err)
	}

	managerFactory := NewManagerFactory(storageBackend, tillerKubeClient, chartDir)
	return gvk, managerFactory, nil
}

// NewManagerFactoriesFromEnv returns a map of managers, keyed by GVK, based on
// configuration provided in the environment.
func NewManagerFactoriesFromEnv(storageBackend *storage.Storage, tillerKubeClient *kube.Client) (map[schema.GroupVersionKind]ManagerFactory, error) {
	if watchesFile, ok := getWatchesFile(); ok {
		return NewManagerFactoriesFromFile(storageBackend, tillerKubeClient, watchesFile)
	}
	gvk, managerFactory, err := newManagerFactoryFromEnv(storageBackend, tillerKubeClient)
	if err != nil {
		return nil, err
	}
	return map[schema.GroupVersionKind]ManagerFactory{gvk: managerFactory}, nil
}

// NewManagerFactoriesFromFile reads the config file at the provided path and returns a map
// of managers, keyed by each GVK in the config.
func NewManagerFactoriesFromFile(storageBackend *storage.Storage, tillerKubeClient *kube.Client, path string) (map[schema.GroupVersionKind]ManagerFactory, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %s", err)
	}
	watches := []watch{}
	err = yaml.Unmarshal(b, &watches)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %s", err)
	}

	m := map[schema.GroupVersionKind]ManagerFactory{}
	for _, w := range watches {
		gvk := schema.GroupVersionKind{
			Group:   w.Group,
			Version: w.Version,
			Kind:    w.Kind,
		}

		if err := verifyGVK(gvk); err != nil {
			return nil, fmt.Errorf("invalid GVK: %s: %s", gvk, err)
		}

		if _, err := chartutil.IsChartDir(w.Chart); err != nil {
			return nil, fmt.Errorf("invalid chart directory %s: %s", w.Chart, err)
		}

		if _, ok := m[gvk]; ok {
			return nil, fmt.Errorf("duplicate GVK: %s", gvk)
		}
		m[gvk] = NewManagerFactory(storageBackend, tillerKubeClient, w.Chart)
	}
	return m, nil
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

func getWatchesFile() (string, bool) {
	// If the watches env variable is set (even if it's an empty string), use it
	// since the user explicitly set it.
	if watchesFile, ok := os.LookupEnv(HelmChartWatchesEnvVar); ok {
		return watchesFile, true
	}

	// Next, check if the default watches file is present. If so, use it.
	if _, err := os.Stat(defaultHelmChartWatchesFile); err == nil {
		return defaultHelmChartWatchesFile, true
	}
	return "", false
}
