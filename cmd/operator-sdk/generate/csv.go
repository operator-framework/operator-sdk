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
	"github.com/operator-framework/operator-sdk/internal/util/k8sutil"
	kbutil "github.com/operator-framework/operator-sdk/internal/util/kubebuilder"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"

	"github.com/coreos/go-semver/semver"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// KB_INTEGRATION_TODO(estroz): if project is a kubebuilder project, change
// defaults. Also look into running this from a makefile or using kustomize to
// create bundle data.
// Once https://github.com/operator-framework/operator-sdk/pull/2776 is merged
// update more defaults.

type csvCmd struct {
	csvVersion     string
	csvChannel     string
	fromVersion    string
	operatorName   string
	outputDir      string
	deployDir      string
	apisDir        string
	crdDir         string
	updateCRDs     bool
	defaultChannel bool
}

func NewGenerateCSVCmd() *cobra.Command {
	c := &csvCmd{}
	defaults := getDefaults()

	cmd := &cobra.Command{
		Use:   "csv",
		Short: "Generates a ClusterServiceVersion YAML file for the operator",
		Long: `The 'generate csv' command generates a ClusterServiceVersion (CSV) YAML manifest
for the operator. This file is used to publish the operator to the OLM Catalog.

A CSV semantic version is supplied via the --csv-version flag. If your operator
has already generated a CSV manifest you want to use as a base, supply its
version to --from-version. Otherwise the SDK will scaffold a new CSV manifest.

CSV input flags:
	--deploy-dir:
		The CSV's install strategy and permissions will be generated from the operator manifests
		(Deployment and Role/ClusterRole) present in this directory.

	--apis-dir:
		The CSV annotation comments will be parsed from the Go types under this path to
		fill out metadata for owned APIs in spec.customresourcedefinitions.owned.

	--crd-dir:
		The CSV's spec.customresourcedefinitions.owned field is generated from the CRD manifests
		in this path.These CRD manifests are also copied over to the bundle directory if --update-crds is set.
		Additionally the CR manifests will be used to populate the CSV example CRs.
`,
		Example: `
		##### Generate CSV from default input paths #####
		$ tree pkg/apis/ deploy/
		pkg/apis/
		├── ...
		└── cache
			├── group.go
			└── v1alpha1
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

		$ operator-sdk generate csv --csv-version=0.0.1 --update-crds
		INFO[0000] Generating CSV manifest version 0.0.1
		...

		$ tree deploy/
		deploy/
		...
		├── olm-catalog
		│   └── memcached-operator
		│       ├── 0.0.1
		│       │   ├── cache.example.com_memcacheds_crd.yaml
		│       │   └── memcached-operator.v0.0.1.clusterserviceversion.yaml
		│       └── memcached-operator.package.yaml
		...



		##### Generate CSV from custom input paths #####
		$ operator-sdk generate csv --csv-version=0.0.1 --update-crds \
		--deploy-dir=config --apis-dir=api --output-dir=production
		INFO[0000] Generating CSV manifest version 0.0.1
		...

		$ tree config/ api/ production/
		config/
		├── crds
		│   ├── cache.example.com_memcacheds_crd.yaml
		│   └── cache.example.com_v1alpha1_memcached_cr.yaml
		├── operator.yaml
		├── role.yaml
		├── role_binding.yaml
		└── service_account.yaml
		api/
		├── ...
		└── cache
			├── group.go
			└── v1alpha1
				├── ...
				└── memcached_types.go
		production/
		└── olm-catalog
			└── memcached-operator
				├── 0.0.1
				│   ├── cache.example.com_memcacheds_crd.yaml
				│   └── memcached-operator.v0.0.1.clusterserviceversion.yaml
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

			if err := c.run(); err != nil {
				log.Fatal(err)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&c.csvVersion, "csv-version", "",
		"Semantic version of the CSV")
	if err := cmd.MarkFlagRequired("csv-version"); err != nil {
		log.Fatalf("Failed to mark `csv-version` flag for `generate csv` subcommand as required: %v", err)
	}
	cmd.Flags().StringVar(&c.fromVersion, "from-version", "",
		"Semantic version of an existing CSV to use as a base")

	// TODO: Allow multiple paths
	// Deployment and RBAC manifests might be in different dirs e.g kubebuilder
	cmd.Flags().StringVar(&c.deployDir, "deploy-dir", defaults.deployDir,
		`Project relative path to root directory for operator manifests (Deployment and RBAC)`)
	cmd.Flags().StringVar(&c.apisDir, "apis-dir", defaults.apisDir,
		`Project relative path to root directory for API type defintions`)
	// TODO: Allow multiple paths
	// CRD and CR manifests might be in different dirs e.g kubebuilder
	cmd.Flags().StringVar(&c.crdDir, "crd-dir", "",
		`Project relative path to root directory for CRD and CR manifests`)

	cmd.Flags().StringVar(&c.outputDir, "output-dir", defaults.deployDir,
		"Base directory to output generated CSV. The resulting CSV bundle directory "+
			"will be \"<output-dir>/olm-catalog/<operator-name>/<csv-version>\".")
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
	cfg := gen.Config{
		OperatorName: c.operatorName,
		// TODO(hasbro17): Remove the Input key map when the Generator input keys
		// are removed in favour of config fields in the csvGenerator
		Inputs: map[string]string{
			gencatalog.DeployDirKey: c.deployDir,
			gencatalog.APIsDirKey:   c.apisDir,
			gencatalog.CRDsDirKey:   c.crdDir,
		},
		OutputDir: c.outputDir,
	}

	csv := gencatalog.NewCSV(cfg, c.csvVersion, c.fromVersion)
	if err := csv.Generate(); err != nil {
		return fmt.Errorf("error generating CSV: %v", err)
	}
	pkg := gencatalog.NewPackageManifest(cfg, c.csvVersion, c.csvChannel, c.defaultChannel)
	if err := pkg.Generate(); err != nil {
		return fmt.Errorf("error generating package manifest: %v", err)
	}

	// Write CRD's to the new or updated CSV package dir.
	if c.updateCRDs {
		// TODO: This path should come from the CSV generator field csvOutputDir
		bundleDir := filepath.Join(c.outputDir, gencatalog.OLMCatalogChildDir, c.operatorName, c.csvVersion)
		if err := copyCustomResourceDefinitions(c.crdDir, bundleDir); err != nil {
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

// copyCustomResourceDefinitions copies all CustomResourceDefinition manifests
// from fromDir to toDir with the same file name.
func copyCustomResourceDefinitions(fromDir, toDir string) error {
	infos, err := ioutil.ReadDir(fromDir)
	if err != nil {
		return err
	}

	for _, info := range infos {
		if info.IsDir() {
			continue
		}

		fromPath := filepath.Join(fromDir, info.Name())
		b, err := ioutil.ReadFile(fromPath)
		if err != nil {
			return fmt.Errorf("error reading manifest %s: %v", fromPath, err)
		}

		scanner := k8sutil.NewYAMLScanner(b)
		manifests := []byte{}
		for scanner.Scan() {
			manifest := scanner.Bytes()
			typeMeta, err := k8sutil.GetTypeMetaFromBytes(manifest)
			if err != nil {
				log.Debugf("Skipping non-manifest file %s: %v", fromPath, err)
				continue
			}
			if typeMeta.Kind == "CustomResourceDefinition" {
				manifests = k8sutil.CombineManifests(manifests, b)
			}
		}
		if err = scanner.Err(); err != nil {
			return err
		}

		if len(manifests) != 0 {
			toPath := filepath.Join(toDir, info.Name())
			if err = ioutil.WriteFile(toPath, manifests, info.Mode()); err != nil {
				return fmt.Errorf("error writing CRD %s: %v", toPath, err)
			}
		}
	}

	return nil
}

func getDefaults() *csvCmd {
	c := &csvCmd{}

	if kbutil.IsConfigExist() {
		cfg, err := kbutil.ReadConfig()
		if err != nil {
			log.Fatal(err)
		}
		c.operatorName = filepath.Base(cfg.Repo)
		c.deployDir = "config"
		c.crdDir = filepath.Join("config", "bases")
		if cfg.MultiGroup {
			c.apisDir = "apis"
		} else {
			c.apisDir = "api"
		}
	} else {
		c.operatorName = filepath.Base(projutil.MustGetwd())
		// For generating CLI docs.
		if c.operatorName == "operator-sdk" {
			c.operatorName = "test-operator"
		}
		c.deployDir = "deploy"
		c.crdDir = filepath.Join("deploy", "crds")
		c.apisDir = filepath.Join("pkg", "apis")
	}

	return c
}
