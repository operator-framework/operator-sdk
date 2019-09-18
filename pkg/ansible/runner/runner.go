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
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/operator-framework/operator-sdk/pkg/ansible/metrics"
	"github.com/operator-framework/operator-sdk/pkg/ansible/paramconv"
	"github.com/operator-framework/operator-sdk/pkg/ansible/runner/eventapi"
	"github.com/operator-framework/operator-sdk/pkg/ansible/runner/internal/inputdir"
	"github.com/operator-framework/operator-sdk/pkg/ansible/watches"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var log = logf.Log.WithName("runner")

const (
	// MaxRunnerArtifactsAnnotation - annotation used by a user to specify the max artifacts to keep
	// in the runner directory. This will override the value provided by the watches file for a
	// particular CR. Setting this to zero will cause all artifact directories to be kept.
	// Example usage "ansible.operator-sdk/max-runner-artifacts: 100"
	MaxRunnerArtifactsAnnotation = "ansible.operator-sdk/max-runner-artifacts"
)

// Runner - a runnable that should take the parameters and name and namespace
// and run the correct code.
type Runner interface {
	Run(string, *unstructured.Unstructured, string) (RunResult, error)
	GetFinalizer() (string, bool)
}

// New - creates a Runner from a Watch struct
func New(watch watches.Watch) (Runner, error) {
	// handle role or playbook
	var path string
	var cmdFunc func(ident, inputDirPath string, maxArtifacts int) *exec.Cmd

	switch {
	case watch.Playbook != "":
		path = watch.Playbook
		cmdFunc = func(ident, inputDirPath string, maxArtifacts int) *exec.Cmd {
			return exec.Command("ansible-runner", "-vv", "--rotate-artifacts", fmt.Sprintf("%v", maxArtifacts), "-p", path, "-i", ident, "run", inputDirPath)
		}
	case watch.Role != "":
		path = watch.Role
		cmdFunc = func(ident, inputDirPath string, maxArtifacts int) *exec.Cmd {
			rolePath, roleName := filepath.Split(path)
			return exec.Command("ansible-runner", "-vv", "--rotate-artifacts", fmt.Sprintf("%v", maxArtifacts), "--role", roleName, "--roles-path", rolePath, "--hosts", "localhost", "-i", ident, "run", inputDirPath)
		}
	default:
		return nil, fmt.Errorf("must specify Role or Path")
	}

	// handle finalizer
	var finalizer *watches.Finalizer
	var finalizerCmdFunc func(ident, inputDirPath string, maxArtifacts int) *exec.Cmd
	switch {
	case watch.Finalizer == nil:
		finalizer = nil
		finalizerCmdFunc = nil
	case watch.Finalizer.Playbook != "":
		finalizer = watch.Finalizer
		finalizerCmdFunc = func(ident, inputDirPath string, maxArtifacts int) *exec.Cmd {
			return exec.Command("ansible-runner", "-vv", "--rotate-artifacts", fmt.Sprintf("%v", maxArtifacts), "-p", finalizer.Playbook, "-i", ident, "run", inputDirPath)
		}
	case watch.Finalizer.Role != "":
		finalizer = watch.Finalizer
		finalizerCmdFunc = func(ident, inputDirPath string, maxArtifacts int) *exec.Cmd {
			path := strings.TrimRight(finalizer.Role, "/")
			rolePath, roleName := filepath.Split(path)
			return exec.Command("ansible-runner", "-vv", "--rotate-artifacts", fmt.Sprintf("%v", maxArtifacts), "--role", roleName, "--roles-path", rolePath, "--hosts", "localhost", "-i", ident, "run", inputDirPath)
		}
	case len(watch.Finalizer.Vars) != 0:
		finalizer = watch.Finalizer
		finalizerCmdFunc = cmdFunc
	}

	return &runner{
		Path:               path,
		cmdFunc:            cmdFunc,
		Finalizer:          finalizer,
		finalizerCmdFunc:   finalizerCmdFunc,
		GVK:                watch.GroupVersionKind,
		maxRunnerArtifacts: watch.MaxRunnerArtifacts,
	}, nil
}

// runner - implements the Runner interface for a GVK that's being watched.
type runner struct {
	Path               string                  // path on disk to a playbook or role depending on what cmdFunc expects
	GVK                schema.GroupVersionKind // GVK being watched that corresponds to the Path
	Finalizer          *watches.Finalizer
	cmdFunc            func(ident, inputDirPath string, maxArtifacts int) *exec.Cmd // returns a Cmd that runs ansible-runner
	finalizerCmdFunc   func(ident, inputDirPath string, maxArtifacts int) *exec.Cmd
	maxRunnerArtifacts int
}

func (r *runner) Run(ident string, u *unstructured.Unstructured, kubeconfig string) (RunResult, error) {

	timer := metrics.ReconcileTimer(r.GVK.String())
	defer timer.ObserveDuration()

	if u.GetDeletionTimestamp() != nil && !r.isFinalizerRun(u) {
		return nil, errors.New("resource has been deleted, but no finalizer was matched, skipping reconciliation")
	}
	logger := log.WithValues(
		"job", ident,
		"name", u.GetName(),
		"namespace", u.GetNamespace(),
	)

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
			"KUBECONFIG":          kubeconfig,
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
	maxArtifacts := r.maxRunnerArtifacts
	if ma, ok := u.GetAnnotations()[MaxRunnerArtifactsAnnotation]; ok {
		i, err := strconv.Atoi(ma)
		if err != nil {
			log.Info("Invalid max runner artifact annotation", "err", err, "value", ma)
		}
		maxArtifacts = i
	}

	go func() {
		var dc *exec.Cmd
		if r.isFinalizerRun(u) {
			logger.V(1).Info("Resource is marked for deletion, running finalizer", "Finalizer", r.Finalizer.Name)
			dc = r.finalizerCmdFunc(ident, inputDir.Path, maxArtifacts)
		} else {
			dc = r.cmdFunc(ident, inputDir.Path, maxArtifacts)
		}
		// Append current environment since setting dc.Env to anything other than nil overwrites current env
		dc.Env = append(dc.Env, os.Environ()...)
		dc.Env = append(dc.Env, fmt.Sprintf("K8S_AUTH_KUBECONFIG=%s", kubeconfig), fmt.Sprintf("KUBECONFIG=%s", kubeconfig))

		output, err := dc.CombinedOutput()
		if err != nil {
			logger.Error(err, string(output))
		} else {
			logger.Info("Ansible-runner exited successfully")
		}

		receiver.Close()
		err = <-errChan
		// http.Server returns this in the case of being closed cleanly
		if err != nil && err != http.ErrServerClosed {
			logger.Error(err, "Error from event API")
		}

		// link the current run to the `latest` directory under artifacts
		currentRun := filepath.Join(inputDir.Path, "artifacts", ident)
		latestArtifacts := filepath.Join(inputDir.Path, "artifacts", "latest")
		if _, err = os.Lstat(latestArtifacts); err == nil {
			if err = os.Remove(latestArtifacts); err != nil {
				logger.Error(err, "Error removing the latest artifacts symlink")
			}
		}
		if err = os.Symlink(currentRun, latestArtifacts); err != nil {
			logger.Error(err, "Error symlinking latest artifacts")
		}

	}()

	return &runResult{
		events:   receiver.Events,
		inputDir: &inputDir,
		ident:    ident,
	}, nil
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

// makeParameters - creates the extravars parameters for ansible
// The resulting structure in json is:
// { "meta": {
//      "name": <object_name>,
//      "namespace": <object_namespace>,
//   },
//   <cr_spec_fields_as_snake_case>,
//   ...
//   _<group_as_snake>_<kind>: {
//       <cr_object> as is
//   }
//   _<group_as_snake>_<kind>_spec: {
//       <cr_object.spec> as is
//   }
// }
func (r *runner) makeParameters(u *unstructured.Unstructured) map[string]interface{} {
	s := u.Object["spec"]
	spec, ok := s.(map[string]interface{})
	if !ok {
		log.Info("Spec was not found for CR", "GroupVersionKind", u.GroupVersionKind(), "Namespace", u.GetNamespace(), "Name", u.GetName())
		spec = map[string]interface{}{}
	}

	parameters := paramconv.MapToSnake(spec)
	parameters["meta"] = map[string]string{"namespace": u.GetNamespace(), "name": u.GetName()}

	objKey := fmt.Sprintf("_%v_%v", strings.Replace(r.GVK.Group, ".", "_", -1), strings.ToLower(r.GVK.Kind))
	parameters[objKey] = u.Object

	specKey := fmt.Sprintf("%s_spec", objKey)
	parameters[specKey] = spec

	if r.isFinalizerRun(u) {
		for k, v := range r.Finalizer.Vars {
			parameters[k] = v
		}
	}
	return parameters
}

func (r *runner) GetFinalizer() (string, bool) {
	if r.Finalizer != nil {
		return r.Finalizer.Name, true
	}
	return "", false
}

// RunResult - result of a ansible run
type RunResult interface {
	// Stdout returns the stdout from ansible-runner if it is available, else an error.
	Stdout() (string, error)
	// Events returns the events from ansible-runner if it is available, else an error.
	Events() <-chan eventapi.JobEvent
}

// RunResult facilitates access to information about a run of ansible.
type runResult struct {
	// Events is a channel of events from ansible that contain state related
	// to a run of ansible.
	events <-chan eventapi.JobEvent

	ident    string
	inputDir *inputdir.InputDir
}

// Stdout returns the stdout from ansible-runner if it is available, else an error.
func (r *runResult) Stdout() (string, error) {
	return r.inputDir.Stdout(r.ident)
}

// Events returns the events from ansible-runner if it is available, else an error.
func (r *runResult) Events() <-chan eventapi.JobEvent {
	return r.events
}
