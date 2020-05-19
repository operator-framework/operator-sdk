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

package cleanup

import (
	"fmt"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	olmcatalog "github.com/operator-framework/operator-sdk/internal/generate/olm-catalog"
	olmoperator "github.com/operator-framework/operator-sdk/internal/olm/operator"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"
)

type packagemanifestsCmdLegacy struct {
	olmoperator.PackageManifestsCmd
}

func newPackageManifestsCmdLegacy() *cobra.Command {
	c := &packagemanifestsCmdLegacy{}

	cmd := &cobra.Command{
		Use:   "packagemanifests",
		Short: "Clean up after an Operator organized in the package manifests format running with OLM",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				if len(args) > 1 {
					return fmt.Errorf("exactly one argument is required")
				}
				c.ManifestsDir = args[0]
			} else {
				operatorName := filepath.Base(projutil.MustGetwd())
				c.ManifestsDir = filepath.Join(olmcatalog.OLMCatalogDir, operatorName)
			}

			log.Infof("Cleaning up operator in directory %s", c.ManifestsDir)

			if err := c.Cleanup(); err != nil {
				log.Fatalf("Failed to clean up operator: %v", err)
			}
			return nil
		},
	}

	c.PackageManifestsCmd.AddToFlagSet(cmd.Flags())

	return cmd
}
