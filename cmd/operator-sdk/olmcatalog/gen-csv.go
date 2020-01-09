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
	"io/ioutil"
	"os"
	"path/filepath"

	gen "github.com/operator-framework/operator-sdk/internal/generate/gen"
	gencatalog "github.com/operator-framework/operator-sdk/internal/generate/olm-catalog"
	"github.com/operator-framework/operator-sdk/internal/scaffold"
	"github.com/operator-framework/operator-sdk/internal/util/fileutil"
	"github.com/operator-framework/operator-sdk/internal/util/k8sutil"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"

	"github.com/coreos/go-semver/semver"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	csvVersion     string
	csvChannel     string
	fromVersion    string
	outputDir      string
	includePaths   []string
	operatorName   string
	updateCRDs     bool
	defaultChannel bool
)

func newGenCSVCmd() *cobra.Command {
	genCSVCmd := &cobra.Command{
		Use:   "gen-csv",
		Short: "Generates a Cluster Service Version yaml file for the operator",
		Long: `The gen-csv command generates a Cluster Service Version (CSV) YAML manifest
for the operator. This file is used to publish the operator to the OLM Catalog.

A CSV semantic version is supplied via the --csv-version flag. If your operator
has already generated a CSV manifest you want to use as a base, supply its
version to --from-version. Otherwise the SDK will scaffold a new CSV manifest.`,
		RunE: genCSVFunc,
	}

	genCSVCmd.Flags().StringVar(&csvVersion, "csv-version", "", "Semantic version of the CSV")
	if err := genCSVCmd.MarkFlagRequired("csv-version"); err != nil {
		log.Fatalf("Failed to mark `csv-version` flag for `olm-catalog gen-csv` subcommand as required: %v", err)
	}
	genCSVCmd.Flags().StringVar(&fromVersion, "from-version", "", "Semantic version of an existing CSV to use as a base")
	genCSVCmd.Flags().StringSliceVar(&includePaths, "include", []string{scaffold.DeployDir}, "Paths to include in CSV generation, ex. \"deploy/prod,deploy/test\". If this flag is set and you want to enable default behavior, you must include \"deploy/\" in the argument list")
	genCSVCmd.Flags().StringVar(&outputDir, "output-dir", scaffold.DeployDir, "Base directory to output generated CSV. The resulting CSV bundle directory will be \"<output-dir>/olm-catalog/<operator-name>/<csv-version>\"")
	genCSVCmd.Flags().BoolVar(&updateCRDs, "update-crds", false, "Update CRD manifests in deploy/{operator-name}/{csv-version} the using latest API's")
	genCSVCmd.Flags().StringVar(&operatorName, "operator-name", "", "Operator name to use while generating CSV")
	genCSVCmd.Flags().StringVar(&csvChannel, "csv-channel", "", "Channel the CSV should be registered under in the package manifest")
	genCSVCmd.Flags().BoolVar(&defaultChannel, "default-channel", false, "Use the channel passed to --csv-channel as the package manifests' default channel. Only valid when --csv-channel is set")

	return genCSVCmd
}

func genCSVFunc(cmd *cobra.Command, args []string) error {
	// The CSV generator assumes that the deploy and pkg directories are present
	// at runtime, so this command must be run in a project's root.
	projutil.MustInProjectRoot()

	if len(args) != 0 {
		return fmt.Errorf("command %s doesn't accept any arguments", cmd.CommandPath())
	}

	if err := verifyGenCSVFlags(); err != nil {
		return err
	}

	log.Infof("Generating CSV manifest version %s", csvVersion)

	if operatorName == "" {
		operatorName = filepath.Base(projutil.MustGetwd())
	}
	cfg := gen.Config{
		OperatorName: operatorName,
		OutputDir:    outputDir,
		Filters:      gen.MakeFilters(includePaths...),
	}

	csv := gencatalog.NewCSV(cfg, csvVersion, fromVersion)
	if err := csv.Generate(); err != nil {
		return fmt.Errorf("error generating CSV: %v", err)
	}
	pkg := gencatalog.NewPackageManifest(cfg, csvVersion, csvChannel, defaultChannel)
	if err := pkg.Generate(); err != nil {
		return fmt.Errorf("error generating package manifest: %v", err)
	}

	// Write CRD's to the new or updated CSV package dir.
	if updateCRDs {
		crdManifestSet, err := findCRDs(includePaths...)
		if err != nil {
			return err
		}
		baseDir := outputDir
		if baseDir == "" {
			baseDir = gencatalog.OLMCatalogDir
		}
		bundleDir := filepath.Join(baseDir, operatorName, csvVersion)
		for path, b := range crdManifestSet {
			path = filepath.Join(bundleDir, path)
			if err = ioutil.WriteFile(path, b, fileutil.DefaultFileMode); err != nil {
				return err
			}
		}
	}

	return nil
}

func verifyGenCSVFlags() error {
	if err := verifyCSVVersion(csvVersion); err != nil {
		return err
	}
	if fromVersion != "" {
		if err := verifyCSVVersion(fromVersion); err != nil {
			return err
		}
	}
	if fromVersion != "" && csvVersion == fromVersion {
		return fmt.Errorf("from-version (%s) cannot equal csv-version; set only csv-version instead", fromVersion)
	}

	if defaultChannel && csvChannel == "" {
		return fmt.Errorf("default-channel can only be used if csv-channel is set")
	}

	return nil
}

func verifyCSVVersion(version string) error {
	v, err := semver.NewVersion(version)
	if err != nil {
		return fmt.Errorf("%s is not a valid semantic version: (%v)", version, err)
	}
	// Ensures numerical values composing csvVersion don't contain leading 0's,
	// ex. 01.01.01
	if v.String() != version {
		return fmt.Errorf("provided CSV version %s contains bad values (parses to %s)", version, v)
	}
	return nil
}

// findCRDs searches directories and files in paths for CRD manifest paths,
// returning a map of paths to file contents.
func findCRDs(paths ...string) (map[string][]byte, error) {
	crdPathSet := map[string][]byte{}
	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			return nil, err
		}
		if info.IsDir() {
			subsetPaths, err := k8sutil.GetCRDManifestPaths(path)
			if err != nil {
				return nil, err
			}
			for _, crdPath := range subsetPaths {
				b, err := ioutil.ReadFile(crdPath)
				if err != nil {
					return nil, err
				}
				crdPathSet[filepath.Clean(crdPath)] = b
			}
		} else {
			b, err := ioutil.ReadFile(path)
			if err != nil {
				return nil, err
			}
			typeMeta, err := k8sutil.GetTypeMetaFromBytes(b)
			if err != nil {
				log.Infof("Skipping non-manifest file %s", path)
				continue
			}
			if typeMeta.Kind == "CustomResourceDefinition" {
				crdPathSet[filepath.Clean(path)] = b
			}
		}
	}
	return crdPathSet, nil
}
