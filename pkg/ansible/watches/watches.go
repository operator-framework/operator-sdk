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

package watches

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"k8s.io/apimachinery/pkg/runtime/schema"

	yaml "gopkg.in/yaml.v2"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var log = logf.Log.WithName("watches")

// Watch - holds data used to create a mapping of GVK to ansible playbook or role.
// The mapping is used to compose an ansible operator.
type Watch struct {
	GroupVersionKind            schema.GroupVersionKind `yaml:",inline"`
	Playbook                    string                  `yaml:"playbook"`
	Role                        string                  `yaml:"role"`
	MaxRunnerArtifacts          int                     `yaml:"maxRunnerArtifacts"`
	ReconcilePeriod             time.Duration           `yaml:"reconcilePeriod"`
	ManageStatus                bool                    `yaml:"manageStatus"`
	WatchDependentResources     bool                    `yaml:"watchDependentResources"`
	WatchClusterScopedResources bool                    `yaml:"watchClusterScopedResources"`
	Finalizer                   *Finalizer              `yaml:"finalizer"`
}

// Finalizer - Expose finalizer to be used by a user.
type Finalizer struct {
	Name     string                 `yaml:"name"`
	Playbook string                 `yaml:"playbook"`
	Role     string                 `yaml:"role"`
	Vars     map[string]interface{} `yaml:"vars"`
}

// Default values for optional fields on Watch
const (
	ManageStatusDefault                bool          = true
	WatchDependentResourcesDefault     bool          = true
	MaxRunnerArtifactsDefault          int           = 20
	ReconcilePeriodDefault             string        = "0s"
	ReconcilePeriodDurationDefault     time.Duration = time.Duration(0)
	WatchClusterScopedResourcesDefault bool          = false
)

// UnmarshalYAML - implements the yaml.Unmarshaler interface for Watch
func (w *Watch) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// Use an alias struct to handle complex types
	type alias struct {
		Group                       string     `yaml:"group"`
		Version                     string     `yaml:"version"`
		Kind                        string     `yaml:"kind"`
		Playbook                    string     `yaml:"playbook"`
		Role                        string     `yaml:"role"`
		MaxRunnerArtifacts          int        `yaml:"maxRunnerArtifacts"`
		ReconcilePeriod             string     `yaml:"reconcilePeriod"`
		ManageStatus                bool       `yaml:"manageStatus"`
		WatchDependentResources     bool       `yaml:"watchDependentResources"`
		WatchClusterScopedResources bool       `yaml:"watchClusterScopedResources"`
		Finalizer                   *Finalizer `yaml:"finalizer"`
	}
	var tmp alias

	// by default, the operator will manage status and watch dependent resources
	// The operator will not manage cluster scoped resources by default.
	tmp.ManageStatus = ManageStatusDefault
	tmp.WatchDependentResources = WatchDependentResourcesDefault
	tmp.MaxRunnerArtifacts = MaxRunnerArtifactsDefault
	tmp.ReconcilePeriod = ReconcilePeriodDefault
	tmp.WatchClusterScopedResources = WatchClusterScopedResourcesDefault

	if err := unmarshal(&tmp); err != nil {
		return err
	}

	reconcilePeriod, err := time.ParseDuration(tmp.ReconcilePeriod)
	if err != nil {
		return fmt.Errorf("failed to parse '%s' to time.Duration: %v", tmp.ReconcilePeriod, err)
	}

	gvk := schema.GroupVersionKind{
		Group:   tmp.Group,
		Version: tmp.Version,
		Kind:    tmp.Kind,
	}
	err = verifyGVK(gvk)
	if err != nil {
		return fmt.Errorf("invalid GVK: %v - %s", gvk.String(), err)
	}

	// Rewrite values to struct being unmarshalled
	w.GroupVersionKind = gvk
	w.Playbook = tmp.Playbook
	w.Role = tmp.Role
	w.MaxRunnerArtifacts = tmp.MaxRunnerArtifacts
	w.ReconcilePeriod = reconcilePeriod
	w.ManageStatus = tmp.ManageStatus
	w.WatchDependentResources = tmp.WatchDependentResources
	w.WatchClusterScopedResources = tmp.WatchClusterScopedResources
	w.Finalizer = tmp.Finalizer

	return nil
}

// Load - loads a slice of Watches from the watch file at `path`.
func Load(path string) ([]Watch, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		log.Error(err, "Failed to get config file")
		return nil, err
	}

	watches := []Watch{}
	err = yaml.Unmarshal(b, &watches)
	if err != nil {
		log.Error(err, "Failed to unmarshal config")
		return nil, err
	}

	watchesMap := make(map[schema.GroupVersionKind]bool)
	for _, watch := range watches {

		// prevent dupes
		if _, ok := watchesMap[watch.GroupVersionKind]; ok {
			return nil, fmt.Errorf("duplicate GVK: %v", watch.GroupVersionKind.String())
		}
		watchesMap[watch.GroupVersionKind] = true

		err = verifyAnsiblePath(watch.Playbook, watch.Role)
		if err != nil {
			log.Error(err, fmt.Sprintf("Invalid ansible path for GVK: %v", watch.GroupVersionKind.String()))
			return nil, err
		}

		if watch.Finalizer != nil {
			if watch.Finalizer.Name == "" {
				err = fmt.Errorf("finalizer must have name")
				log.Error(err, fmt.Sprintf("Invalid finalizer for GVK: %v", watch.GroupVersionKind.String()))
				return nil, err
			}
			// only fail if Vars not set
			err = verifyAnsiblePath(watch.Finalizer.Playbook, watch.Finalizer.Role)
			if err != nil && len(watch.Finalizer.Vars) == 0 {
				log.Error(err, fmt.Sprintf("Invalid ansible path on Finalizer for GVK: %v", watch.GroupVersionKind.String()))
				return nil, err
			}
		}

	}

	return watches, nil
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

func verifyAnsiblePath(playbook string, role string) error {
	switch {
	case playbook != "":
		if !filepath.IsAbs(playbook) {
			return fmt.Errorf("playbook path must be absolute")
		}
		if _, err := os.Stat(playbook); err != nil {
			return fmt.Errorf("playbook: %v was not found", playbook)
		}
	case role != "":
		if !filepath.IsAbs(role) {
			return fmt.Errorf("role path must be absolute")
		}
		if _, err := os.Stat(role); err != nil {
			return fmt.Errorf("role path: %v was not found", role)
		}
	default:
		return fmt.Errorf("must specify Role or Playbook")
	}
	return nil
}
