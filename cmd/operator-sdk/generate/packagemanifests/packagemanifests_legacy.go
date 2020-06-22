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

package packagemanifests

import (
	"fmt"
	"path/filepath"

	log "github.com/sirupsen/logrus"

	genutil "github.com/operator-framework/operator-sdk/cmd/operator-sdk/generate/internal"
	gencsv "github.com/operator-framework/operator-sdk/internal/generate/clusterserviceversion"
	"github.com/operator-framework/operator-sdk/internal/generate/collector"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"
)

const (
	longHelpLegacy = `
Note: while the package manifests format is not yet deprecated, the operator-framework is migrated
towards using bundles by default. Run 'operator-sdk generate bundle -h' for more information.

Running 'generate packagemanifests' is the first step to publishing your operator to a catalog
and/or deploying it with OLM. This command generates a set of manifests in a versioned directory
and a package manifest file for your operator. It will interactively ask for UI metadata,
an important component of publishing your operator, by default unless a package for your
operator exists or you set '--interactive=false'.

Set '--version' to supply a semantic version for your new package. This is a required flag when running
'generate packagemanifests --manifests'.

More information on the package manifests format:
https://github.com/operator-framework/operator-registry/#manifest-format
`

	examplesLegacy = `
  # Create the package manifest file and a new package:
  $ operator-sdk generate packagemanifests --version 0.0.1
  INFO[0000] Generating package manifests version 0.0.1

  Display name for the operator (required):
  > memcached-operator
  ...

  # After running the above commands, you should see:
  $ tree deploy/olm-catalog
  deploy/olm-catalog
  └── memcached-operator
      ├── 0.0.1
      │   ├── cache.example.com_memcacheds_crd.yaml
      │   └── memcached-operator.clusterserviceversion.yaml
      └── memacached-operator.package.yaml
`
)

// setDefaults sets defaults useful to all modes of this subcommand for legacy project layouts.
func (c *packagemanifestsCmdLegacy) setDefaults() {
	if c.operatorName == "" {
		c.operatorName = filepath.Base(projutil.MustGetwd())
	}

	if c.apisDir == "" {
		c.apisDir = filepath.Join("pkg", "apis")
	}
	if c.deployDir == "" {
		c.deployDir = "deploy"
	}
	if c.crdsDir == "" {
		c.crdsDir = filepath.Join(c.deployDir, "crds")
	}

	defaultBundleDir := filepath.Join(c.deployDir, "olm-catalog", c.operatorName)
	if c.inputDir == "" {
		c.inputDir = defaultBundleDir
	}
	if c.outputDir == "" {
		c.outputDir = defaultBundleDir
	}
}

// validate validates c for package manifests generation for legacy project layouts.
func (c packagemanifestsCmdLegacy) validate() error {

	if err := genutil.ValidateVersion(c.version); err != nil {
		return err
	}
	if c.fromVersion != "" {
		if err := genutil.ValidateVersion(c.fromVersion); err != nil {
			return err
		}
	}

	if c.isDefaultChannel && c.channelName == "" {
		return fmt.Errorf("--default-channel can only be set if --channel is set")
	}

	return nil
}

// run generates package manifests for legacy project layouts.
func (c packagemanifestsCmdLegacy) run() error {

	if !c.quiet {
		log.Infoln("Generating package manifests version", c.version)
	}

	if err := c.generatePackageManifest(); err != nil {
		return err
	}

	col := &collector.Manifests{}
	if err := col.UpdateFromDirs(c.deployDir, c.crdsDir); err != nil {
		return err
	}

	csvGen := gencsv.Generator{
		OperatorName: c.operatorName,
		OperatorType: projutil.GetOperatorType(),
		Version:      c.version,
		FromVersion:  c.fromVersion,
		Collector:    col,
	}

	opts := []gencsv.LegacyOption{
		gencsv.WithPackageBase(c.inputDir, c.apisDir, c.interactiveLevel),
		gencsv.LegacyOption(gencsv.WithPackageWriter(c.outputDir)),
	}
	if err := csvGen.GenerateLegacy(opts...); err != nil {
		return fmt.Errorf("error generating ClusterServiceVersion: %v", err)
	}

	if c.updateCRDs {
		var objs []interface{}
		for _, crd := range col.V1CustomResourceDefinitions {
			objs = append(objs, crd)
		}
		for _, crd := range col.V1beta1CustomResourceDefinitions {
			objs = append(objs, crd)
		}
		dir := filepath.Join(c.outputDir, c.version)
		if err := genutil.WriteObjectsToFilesLegacy(dir, objs...); err != nil {
			return err
		}
	}

	if !c.quiet {
		log.Infoln("Package manifests generated successfully in", c.outputDir)
	}

	return nil
}
