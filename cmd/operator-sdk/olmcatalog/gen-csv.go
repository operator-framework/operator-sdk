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

package olmcatalog

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/operator-framework/operator-sdk/internal/pkg/scaffold"
	catalog "github.com/operator-framework/operator-sdk/internal/pkg/scaffold/olm-catalog"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"
	"github.com/operator-framework/operator-sdk/pkg/config"
	catalogcmd "github.com/operator-framework/operator-sdk/pkg/scaffold/olm-catalog"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func newGenCSVCmd() *cobra.Command {
	c := &catalogcmd.GenCSVCmd{}

	genCSVCmd := &cobra.Command{
		Use:   "gen-csv",
		Short: "Generates a Cluster Service Version yaml file for the operator",
		Long: `The gen-csv command generates a Cluster Service Version (CSV) YAML manifest
for the operator. This file is used to publish the operator to the OLM Catalog.

A CSV semantic version is supplied via the --csv-version flag. If your operator
has already generated a CSV manifest you want to use as a base, supply its
version to --from-version. Otherwise the SDK will scaffold a new CSV manifest.

Configure CSV generation in your operator-sdk config file.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 0 {
				return fmt.Errorf("command %s doesn't accept any arguments", cmd.CommandPath())
			}
			projutil.MustInProjectRoot()
			return c.Run()
		},
	}

	genCSVCmd.Flags().StringVar(&c.CSVVersion, "csv-version", "", "Semantic version of the CSV")
	if err := genCSVCmd.MarkFlagRequired("csv-version"); err != nil {
		log.Fatalf("Failed to mark `csv-version` flag for `gen-csv` subcommand as required")
	}
	genCSVCmd.Flags().StringVar(&c.FromVersion, "from-version", "", "Semantic version of an existing CSV to use as a base")
	genCSVCmd.Flags().BoolVar(&c.UpdateCRDs, "update-crds", false, "Update CRD manifests in deploy/{operator-name}/{csv-version} the using latest API's")

	fset := pflag.NewFlagSet("", pflag.ExitOnError)
	fset.String(stripPrefix(catalog.OperatorPathOpt), filepath.Join(scaffold.DeployDir, scaffold.OperatorYamlFile), "Path to operator manifest")
	fset.String(stripPrefix(catalog.RolePathOpt), filepath.Join(scaffold.DeployDir, scaffold.RoleYamlFile), "Path to RBAC role manifest")
	fset.StringSlice(stripPrefix(catalog.CRDCRPathsOpt), []string{scaffold.CRDsDir}, "Path slice of CRD and CR manifests")
	config.BindFlagsWithPrefix(fset, catalog.OLMCatalogConfigOpt)
	genCSVCmd.Flags().AddFlagSet(fset)

	return genCSVCmd
}

func stripPrefix(k string) string {
	return strings.TrimPrefix(k, catalog.OLMCatalogConfigOpt+".")
}
