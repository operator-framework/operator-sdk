// Copyright 2019 The Operator-SDK Authors
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

package olm

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/pflag"
)

// PackageManifestsCmd configures deployment and teardown of an operator
// managed in a package manifests format via OLM.
type PackageManifestsCmd struct {
	OperatorCmd

	// ManifestsDir is a directory containing 1..N package directories and
	// a package manifest.
	// Version can be set to the version of the desired operator package
	// and Run() will deploy that operator version.
	ManifestsDir string
	// Version is the version of the operator to deploy. It must be
	// a semantic version, ex. 0.0.1.
	Version string
}

func (c *PackageManifestsCmd) AddToFlagSet(fs *pflag.FlagSet) {
	c.OperatorCmd.AddToFlagSet(fs)

	fs.StringVar(&c.Version, "version", "", "Packaged version of the operator to deploy")
}

func (c *PackageManifestsCmd) validate() error {
	if c.ManifestsDir == "" {
		return errors.New("manifests dir must be set")
	}
	manDirInfo, err := os.Stat(c.ManifestsDir)
	if err != nil {
		return err
	}
	if !manDirInfo.IsDir() {
		return fmt.Errorf("%s must be a directory", c.ManifestsDir)
	}

	if c.Version == "" {
		return errors.New("operator version must be set")
	}

	return c.OperatorCmd.validate()
}

func (c *PackageManifestsCmd) initialize() {
	c.OperatorCmd.initialize()
}

func (c *PackageManifestsCmd) Run() error {
	c.initialize()
	if err := c.validate(); err != nil {
		return fmt.Errorf("validation error: %w", err)
	}
	m, err := c.newManager()
	if err != nil {
		return fmt.Errorf("error initializing operator manager: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), c.Timeout)
	defer cancel()
	return m.run(ctx)
}
