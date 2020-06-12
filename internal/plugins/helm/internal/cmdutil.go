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

// todo: check if we can use it directly from kb or if we need to have our own
// it is the same implementation of kubebuilder/internal/cmdutil
package cmdutil

import (
	"sigs.k8s.io/kubebuilder/pkg/scaffold"
)

// RunOptions represent the types used to implement the different commands
type RunOptions interface {
	// - Step 1: verify that the command can be run (e.g., go version, project version, arguments, ...)
	Validate() error
	// - Step 2: create the Scaffolder instance
	// todo: check if we will use the scaffold.Scaffolder or have our own in the future
	GetScaffolder() (scaffold.Scaffolder, error)
	// - Step 3: call the Scaffold method of the Scaffolder instance. Doesn't need any method
	// - Step 4: finish the command execution
	PostScaffold() error
}

// Run executes a command
func Run(options RunOptions) error {
	// Step 1: validate
	if err := options.Validate(); err != nil {
		return err
	}

	// Step 2: get scaffolder
	scaffolder, err := options.GetScaffolder()
	if err != nil {
		return err
	}
	// Step 3: scaffold
	// for the helm plugin the scaffolder for now has not been returned
	// it might change after we replace model and business layers for the new layout
	if scaffolder != nil {
		if err := scaffolder.Scaffold(); err != nil {
			return err
		}
	}

	// Step 4: finish
	if err := options.PostScaffold(); err != nil {
		return err
	}

	return nil
}
