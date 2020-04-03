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
	"os"

	log "github.com/sirupsen/logrus"
)

// IsConfigExist returns true if the project is configured as a kubebuilder
// project.
func IsConfigExist() bool {
	_, err := os.Stat("PROJECT")
	return err == nil || os.IsExist(err)
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
