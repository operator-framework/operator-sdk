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
	"path/filepath"

	gencatalog "github.com/operator-framework/operator-sdk/internal/generate/olm-catalog"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"

	"github.com/blang/semver"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type csvCmd struct {
	csvVersion       string
	csvChannel       string
	fromVersion      string
	operatorName     string
	outputDir        string
	deployDir        string
	apisDir          string
	crdDir           string
	interactivelevel projutil.InteractiveLevel
	updateCRDs       bool
	defaultChannel   bool
	makeManifests    bool
	interactive      bool
}

//nolint:lll
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

The --make-manifests flag directs the generator to create a bundle manifests directory
intended to hold your latest operator manifests. This flag is true by default.

More information on bundles:
https://github.com/operator-framework/operator-registry/blob/master/docs/design/operator-bundle.md#operator-bundle-overview

Flags that change project default paths:
  --deploy-dir:
    The CSV's install strategy and permissions will be generated from the operator manifests
    (Deployment and Role/ClusterRole) present in this directory.

  --apis-dir:
    The CSV annotation comments will be parsed from the Go types under this path to
    fill out metadata for owned APIs in spec.customresourcedefinitions.owned.

  --crd-dir:
    The CSV's spec.customresourcedefinitions.owned field is generated from the CRD manifests
    in this path. These CRD manifests are also copied over to the bundle directory if
    --update-crds=true (the default). Additionally the CR manifests will be used to populate
    the CSV example CRs.
`,
		Example: `    ##### Generate a CSV in bundle format from default input paths #####
    $ tree pkg/apis/ deploy/
    pkg/apis/
    ├── ...
    └── cache
        ├── group.go
        ├── v1alpha1
        ├── ...
        └── memcached_types.go
    deploy/
    ├── crds
    │   ├── cache.example.com_memcacheds_crd.yaml
    │   └── cache.example.com_v1alpha1_memcached_cr.yaml
    ├── operator.yaml
    ├── role.yaml
    ├── role_binding.yaml
    └── service_account.yaml

    $ operator-sdk generate csv --csv-version=0.0.1
    INFO[0000] Generating CSV manifest version 0.0.1
    ...

    $ tree deploy/
    deploy/
    ...
    └── olm-catalog
        └── memcached-operator
            └── manifests
                ├── cache.example.com_memcacheds_crd.yaml
                └── memcached-operator.clusterserviceversion.yaml
    ...

    ##### Generate a CSV in package manifests format from default input paths #####

		$ operator-sdk generate csv --csv-version=0.0.1 --make-manifests=false --update-crds
    INFO[0000] Generating CSV manifest version 0.0.1
    ...
    $ tree deploy/
    deploy/
    ...
    └── olm-catalog
        └── memcached-operator
            ├── 0.0.1
            │   ├── cache.example.com_memcacheds_crd.yaml
            │   └── memcached-operator.v0.0.1.clusterserviceversion.yaml
            └── memcached-operator.package.yaml
    ...

    ##### Generate CSV from custom input paths #####
    $ operator-sdk generate csv --csv-version=0.0.1 --update-crds \
    --deploy-dir=config --apis-dir=api --output-dir=production
    INFO[0000] Generating CSV manifest version 0.0.1
    ...

    $ tree config/ api/ production/
    config/
    ├── crds
    │   ├── cache.example.com_memcacheds_crd.yaml
    │   └── cache.example.com_v1alpha1_memcached_cr.yaml
    ├── operator.yaml
    ├── role.yaml
    ├── role_binding.yaml
    └── service_account.yaml
    api/
    ├── ...
    └── cache
    |   ├── group.go
    |   └── v1alpha1
    |       ├── ...
    |       └── memcached_types.go
    production/
    └── olm-catalog
        └── memcached-operator
            ├── 0.0.1
            │   ├── cache.example.com_memcacheds_crd.yaml
            │   └── memcached-operator.v0.0.1.clusterserviceversion.yaml
            └── memcached-operator.package.yaml
`,

		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 0 {
				return fmt.Errorf("command %s doesn't accept any arguments", cmd.CommandPath())
			}
			if err := c.validate(); err != nil {
				return fmt.Errorf("error validating command flags: %v", err)
			}

			if err := projutil.CheckProjectRoot(); err != nil {
				log.Warn("Could not detect project root. Ensure that this command " +
					"runs from the project root directory.")
			}

			// Legacy behavior.
			if !c.makeManifests && !cmd.Flags().Changed("update-crds") {
				c.updateCRDs = false
			}

			// Check if the user has any specific preference to enable / disable interactive prompts.
			// Default behaviour is to disable the prompts.
			if cmd.Flags().Changed("interactive") {
				if c.interactive {
					c.interactivelevel = projutil.InteractiveOnAll
				} else {
					c.interactivelevel = projutil.InteractiveHardOff
				}
			}

			if err := c.run(); err != nil {
				log.Fatal(err)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&c.csvVersion, "csv-version", "",
		"Semantic version of the CSV. This flag must be set if a package manifest exists")
	cmd.Flags().StringVar(&c.fromVersion, "from-version", "",
		"Semantic version of an existing CSV to use as a base")

	// TODO: Allow multiple paths
	// Deployment and RBAC manifests might be in different dirs e.g kubebuilder
	cmd.Flags().StringVar(&c.deployDir, "deploy-dir", "deploy",
		`Project relative path to root directory for operator manifests (Deployment and RBAC)`)
	cmd.Flags().StringVar(&c.apisDir, "apis-dir", filepath.Join("pkg", "apis"),
		`Project relative path to root directory for API type defintions`)
	// TODO: Allow multiple paths
	// CRD and CR manifests might be in different dirs e.g kubebuilder
	cmd.Flags().StringVar(&c.crdDir, "crd-dir", "",
		`Project relative path to root directory for CRD and CR manifests`)

	cmd.Flags().StringVar(&c.outputDir, "output-dir", "",
		"Base directory to output generated CSV. If --make-manifests=false the resulting "+
			"CSV bundle directory will be <output-dir>/olm-catalog/<operator-name>/<csv-version>. "+
			"If --make-manifests=true, the bundle directory will be <output-dir>/manifests")
	cmd.Flags().StringVar(&c.operatorName, "operator-name", "",
		"Operator name to use while generating CSV")

	cmd.Flags().StringVar(&c.csvChannel, "csv-channel", "",
		"Channel the CSV should be registered under in the package manifest")
	cmd.Flags().BoolVar(&c.defaultChannel, "default-channel", false,
		"Use the channel passed to --csv-channel as the package manifests' default channel. "+
			"Only valid when --csv-channel is set")

	cmd.Flags().BoolVar(&c.updateCRDs, "update-crds", true,
		"Update CRD manifests in deploy/<operator-name>/<csv-version> from the default "+
			"CRDs dir deploy/crds or --crd-dir if set. If --make-manifests=false, this option "+
			"is false by default")
	cmd.Flags().BoolVar(&c.makeManifests, "make-manifests", true,
		"When set, the generator will create or update a CSV manifest in a 'manifests' "+
			"directory. This directory is intended to be used for your latest bundle manifests. "+
			"The default location is deploy/olm-catalog/<operator-name>/manifests. "+
			"If --output-dir is set, the directory will be <output-dir>/manifests")
	cmd.Flags().BoolVar(&c.interactive, "interactive", false,
		"When set, will enable the interactive command prompt feature to fill the UI "+
			"metadata fields in CSV")

	return cmd
}

func (c csvCmd) run() error {
	log.Infof("Generating CSV manifest version %s", c.csvVersion)

	// Default crdDir differently if deployDir is set, since the CRD manifest dir
	// is expected to be in a projects manifests (deploy) dir.
	if c.crdDir == "" {
		if c.deployDir != "" {
			c.crdDir = filepath.Join(c.deployDir, "crds")
		} else {
			c.crdDir = filepath.Join("deploy", "crds")
		}
	}

	if c.operatorName == "" {
		c.operatorName = filepath.Base(projutil.MustGetwd())
	}

	csv := gencatalog.BundleGenerator{
		OperatorName:          c.operatorName,
		CSVVersion:            c.csvVersion,
		FromVersion:           c.fromVersion,
		UpdateCRDs:            c.updateCRDs,
		MakeManifests:         c.makeManifests,
		DeployDir:             c.deployDir,
		ApisDir:               c.apisDir,
		CRDsDir:               c.crdDir,
		OutputDir:             c.outputDir,
		InteractivePreference: c.interactivelevel,
	}

	if err := csv.Generate(); err != nil {
		return fmt.Errorf("error generating CSV: %v", err)
	}

	// A package manifest file is not a part of the bundle format.
	if !c.makeManifests {
		pkg := gencatalog.PkgGenerator{
			OperatorName:     c.operatorName,
			CSVVersion:       c.csvVersion,
			OutputDir:        c.outputDir,
			Channel:          c.csvChannel,
			ChannelIsDefault: c.defaultChannel,
		}
		if err := pkg.Generate(); err != nil {
			return fmt.Errorf("error generating package manifest: %v", err)
		}
	}

	log.Info("CSV manifest generated successfully")

	return nil
}

func (c csvCmd) validate() error {
	// If a manifests directory exists, allow no versions to be set. In this case
	// either a new CSV will be created or existing CSV updated in the manifests dir.
	if c.csvVersion != "" {
		if err := validateVersion(c.csvVersion); err != nil {
			return err
		}
	}
	if c.fromVersion != "" {
		if err := validateVersion(c.fromVersion); err != nil {
			return err
		}
		if c.csvVersion == c.fromVersion {
			return fmt.Errorf("--from-version (%s) cannot equal --csv-version; set only csv-version instead",
				c.fromVersion)
		}
	}

	if c.defaultChannel && c.csvChannel == "" {
		return fmt.Errorf("default-channel can only be used if csv-channel is set")
	}

	return nil
}

func validateVersion(version string) error {
	v, err := semver.Parse(version)
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
