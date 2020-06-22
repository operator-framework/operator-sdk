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

package bundle

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/operator-framework/operator-registry/pkg/lib/bundle"
	log "github.com/sirupsen/logrus"

	genutil "github.com/operator-framework/operator-sdk/cmd/operator-sdk/generate/internal"
	gencsv "github.com/operator-framework/operator-sdk/internal/generate/clusterserviceversion"
	"github.com/operator-framework/operator-sdk/internal/generate/collector"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"
)

const (
	longHelpLegacy = `
Running 'generate bundle' is the first step to publishing your operator to a catalog
and/or deploying it with OLM. This command generates a set of bundle manifests,
metadata, and a bundle.Dockerfile for your operator, and will interactively ask
for UI metadata, an important component of publishing your operator, by default unless
a bundle for your operator exists or you set '--interactive=false'.

Set '--version' to supply a semantic version for your bundle if you are creating one
for the first time or upgrading an existing one.

If '--output-dir' is set and you wish to build bundle images from that directory,
either manually update your bundle.Dockerfile or set '--overwrite'.

More information on bundles:
https://github.com/operator-framework/operator-registry/#manifest-format
`

	examplesLegacy = `
  # Create bundle manifests, metadata, and a bundle.Dockerfile:
  $ operator-sdk generate bundle --version 0.0.1
  INFO[0000] Generating bundle manifest version 0.0.1

  Display name for the operator (required):
  > memcached-operator
  ...

  # After running the above commands, you should see:
  $ tree deploy/olm-catalog
  deploy/olm-catalog
  └── memcached-operator
      ├── manifests
      │   ├── cache.example.com_memcacheds_crd.yaml
      │   └── memcached-operator.clusterserviceversion.yaml
      └── metadata
          └── annotations.yaml

  # Then build and push your bundle image:
  $ export USERNAME=<your registry username>
  $ export BUNDLE_IMG=quay.io/$USERNAME/memcached-operator-bundle:v0.0.1
  $ docker build -f bundle.Dockerfile -t $BUNDLE_IMG .
  Sending build context to Docker daemon  42.33MB
  Step 1/9 : FROM scratch
  ...
  $ docker push $BUNDLE_IMG
`
)

// setCommonDefaultsLegacy sets defaults useful to all modes of this subcommand.
func (c *bundleCmdLegacy) setCommonDefaults() {
	if c.operatorName == "" {
		c.operatorName = filepath.Base(projutil.MustGetwd())
	}
	// A default channel can be inferred if there is only one channel. Don't infer
	// default otherwise; the user must set this value.
	if c.defaultChannel == "" && strings.Count(c.channels, ",") == 0 {
		c.defaultChannel = c.channels
	}
}

// validateManifestsLegacy validates c for bundle manifests generation for
// legacy project layouts.
func (c bundleCmdLegacy) validateManifests() error {
	if c.version != "" {
		if err := genutil.ValidateVersion(c.version); err != nil {
			return err
		}
	}
	return nil
}

// runManifestsLegacy generates bundle manifests for legacy project layouts.
func (c bundleCmdLegacy) runManifests() (err error) {

	if !c.quiet {
		if c.version == "" {
			log.Info("Generating bundle manifests")
		} else {
			log.Infoln("Generating bundle manifests version", c.version)
		}
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

	col := &collector.Manifests{}
	if err := col.UpdateFromDirs(c.deployDir, c.crdsDir); err != nil {
		return err
	}

	csvGen := gencsv.Generator{
		OperatorName: c.operatorName,
		OperatorType: projutil.GetOperatorType(),
		Version:      c.version,
		Collector:    col,
	}

	opts := []gencsv.LegacyOption{
		gencsv.WithBundleBase(c.inputDir, c.apisDir, c.interactiveLevel),
		gencsv.LegacyOption(gencsv.WithBundleWriter(c.outputDir)),
	}
	if err := csvGen.GenerateLegacy(opts...); err != nil {
		return fmt.Errorf("error generating ClusterServiceVersion: %v", err)
	}

	var objs []interface{}
	for _, crd := range col.V1CustomResourceDefinitions {
		objs = append(objs, crd)
	}
	for _, crd := range col.V1beta1CustomResourceDefinitions {
		objs = append(objs, crd)
	}
	dir := filepath.Join(c.outputDir, bundle.ManifestsDir)
	if err := genutil.WriteObjectsToFilesLegacy(dir, objs...); err != nil {
		return err
	}

	if !c.quiet {
		log.Infoln("Bundle manifests generated successfully in", c.outputDir)
	}

	return nil
}

// validateMetadataLegacy validates c for bundle metadata generation for
// legacy project layouts.
func (c bundleCmdLegacy) validateMetadata() (err error) {
	// Ensure a default channel is present.
	if c.defaultChannel == "" {
		return fmt.Errorf("--default-channel must be set if setting multiple channels")
	}

	return nil
}

// runMetadataLegacy generates a bundle.Dockerfile and bundle metadata for
// legacy project layouts.
func (c bundleCmdLegacy) runMetadata() error {

	directory := c.inputDir
	if directory == "" {
		// There may be no existing bundle at the default path, so assume manifests
		// were generated in the output directs.
		defaultDirectory := filepath.Join("deploy", "olm-catalog", c.operatorName, bundle.ManifestsDir)
		if c.outputDir != "" && genutil.IsNotExist(defaultDirectory) {
			directory = filepath.Join(c.outputDir, bundle.ManifestsDir)
		} else {
			directory = defaultDirectory
		}
	} else {
		directory = filepath.Join(directory, bundle.ManifestsDir)
	}
	outputDir := c.outputDir
	if filepath.Clean(outputDir) == filepath.Clean(directory) {
		outputDir = ""
	}

	return c.generateMetadata(directory, outputDir)
}
