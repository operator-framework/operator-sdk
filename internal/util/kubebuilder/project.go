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

package kbutil

import (
	"io/ioutil"
	"os"

	log "github.com/sirupsen/logrus"
	"sigs.k8s.io/kubebuilder/pkg/model/config"
)

const configFile = "PROJECT"

// HasProjectFile returns true if the project is configured as a kubebuilder
// project.
func HasProjectFile() bool {
	_, err := os.Stat(configFile)
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}
		log.Fatalf("Failed to read PROJECT file to detect kubebuilder project: %v", err)
	}
	return true
}

// ReadConfig returns a configuration if a file containing one exists at the
// default path (project root).
func ReadConfig() (*config.Config, error) {
	b, err := ioutil.ReadFile(configFile)
	if err != nil {
		return nil, err
	}
	c := &config.Config{}
	if err = c.Unmarshal(b); err != nil {
		return nil, err
	}
	return c, nil
}
