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

	"github.com/operator-framework/operator-registry/pkg/lib/bundle"
	log "github.com/sirupsen/logrus"

	genutil "github.com/operator-framework/operator-sdk/cmd/operator-sdk/generate/internal"
	gencsv "github.com/operator-framework/operator-sdk/internal/generate/clusterserviceversion"
	"github.com/operator-framework/operator-sdk/internal/generate/collector"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"
)

// setCommonDefaultsLegacy sets defaults useful to all modes of this subcommand.
func (c *bundleCmd) setCommonDefaultsLegacy() {
	if c.operatorName == "" {
		c.operatorName = filepath.Base(projutil.MustGetwd())
	}
}

// validateManifestsLegacy validates c for bundle manifests generation for
// legacy project layouts.
func (c bundleCmd) validateManifestsLegacy() error {
	if c.version != "" {
		if err := genutil.ValidateVersion(c.version); err != nil {
			return err
		}
	}
	return nil
}

// runManifestsLegacy generates bundle manifests for legacy project layouts.
func (c bundleCmd) runManifestsLegacy() (err error) {

	if !c.quiet {
		if c.version == "" {
			log.Info("Generating bundle manifests")
		} else {
			log.Info("Generating bundle manifests version", c.version)
		}
	}

	if c.apisDir == "" {
		c.apisDir = filepath.Join("pkg", "apis")
	}
	if c.manifestRoot == "" {
		c.manifestRoot = "deploy"
	}
	if c.crdsDir == "" {
		c.crdsDir = filepath.Join(c.manifestRoot, "crds")
	}
	defaultBundleDir := filepath.Join(c.manifestRoot, "olm-catalog", c.operatorName)
	if c.inputDir == "" {
		c.inputDir = defaultBundleDir
	}
	if c.outputDir == "" {
		c.outputDir = defaultBundleDir
	}

	col := &collector.Manifests{}
	if err := col.UpdateFromDirs(c.manifestRoot, c.crdsDir); err != nil {
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

// runMetadataLegacy generates a bundle.Dockerfile and bundle metadata for
// legacy project layouts.
func (c bundleCmd) runMetadataLegacy() error {

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
