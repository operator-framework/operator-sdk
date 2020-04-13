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
	"fmt"
	"io/ioutil"
	"os"

	log "github.com/sirupsen/logrus"
	"sigs.k8s.io/kubebuilder/pkg/model/config"
)

const ConfigFile = "PROJECT"

// IsConfigExist returns true if the project is configured as a kubebuilder
// project.
func IsConfigExist() bool {
	_, err := os.Stat(ConfigFile)
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}
		log.Fatalf("Failed to read PROJECT file to detect kubebuilder project: %v", err)
	}
	return true
}

// DieIfCmdNotAllowed logs a message and exits if a callee cannot run on a
// kubebuilder-style project.
// FEAT(estroz): pass an equivalent command as string to this function to
// customize log message.
func DieIfCmdNotAllowed(hasEquivalent bool) {
	if IsConfigExist() {
		if !hasEquivalent {
			log.Fatal("This command does not work with kubebuilder-style projects.")
		}
		log.Fatal("This command does not work with kubebuilder-style projects. " +
			"Please read the kubebuilder book for an equivalent command: https://book.kubebuilder.io/")
	}
}

func ReadConfig() (*config.Config, error) {
	content, err := ioutil.ReadFile(ConfigFile)
	if err != nil {
		return nil, fmt.Errorf("error reading config: %v", err)
	}
	c := &config.Config{}
	if err = config.Unmarshal(content, c); err != nil {
		return nil, err
	}
	return c, nil
}
