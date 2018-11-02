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

package inputdir

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var log = logf.Log.WithName("inputdir")

// InputDir represents an input directory for ansible-runner.
type InputDir struct {
	Path         string
	PlaybookPath string
	Parameters   map[string]interface{}
	EnvVars      map[string]string
	Settings     map[string]string
}

// makeDirs creates the required directory structure.
func (i *InputDir) makeDirs() error {
	for _, path := range []string{"env", "project", "inventory"} {
		fullPath := filepath.Join(i.Path, path)
		err := os.MkdirAll(fullPath, os.ModePerm)
		if err != nil {
			log.Error(err, "unable to create directory", "Path", fullPath)
			return err
		}
	}
	return nil
}

// addFile adds a file to the given relative path within the input directory.
func (i *InputDir) addFile(path string, content []byte) error {
	fullPath := filepath.Join(i.Path, path)
	err := ioutil.WriteFile(fullPath, content, 0644)
	if err != nil {
		log.Error(err, "unable to write file", "Path", fullPath)
	}
	return err
}

// Stdout reads the stdout from the ansible artifact that corresponds to the
// given ident and returns it as a string.
func (i *InputDir) Stdout(ident string) (string, error) {
	errorPath := filepath.Join(i.Path, "artifacts", ident, "stdout")
	errorText, err := ioutil.ReadFile(errorPath)
	return string(errorText), err
}

// Write commits the object's state to the filesystem at i.Path.
func (i *InputDir) Write() error {
	paramBytes, err := json.Marshal(i.Parameters)
	if err != nil {
		return err
	}
	envVarBytes, err := json.Marshal(i.EnvVars)
	if err != nil {
		return err
	}
	settingsBytes, err := json.Marshal(i.Settings)
	if err != nil {
		return err
	}

	err = i.makeDirs()
	if err != nil {
		return err
	}

	err = i.addFile("env/envvars", envVarBytes)
	if err != nil {
		return err
	}
	err = i.addFile("env/extravars", paramBytes)
	if err != nil {
		return err
	}
	err = i.addFile("env/settings", settingsBytes)
	if err != nil {
		return err
	}

	// If ansible-runner is running in a python virtual environment, propagate
	// that to ansible.
	venv := os.Getenv("VIRTUAL_ENV")
	hosts := "localhost ansible_connection=local"
	if venv != "" {
		hosts = fmt.Sprintf("%s ansible_python_interpreter=%s", hosts, filepath.Join(venv, "bin/python"))
	}
	err = i.addFile("inventory/hosts", []byte(hosts))
	if err != nil {
		return err
	}

	if i.PlaybookPath != "" {
		f, err := os.Open(i.PlaybookPath)
		if err != nil {
			log.Error(err, "failed to open playbook file", "Path", i.PlaybookPath)
			return err
		}
		defer f.Close()

		playbookBytes, err := ioutil.ReadAll(f)
		if err != nil {
			return err
		}

		err = i.addFile("project/playbook.yaml", playbookBytes)
		if err != nil {
			return err
		}
	}
	return nil
}
