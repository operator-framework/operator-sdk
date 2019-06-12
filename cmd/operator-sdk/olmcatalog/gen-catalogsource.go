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

package olmcatalog

import (
	"fmt"

	catalogcmd "github.com/operator-framework/operator-sdk/pkg/scaffold/olm-catalog"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const noFileIndicator = "##nofile##"

func newGenCatalogSourceCmd() *cobra.Command {
	c := &catalogcmd.GenCatalogSourceCmd{}

	cmd := &cobra.Command{
		Use:   "gen-catalogsource",
		Short: "Generates a CatalogSource ConfigMap yaml file from an operator's registry bundle",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 0 {
				return fmt.Errorf("command %s doesn't accept any arguments", cmd.CommandPath())
			}
			// If the user has specified --write/-w without any args, inform c that
			// the CatalogSource should be written to the default path.
			c.Write = cmd.Flags().Changed("write") && c.WriteTo == noFileIndicator
			return c.Run()
		},
	}

	cmd.Flags().StringVar(&c.BundleDir, "bundle-dir", "", "Directory of bundled operator CSV, CRD's, and optionally a package manifest")
	if err := cmd.MarkFlagRequired("bundle-dir"); err != nil {
		log.Fatalf("Failed to mark `bundle-dir` flag for `gen-catalogsource` subcommand as required")
	}
	cmd.Flags().StringVar(&c.PackageManifestPath, "package-manifest", "", "Path of the package manifest. Optional if the bundle dir contains one")
	cmd.Flags().StringVarP(&c.OutputFormat, "output-format", "o", string(catalogcmd.OutputFormatYAML),
		fmt.Sprintf("Format of ConfigMap being printed or written. Must be one of: %s, %s", catalogcmd.OutputFormatJSON, catalogcmd.OutputFormatYAML))
	cmd.Flags().StringVarP(&c.WriteTo, "write", "w", "", "Write output to a specified file, or to deploy/olm-catalog/{project_name} if no file path is provided")
	// Set the default value of "write" so we can treat it as a boolean.
	cmd.Flags().Lookup("write").NoOptDefVal = noFileIndicator
	return cmd
}
