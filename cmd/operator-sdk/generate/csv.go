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

package generate

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/operator-framework/operator-sdk/internal/generate/gen"
	gencatalog "github.com/operator-framework/operator-sdk/internal/generate/olm-catalog"
	"github.com/operator-framework/operator-sdk/internal/scaffold"
	"github.com/operator-framework/operator-sdk/internal/scaffold/input"
	catalog "github.com/operator-framework/operator-sdk/internal/scaffold/olm-catalog"
	"github.com/operator-framework/operator-sdk/internal/util/fileutil"
	"github.com/operator-framework/operator-sdk/internal/util/k8sutil"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"

	"github.com/coreos/go-semver/semver"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type csvCmd struct {
	csvVersion     string
	csvChannel     string
	fromVersion    string
	csvConfigPath  string
	operatorName   string
	updateCRDs     bool
	defaultChannel bool
}

func newGenerateCSVCmd() *cobra.Command {
	c := &csvCmd{}
	cmd := &cobra.Command{
		Use:   "csv",
		Short: "Generates a ClusterServiceVersion YAML file for the operator",
		Long: `The 'generate csv' command generates a ClusterServiceVersion (CSV) YAML manifest
for the operator. This file is used to publish the operator to the OLM Catalog.

A CSV semantic version is supplied via the --csv-version flag. If your operator
has already generated a CSV manifest you want to use as a base, supply its
version to --from-version. Otherwise the SDK will scaffold a new CSV manifest.

Configure CSV generation by writing a config file 'deploy/olm-catalog/csv-config.yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// The CSV generator assumes that the deploy and pkg directories are
			// present at runtime, so this command must be run in a project's root.
			projutil.MustInProjectRoot()

			if len(args) != 0 {
				return fmt.Errorf("command %s doesn't accept any arguments", cmd.CommandPath())
			}
			if err := c.validate(); err != nil {
				return fmt.Errorf("error validating command flags: %v", err)
			}
			if err := c.run(); err != nil {
				log.Fatal(err)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&c.csvVersion, "csv-version", "", "Semantic version of the CSV")
	if err := cmd.MarkFlagRequired("csv-version"); err != nil {
		log.Fatalf("Failed to mark `csv-version` flag for `generate csv` subcommand as required: %v", err)
	}
	cmd.Flags().StringVar(&c.fromVersion, "from-version", "",
		"Semantic version of an existing CSV to use as a base")
	cmd.Flags().StringVar(&c.csvConfigPath, "csv-config", "",
		"Path to CSV config file. Defaults to deploy/olm-catalog/csv-config.yaml")
	cmd.Flags().BoolVar(&c.updateCRDs, "update-crds", false,
		"Update CRD manifests in deploy/{operator-name}/{csv-version} the using latest API's")
	cmd.Flags().StringVar(&c.operatorName, "operator-name", "",
		"Operator name to use while generating CSV")
	cmd.Flags().StringVar(&c.csvChannel, "csv-channel", "",
		"Channel the CSV should be registered under in the package manifest")
	cmd.Flags().BoolVar(&c.defaultChannel, "default-channel", false,
		"Use the channel passed to --csv-channel as the package manifests' default channel. "+
			"Only valid when --csv-channel is set")

	return cmd
}

func (c csvCmd) run() error {

	absProjectPath := projutil.MustGetwd()
	cfg := &input.Config{
		AbsProjectPath: absProjectPath,
		ProjectName:    filepath.Base(absProjectPath),
	}
	if projutil.IsOperatorGo() {
		cfg.Repo = projutil.GetGoPkg()
	}

	log.Infof("Generating CSV manifest version %s", c.csvVersion)

	csvCfg, err := catalog.GetCSVConfig(c.csvConfigPath)
	if err != nil {
		return err
	}
	if c.operatorName == "" {
		// Use config operator name if not set by CLI, i.e. prefer CLI value over
		// config value.
		if c.operatorName = csvCfg.OperatorName; c.operatorName == "" {
			// Default to using project name if both are empty.
			c.operatorName = filepath.Base(absProjectPath)
		}
	}

	s := &scaffold.Scaffold{}
	csv := &catalog.CSV{
		CSVVersion:     c.csvVersion,
		FromVersion:    c.fromVersion,
		ConfigFilePath: c.csvConfigPath,
		OperatorName:   c.operatorName,
	}
	err = s.Execute(cfg, csv)
	if err != nil {
		return fmt.Errorf("catalog scaffold failed: %v", err)
	}

	gcfg := gen.Config{
		OperatorName: c.operatorName,
		OutputDir:    filepath.Join(gencatalog.OLMCatalogDir, c.operatorName),
	}
	pkg := gencatalog.NewPackageManifest(gcfg, c.csvVersion, c.csvChannel, c.defaultChannel)
	if err := pkg.Generate(); err != nil {
		return fmt.Errorf("error generating package manifest: %v", err)
	}

	// Write CRD's to the new or updated CSV package dir.
	if c.updateCRDs {
		input, err := csv.GetInput()
		if err != nil {
			return err
		}
		err = writeCRDsToDir(csvCfg.CRDCRPaths, filepath.Dir(input.Path))
		if err != nil {
			return err
		}
	}

	return nil
}

func (c csvCmd) validate() error {
	if err := validateVersion(c.csvVersion); err != nil {
		return err
	}
	if c.fromVersion != "" {
		if err := validateVersion(c.fromVersion); err != nil {
			return err
		}
	}
	if c.fromVersion != "" && c.csvVersion == c.fromVersion {
		return fmt.Errorf(
			"from-version (%s) cannot equal csv-version; set only csv-version instead", c.fromVersion)
	}

	if c.defaultChannel && c.csvChannel == "" {
		return fmt.Errorf("default-channel can only be used if csv-channel is set")
	}

	return nil
}

func validateVersion(version string) error {
	v, err := semver.NewVersion(version)
	if err != nil {
		return fmt.Errorf("%s is not a valid semantic version: %v", version, err)
	}
	// Ensures numerical values composing csvVersion don't contain leading 0's,
	// ex. 01.01.01
	if v.String() != version {
		return fmt.Errorf("provided CSV version %s contains bad values (parses to %s)", version, v)
	}
	return nil
}

func writeCRDsToDir(crdPaths []string, toDir string) error {
	for _, p := range crdPaths {
		b, err := ioutil.ReadFile(p)
		if err != nil {
			return err
		}
		typeMeta, err := k8sutil.GetTypeMetaFromBytes(b)
		if err != nil {
			return fmt.Errorf("error in %s : %v", p, err)
		}
		if typeMeta.Kind != "CustomResourceDefinition" {
			continue
		}

		path := filepath.Join(toDir, filepath.Base(p))
		err = ioutil.WriteFile(path, b, fileutil.DefaultFileMode)
		if err != nil {
			return err
		}
	}
	return nil
}
