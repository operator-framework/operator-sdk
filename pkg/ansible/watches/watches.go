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

// Package watches provides the structures and functions for mapping a
// GroupVersionKind to an Ansible playbook or role.
package watches

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/runtime/schema"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	// todo(camila): replace it for yaml "sigs.k8s.io/yaml"
	// See that the unmarshaling JSON will be affected
	yaml "gopkg.in/yaml.v3"

	"github.com/operator-framework/operator-sdk/internal/util/projutil"
)

var log = logf.Log.WithName("watches")

// Watch - holds data used to create a mapping of GVK to ansible playbook or role.
// The mapping is used to compose an ansible operator.
type Watch struct {
	GroupVersionKind            schema.GroupVersionKind   `yaml:",inline"`
	Blacklist                   []schema.GroupVersionKind `yaml:"blacklist"`
	Playbook                    string                    `yaml:"playbook"`
	Role                        string                    `yaml:"role"`
	Vars                        map[string]interface{}    `yaml:"vars"`
	MaxRunnerArtifacts          int                       `yaml:"maxRunnerArtifacts"`
	ReconcilePeriod             time.Duration             `yaml:"reconcilePeriod"`
	Finalizer                   *Finalizer                `yaml:"finalizer"`
	ManageStatus                bool                      `yaml:"manageStatus"`
	WatchDependentResources     bool                      `yaml:"watchDependentResources"`
	WatchClusterScopedResources bool                      `yaml:"watchClusterScopedResources"`

	// Not configurable via watches.yaml
	MaxWorkers       int `yaml:"-"`
	AnsibleVerbosity int `yaml:"-"`
}

// Finalizer - Expose finalizer to be used by a user.
type Finalizer struct {
	Name     string                 `yaml:"name"`
	Playbook string                 `yaml:"playbook"`
	Role     string                 `yaml:"role"`
	Vars     map[string]interface{} `yaml:"vars"`
}

// Default values for optional fields on Watch
var (
	blacklistDefault                   = []schema.GroupVersionKind{}
	maxRunnerArtifactsDefault          = 20
	reconcilePeriodDefault             = "0s"
	manageStatusDefault                = true
	watchDependentResourcesDefault     = true
	watchClusterScopedResourcesDefault = false

	// these are overridden by cmdline flags
	maxWorkersDefault       = 1
	ansibleVerbosityDefault = 2
)

// UnmarshalYAML - implements the yaml.Unmarshaler interface for Watch.
// This makes it possible to verify, when loading, that the GroupVersionKind
// specified for a given watch is valid as well as provide sensible defaults
// for values that were omitted.
func (w *Watch) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// Use an alias struct to handle complex types
	type alias struct {
		Group                       string                    `yaml:"group"`
		Version                     string                    `yaml:"version"`
		Kind                        string                    `yaml:"kind"`
		Playbook                    string                    `yaml:"playbook"`
		Role                        string                    `yaml:"role"`
		Vars                        map[string]interface{}    `yaml:"vars"`
		MaxRunnerArtifacts          int                       `yaml:"maxRunnerArtifacts"`
		ReconcilePeriod             string                    `yaml:"reconcilePeriod"`
		ManageStatus                bool                      `yaml:"manageStatus"`
		WatchDependentResources     bool                      `yaml:"watchDependentResources"`
		WatchClusterScopedResources bool                      `yaml:"watchClusterScopedResources"`
		Blacklist                   []schema.GroupVersionKind `yaml:"blacklist"`
		Finalizer                   *Finalizer                `yaml:"finalizer"`
	}
	var tmp alias

	// by default, the operator will manage status and watch dependent resources
	tmp.ManageStatus = manageStatusDefault
	// the operator will not manage cluster scoped resources by default.
	tmp.WatchDependentResources = watchDependentResourcesDefault
	tmp.MaxRunnerArtifacts = maxRunnerArtifactsDefault
	tmp.ReconcilePeriod = reconcilePeriodDefault
	tmp.WatchClusterScopedResources = watchClusterScopedResourcesDefault
	tmp.Blacklist = blacklistDefault

	if err := unmarshal(&tmp); err != nil {
		return err
	}

	reconcilePeriod, err := time.ParseDuration(tmp.ReconcilePeriod)
	if err != nil {
		return fmt.Errorf("failed to parse '%s' to time.Duration: %w", tmp.ReconcilePeriod, err)
	}

	gvk := schema.GroupVersionKind{
		Group:   tmp.Group,
		Version: tmp.Version,
		Kind:    tmp.Kind,
	}
	err = verifyGVK(gvk)
	if err != nil {
		return fmt.Errorf("invalid GVK: %s: %w", gvk, err)
	}

	// Rewrite values to struct being unmarshalled
	w.GroupVersionKind = gvk
	w.Playbook = tmp.Playbook
	w.Role = tmp.Role
	w.Vars = tmp.Vars
	w.MaxRunnerArtifacts = tmp.MaxRunnerArtifacts
	w.MaxWorkers = getMaxWorkers(gvk, maxWorkersDefault)
	w.ReconcilePeriod = reconcilePeriod
	w.ManageStatus = tmp.ManageStatus
	w.WatchDependentResources = tmp.WatchDependentResources
	w.WatchClusterScopedResources = tmp.WatchClusterScopedResources
	w.Finalizer = tmp.Finalizer
	w.AnsibleVerbosity = getAnsibleVerbosity(gvk, ansibleVerbosityDefault)
	w.Blacklist = tmp.Blacklist
	w.addRolePlaybookPaths()

	return nil
}

// addRolePlaybookPaths will add the full path based on the current dir
func (w *Watch) addRolePlaybookPaths() {
	if len(w.Playbook) > 0 {
		w.Playbook = getFullPath(w.Playbook)
	}

	if len(w.Role) > 0 {
		possibleRolePaths := getPossibleRolePaths(w.Role)
		for _, possiblePath := range possibleRolePaths {
			if _, err := os.Stat(possiblePath); err == nil {
				w.Role = possiblePath
				break
			}
		}
	}
	if w.Finalizer != nil && len(w.Finalizer.Role) > 0 {
		possibleRolePaths := getPossibleRolePaths(w.Finalizer.Role)
		for _, possiblePath := range possibleRolePaths {
			if _, err := os.Stat(possiblePath); err == nil {
				w.Finalizer.Role = possiblePath
				break
			}
		}
	}
	if w.Finalizer != nil && len(w.Finalizer.Playbook) > 0 {
		w.Finalizer.Playbook = getFullPath(w.Finalizer.Playbook)
	}
}

// getFullPath returns an absolute path for the playbook
func getFullPath(path string) string {
	if len(path) > 0 && !filepath.IsAbs(path) {
		return filepath.Join(projutil.MustGetwd(), path)
	}
	return path
}

// getPossibleRolePaths returns list of possible absolute paths derived from a user provided value.
func getPossibleRolePaths(path string) []string {
	possibleRolePaths := []string{}
	if filepath.IsAbs(path) || len(path) == 0 {
		return append(possibleRolePaths, path)
	}
	fqcn := strings.Split(path, ".")
	// If fqcn is a valid fully qualified collection name, it is <namespace>.<collectionName>.<roleName>
	if len(fqcn) == 3 {
		ansibleCollectionsPathEnv, ok := os.LookupEnv("ANSIBLE_COLLECTIONS_PATH")
		if !ok || len(ansibleCollectionsPathEnv) == 0 {
			ansibleCollectionsPathEnv = "/usr/share/ansible/collections"
			home, err := os.UserHomeDir()
			if err == nil {
				homeCollections := filepath.Join(home, ".ansible/collections")
				ansibleCollectionsPathEnv = ansibleCollectionsPathEnv + ":" + homeCollections
			}
		}
		for _, possiblePathParent := range strings.Split(ansibleCollectionsPathEnv, ":") {
			possiblePath := filepath.Join(possiblePathParent, "ansible_collections", fqcn[0], fqcn[1], "roles", fqcn[2])
			possibleRolePaths = append(possibleRolePaths, possiblePath)
		}
	}

	// Check for the role where Ansible would. If it exists, use it.
	ansibleRolesPathEnv, ok := os.LookupEnv("ANSIBLE_ROLES_PATH")
	if ok && len(ansibleRolesPathEnv) > 0 {
		for _, possiblePathParent := range strings.Split(ansibleRolesPathEnv, ":") {
			// "roles" is optionally a part of the path. Check with, and without.
			possibleRolePaths = append(possibleRolePaths, filepath.Join(possiblePathParent, path))
			possibleRolePaths = append(possibleRolePaths, filepath.Join(possiblePathParent, "roles", path))
		}
	}
	// Roles can also live in the current working directory.
	return append(possibleRolePaths, getFullPath(filepath.Join("roles", path)))
}

// Validate - ensures that a Watch is valid
// A Watch is considered valid if it:
// - Specifies a valid path to a Role||Playbook
// - If a Finalizer is non-nil, it must have a name + valid path to a Role||Playbook or Vars
func (w *Watch) Validate() error {
	err := verifyAnsiblePath(w.Playbook, w.Role)
	if err != nil {
		log.Error(err, fmt.Sprintf("Invalid ansible path for GVK: %v", w.GroupVersionKind.String()))
		return err
	}

	if w.Finalizer != nil {
		if w.Finalizer.Name == "" {
			err = fmt.Errorf("finalizer must have name")
			log.Error(err, fmt.Sprintf("Invalid finalizer for GVK: %v", w.GroupVersionKind.String()))
			return err
		}
		// only fail if Vars not set
		err = verifyAnsiblePath(w.Finalizer.Playbook, w.Finalizer.Role)
		if err != nil && len(w.Finalizer.Vars) == 0 {
			log.Error(err, fmt.Sprintf("Invalid ansible path on Finalizer for GVK: %v",
				w.GroupVersionKind.String()))
			return err
		}
	}

	return nil
}

// New - returns a Watch with sensible defaults.
func New(gvk schema.GroupVersionKind, role, playbook string, vars map[string]interface{}, finalizer *Finalizer) *Watch {
	reconcilePeriod, _ := time.ParseDuration(reconcilePeriodDefault)
	return &Watch{
		Blacklist:                   blacklistDefault,
		GroupVersionKind:            gvk,
		Playbook:                    playbook,
		Role:                        role,
		Vars:                        vars,
		MaxRunnerArtifacts:          maxRunnerArtifactsDefault,
		MaxWorkers:                  maxWorkersDefault,
		ReconcilePeriod:             reconcilePeriod,
		ManageStatus:                manageStatusDefault,
		WatchDependentResources:     watchDependentResourcesDefault,
		WatchClusterScopedResources: watchClusterScopedResourcesDefault,
		Finalizer:                   finalizer,
		AnsibleVerbosity:            ansibleVerbosityDefault,
	}
}

// Load - loads a slice of Watches from the watches file from the CLI
func Load(path string, maxWorkers, ansibleVerbosity int) ([]Watch, error) {
	maxWorkersDefault = maxWorkers
	ansibleVerbosityDefault = ansibleVerbosity
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

		err = watch.Validate()
		if err != nil {
			log.Error(err, fmt.Sprintf("Watch with GVK %v failed validation", watch.GroupVersionKind.String()))
			return nil, err
		}
	}

	return watches, nil
}

// verify that a given GroupVersionKind has a Version and Kind
// A GVK without a group is valid. Certain scenarios may cause a GVK
// without a group to fail in other ways later in the initialization
// process.
func verifyGVK(gvk schema.GroupVersionKind) error {
	if gvk.Version == "" {
		return errors.New("version must not be empty")
	}
	if gvk.Kind == "" {
		return errors.New("kind must not be empty")
	}
	return nil
}

// verify that a valid path is specified for a given role or playbook
func verifyAnsiblePath(playbook string, role string) error {
	switch {
	case playbook != "":
		if _, err := os.Stat(playbook); err != nil {
			return fmt.Errorf("playbook: %v was not found", playbook)
		}
	case role != "":
		if _, err := os.Stat(role); err != nil {
			return fmt.Errorf("role: %v was not found", role)
		}
	default:
		return fmt.Errorf("must specify Role or Playbook")
	}
	return nil
}

// if the WORKER_* environment variable is set, use that value.
// Otherwise, use defValue. This is definitely
// counter-intuitive but it allows the operator admin adjust the
// number of workers based on their cluster resources. While the
// author may use the CLI option to specify a suggested
// configuration for the operator.
func getMaxWorkers(gvk schema.GroupVersionKind, defValue int) int {
	envVar := strings.ToUpper(strings.Replace(
		fmt.Sprintf("WORKER_%s_%s", gvk.Kind, gvk.Group),
		".",
		"_",
		-1,
	))
	maxWorkers := getIntegerEnvWithDefault(envVar, defValue)
	if maxWorkers <= 0 {
		log.Info("Value %v not valid. Using default %v", maxWorkers, defValue)
		return defValue
	}
	return maxWorkers
}

// if the ANSIBLE_VERBOSITY_* environment variable is set, use that value.
// Otherwise, use defValue.
func getAnsibleVerbosity(gvk schema.GroupVersionKind, defValue int) int {
	envVar := strings.ToUpper(strings.Replace(
		fmt.Sprintf("ANSIBLE_VERBOSITY_%s_%s", gvk.Kind, gvk.Group),
		".",
		"_",
		-1,
	))
	ansibleVerbosity := getIntegerEnvWithDefault(envVar, defValue)
	// Use default value when value doesn't make sense
	if ansibleVerbosity < 0 {
		log.Info("Value %v not valid. Using default %v", ansibleVerbosity, defValue)
		return defValue
	}
	if ansibleVerbosity > 7 {
		log.Info("Value %v not valid. Using default %v", ansibleVerbosity, defValue)
		return defValue
	}
	return ansibleVerbosity
}

// getIntegerEnvWithDefault returns value for MaxWorkers/Ansibleverbosity based on if envVar is set
// sor a defvalue is used.
func getIntegerEnvWithDefault(envVar string, defValue int) int {
	val := defValue
	if envVal, ok := os.LookupEnv(envVar); ok {
		if i, err := strconv.Atoi(envVal); err != nil {
			log.Info("Could not parse environment variable as an integer; using default value",
				"envVar", envVar, "default", defValue)
		} else {
			val = i
		}
	} else if !ok {
		log.Info("Environment variable not set; using default value", "envVar", envVar,
			"default", defValue)
	}
	return val
}
