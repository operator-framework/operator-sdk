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

package runner

import (
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/operator-framework/operator-sdk/pkg/ansible/paramconv"
	"github.com/operator-framework/operator-sdk/pkg/ansible/runner/eventapi"
	"github.com/operator-framework/operator-sdk/pkg/ansible/runner/internal/inputdir"
	"github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Runner - a runnable that should take the parameters and name and namespace
// and run the correct code.
type Runner interface {
	Run(*unstructured.Unstructured, string) (chan eventapi.JobEvent, error)
	GetFinalizer() (string, bool)
	GetReconcilePeriod() (time.Duration, bool)
}

// watch holds data used to create a mapping of GVK to ansible playbook or role.
// The mapping is used to compose an ansible operator.
type watch struct {
	Version         string     `yaml:"version"`
	Group           string     `yaml:"group"`
	Kind            string     `yaml:"kind"`
	Playbook        string     `yaml:"playbook"`
	Role            string     `yaml:"role"`
	ReconcilePeriod string     `yaml:"reconcilePeriod"`
	Finalizer       *Finalizer `yaml:"finalizer"`
}

// Finalizer - Expose finalizer to be used by a user.
type Finalizer struct {
	Name     string                 `yaml:"name"`
	Playbook string                 `yaml:"playbook"`
	Role     string                 `yaml:"role"`
	Vars     map[string]interface{} `yaml:"vars"`
}

// NewFromWatches reads the operator's config file at the provided path.
func NewFromWatches(path string) (map[schema.GroupVersionKind]Runner, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		logrus.Errorf("failed to get config file %v", err)
		return nil, err
	}
	watches := []watch{}
	err = yaml.Unmarshal(b, &watches)
	if err != nil {
		logrus.Errorf("failed to unmarshal config %v", err)
		return nil, err
	}

	m := map[schema.GroupVersionKind]Runner{}
	for _, w := range watches {
		s := schema.GroupVersionKind{
			Group:   w.Group,
			Version: w.Version,
			Kind:    w.Kind,
		}
		var reconcilePeriod time.Duration
		if w.ReconcilePeriod != "" {
			d, err := time.ParseDuration(w.ReconcilePeriod)
			if err != nil {
				return nil, fmt.Errorf("unable to parse duration: %v - %v, setting to default", w.ReconcilePeriod, err)
			}
			reconcilePeriod = d
		}

		// Check if schema is a duplicate
		if _, ok := m[s]; ok {
			return nil, fmt.Errorf("duplicate GVK: %v", s.String())
		}
		switch {
		case w.Playbook != "":
			r, err := NewForPlaybook(w.Playbook, s, w.Finalizer, reconcilePeriod)
			if err != nil {
				return nil, err
			}
			m[s] = r
		case w.Role != "":
			r, err := NewForRole(w.Role, s, w.Finalizer, reconcilePeriod)
			if err != nil {
				return nil, err
			}
			m[s] = r
		default:
			return nil, fmt.Errorf("either playbook or role must be defined for %v", s)
		}
	}
	return m, nil
}

// NewForPlaybook returns a new Runner based on the path to an ansible playbook.
func NewForPlaybook(path string, gvk schema.GroupVersionKind, finalizer *Finalizer, reconcilePeriod time.Duration) (Runner, error) {
	if !filepath.IsAbs(path) {
		return nil, fmt.Errorf("playbook path must be absolute for %v", gvk)
	}
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("playbook: %v was not found for %v", path, gvk)
	}
	r := &runner{
		Path: path,
		GVK:  gvk,
		cmdFunc: func(ident, inputDirPath string) *exec.Cmd {
			return exec.Command("ansible-runner", "-vv", "-p", path, "-i", ident, "run", inputDirPath)
		},
		reconcilePeriod: reconcilePeriod,
	}
	err := r.addFinalizer(finalizer)
	if err != nil {
		return nil, err
	}
	return r, nil
}

// NewForRole returns a new Runner based on the path to an ansible role.
func NewForRole(path string, gvk schema.GroupVersionKind, finalizer *Finalizer, reconcilePeriod time.Duration) (Runner, error) {
	if !filepath.IsAbs(path) {
		return nil, fmt.Errorf("role path must be absolute for %v", gvk)
	}
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("role path: %v was not found for %v", path, gvk)
	}
	path = strings.TrimRight(path, "/")
	r := &runner{
		Path: path,
		GVK:  gvk,
		cmdFunc: func(ident, inputDirPath string) *exec.Cmd {
			rolePath, roleName := filepath.Split(path)
			return exec.Command("ansible-runner", "-vv", "--role", roleName, "--roles-path", rolePath, "--hosts", "localhost", "-i", ident, "run", inputDirPath)
		},
		reconcilePeriod: reconcilePeriod,
	}
	err := r.addFinalizer(finalizer)
	if err != nil {
		return nil, err
	}
	return r, nil
}

// runner - implements the Runner interface for a GVK that's being watched.
type runner struct {
	Path             string                  // path on disk to a playbook or role depending on what cmdFunc expects
	GVK              schema.GroupVersionKind // GVK being watched that corresponds to the Path
	Finalizer        *Finalizer
	cmdFunc          func(ident, inputDirPath string) *exec.Cmd // returns a Cmd that runs ansible-runner
	finalizerCmdFunc func(ident, inputDirPath string) *exec.Cmd
	reconcilePeriod  time.Duration
}

func (r *runner) Run(u *unstructured.Unstructured, kubeconfig string) (chan eventapi.JobEvent, error) {
	if u.GetDeletionTimestamp() != nil && !r.isFinalizerRun(u) {
		return nil, errors.New("resource has been deleted, but no finalizer was matched, skipping reconciliation")
	}
	ident := strconv.Itoa(rand.Int())
	logger := logrus.WithFields(logrus.Fields{
		"component": "runner",
		"job":       ident,
		"name":      u.GetName(),
		"namespace": u.GetNamespace(),
	})
	// start the event receiver. We'll check errChan for an error after
	// ansible-runner exits.
	errChan := make(chan error, 1)
	receiver, err := eventapi.New(ident, errChan)
	if err != nil {
		return nil, err
	}
	inputDir := inputdir.InputDir{
		Path:       filepath.Join("/tmp/ansible-operator/runner/", r.GVK.Group, r.GVK.Version, r.GVK.Kind, u.GetNamespace(), u.GetName()),
		Parameters: r.makeParameters(u),
		EnvVars: map[string]string{
			"K8S_AUTH_KUBECONFIG": kubeconfig,
		},
		Settings: map[string]string{
			"runner_http_url":  receiver.SocketPath,
			"runner_http_path": receiver.URLPath,
		},
	}
	// If Path is a dir, assume it is a role path. Otherwise assume it's a
	// playbook path
	fi, err := os.Lstat(r.Path)
	if err != nil {
		return nil, err
	}
	if !fi.IsDir() {
		inputDir.PlaybookPath = r.Path
	}
	err = inputDir.Write()
	if err != nil {
		return nil, err
	}

	go func() {
		var dc *exec.Cmd
		if r.isFinalizerRun(u) {
			logger.Debugf("Resource is marked for deletion, running finalizer %s", r.Finalizer.Name)
			dc = r.finalizerCmdFunc(ident, inputDir.Path)
		} else {
			dc = r.cmdFunc(ident, inputDir.Path)
		}

		err := dc.Run()
		if err != nil {
			logger.Errorf("error from ansible-runner: %s", err.Error())
		} else {
			logger.Info("ansible-runner exited successfully")
		}

		receiver.Close()
		err = <-errChan
		// http.Server returns this in the case of being closed cleanly
		if err != nil && err != http.ErrServerClosed {
			logger.Errorf("error from event api: %s", err.Error())
		}
	}()
	return receiver.Events, nil
}

// GetReconcilePeriod - new reconcile period.
func (r *runner) GetReconcilePeriod() (time.Duration, bool) {
	if r.reconcilePeriod == time.Duration(0) {
		return r.reconcilePeriod, false
	}
	return r.reconcilePeriod, true
}

func (r *runner) GetFinalizer() (string, bool) {
	if r.Finalizer != nil {
		return r.Finalizer.Name, true
	}
	return "", false
}

func (r *runner) isFinalizerRun(u *unstructured.Unstructured) bool {
	finalizersSet := r.Finalizer != nil && u.GetFinalizers() != nil
	// The resource is deleted and our finalizer is present, we need to run the finalizer
	if finalizersSet && u.GetDeletionTimestamp() != nil {
		for _, f := range u.GetFinalizers() {
			if f == r.Finalizer.Name {
				return true
			}
		}
	}
	return false
}

func (r *runner) addFinalizer(finalizer *Finalizer) error {
	r.Finalizer = finalizer
	switch {
	case finalizer == nil:
		return nil
	case finalizer.Playbook != "":
		if !filepath.IsAbs(finalizer.Playbook) {
			return fmt.Errorf("finalizer playbook path must be absolute for %v", r.GVK)
		}
		r.finalizerCmdFunc = func(ident, inputDirPath string) *exec.Cmd {
			return exec.Command("ansible-runner", "-vv", "-p", finalizer.Playbook, "-i", ident, "run", inputDirPath)
		}
	case finalizer.Role != "":
		if !filepath.IsAbs(finalizer.Role) {
			return fmt.Errorf("finalizer role path must be absolute for %v", r.GVK)
		}
		r.finalizerCmdFunc = func(ident, inputDirPath string) *exec.Cmd {
			path := strings.TrimRight(finalizer.Role, "/")
			rolePath, roleName := filepath.Split(path)
			return exec.Command("ansible-runner", "-vv", "--role", roleName, "--roles-path", rolePath, "--hosts", "localhost", "-i", ident, "run", inputDirPath)
		}
	case len(finalizer.Vars) != 0:
		r.finalizerCmdFunc = r.cmdFunc
	}
	return nil
}

// makeParameters - creates the extravars parameters for ansible
// The resulting structure in json is:
// { "meta": {
//      "name": <object_name>,
//      "namespace": <object_namespace>,
//   },
//   <cr_spec_fields_as_snake_case>,
//   ...
//   _<group_as_snake>_<kind>: {
//       <cr_object as is
//   }
// }
func (r *runner) makeParameters(u *unstructured.Unstructured) map[string]interface{} {
	s := u.Object["spec"]
	spec, ok := s.(map[string]interface{})
	if !ok {
		logrus.Warnf("spec was not found for CR:%v - %v in %v", u.GroupVersionKind(), u.GetNamespace(), u.GetName())
		spec = map[string]interface{}{}
	}
	parameters := paramconv.MapToSnake(spec)
	parameters["meta"] = map[string]string{"namespace": u.GetNamespace(), "name": u.GetName()}
	objectKey := fmt.Sprintf("_%v_%v", strings.Replace(r.GVK.Group, ".", "_", -1), strings.ToLower(r.GVK.Kind))
	parameters[objectKey] = u.Object
	if r.isFinalizerRun(u) {
		for k, v := range r.Finalizer.Vars {
			parameters[k] = v
		}
	}
	return parameters
}
