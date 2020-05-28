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
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/operator-framework/operator-registry/pkg/lib/bundle"
	"sigs.k8s.io/kubebuilder/pkg/model/config"

	genutil "github.com/operator-framework/operator-sdk/cmd/operator-sdk/generate/internal"
	gencsv "github.com/operator-framework/operator-sdk/internal/generate/clusterserviceversion"
	"github.com/operator-framework/operator-sdk/internal/generate/collector"
)

// setCommonDefaults sets defaults useful to all modes of this subcommand.
func (c *bundleCmd) setCommonDefaults(cfg *config.Config) {
	if c.operatorName == "" {
		c.operatorName = filepath.Base(cfg.Repo)
	}
}

// runKustomize generates kustomize bundle bases.
func (c bundleCmd) runKustomize(cfg *config.Config) error {

	if !c.quiet {
		fmt.Println("Generating bundle manifest kustomize bases")
	}

	defaultDir := filepath.Join("config", "bundle")
	if c.inputDir == "" {
		c.inputDir = defaultDir
	}
	if c.outputDir == "" {
		c.outputDir = defaultDir
	}
	if c.apisDir == "" {
		if cfg.MultiGroup {
			c.apisDir = "apis"
		} else {
			c.apisDir = "api"
		}
	}

	csvGen := gencsv.Generator{
		OperatorName: c.operatorName,
		OperatorType: genutil.PluginKeyToOperatorType(cfg.Layout),
	}
	opts := []gencsv.Option{
		gencsv.WithBase(c.inputDir, c.apisDir, c.interactiveLevel),
		gencsv.WithBaseWriter(c.outputDir),
	}
	if err := csvGen.Generate(cfg, opts...); err != nil {
		return fmt.Errorf("error generating ClusterServiceVersion: %v", err)
	}

	if !c.quiet {
		fmt.Println("Bases generated successfully in", c.outputDir)
	}

	return nil
}

// validateManifests validates c for bundle manifests generation.
func (c bundleCmd) validateManifests(*config.Config) (err error) {
	if c.version != "" {
		if err := genutil.ValidateVersion(c.version); err != nil {
			return err
		}
	}

	if !genutil.IsPipeReader() {
		if c.manifestRoot == "" {
			return errors.New("--manifest-root must be set if not reading from stdin")
		}
		if c.crdsDir == "" {
			return errors.New("--crds-dir must be set if not reading from stdin")
		}
	}

	if c.stdout {
		if c.outputDir != "" {
			return errors.New("--output-dir cannot be set if writing to stdout")
		}
	}

	return nil
}

// runManifests generates bundle manifests.
func (c bundleCmd) runManifests(cfg *config.Config) (err error) {

	if !c.quiet && !c.stdout {
		if c.version == "" {
			fmt.Println("Generating bundle manifests")
		} else {
			fmt.Println("Generating bundle manifests version", c.version)
		}
	}

	defaultBundleDir := filepath.Join("config", "bundle")
	if c.inputDir == "" {
		c.inputDir = defaultBundleDir
	}
	if !c.stdout {
		if c.outputDir == "" {
			c.outputDir = defaultBundleDir
		}
	}
	// Only regenerate API definitions once.
	if c.apisDir == "" && !c.kustomize {
		if cfg.MultiGroup {
			c.apisDir = "apis"
		} else {
			c.apisDir = "api"
		}
	}

	col := &collector.Manifests{}
	if genutil.IsPipeReader() {
		if err := col.UpdateFromReader(os.Stdin); err != nil {
			return err
		}
	}
	if c.manifestRoot != "" {
		if err := col.UpdateFromDirs(c.manifestRoot, c.crdsDir); err != nil {
			return err
		}
	}

	csvGen := gencsv.Generator{
		OperatorName: c.operatorName,
		OperatorType: genutil.PluginKeyToOperatorType(cfg.Layout),
		Version:      c.version,
		Collector:    col,
	}

	stdout := genutil.NewMultiManifestWriter(os.Stdout)
	opts := []gencsv.Option{
		gencsv.WithBase(c.inputDir, c.apisDir, c.interactiveLevel),
	}
	if c.stdout {
		opts = append(opts, gencsv.WithWriter(stdout))
	} else {
		opts = append(opts, gencsv.WithBundleWriter(c.outputDir))
	}

	if err := csvGen.Generate(cfg, opts...); err != nil {
		return fmt.Errorf("error generating ClusterServiceVersion: %v", err)
	}

	var objs []interface{}
	for _, crd := range col.V1CustomResourceDefinitions {
		objs = append(objs, crd)
	}
	for _, crd := range col.V1beta1CustomResourceDefinitions {
		objs = append(objs, crd)
	}
	if c.stdout {
		if err := genutil.WriteObjects(stdout, objs...); err != nil {
			return err
		}
	} else {
		dir := filepath.Join(c.outputDir, bundle.ManifestsDir)
		if err := genutil.WriteObjectsToFiles(dir, objs...); err != nil {
			return err
		}
	}

	if !c.quiet && !c.stdout {
		fmt.Println("Bundle manifests generated successfully in", c.outputDir)
	}

	return nil
}

// runMetadata generates a bundle.Dockerfile and bundle metadata.
func (c bundleCmd) runMetadata() error {

	directory := c.inputDir
	if directory == "" {
		// There may be no existing bundle at the default path, so assume manifests
		// only exist in the output directory.
		defaultDirectory := filepath.Join("config", "bundle", bundle.ManifestsDir)
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

// generateMetadata wraps the operator-registry bundle Dockerfile/metadata generator.
func (c bundleCmd) generateMetadata(manifestsDir, outputDir string) error {
	err := bundle.GenerateFunc(manifestsDir, outputDir, c.operatorName, c.channels, c.defaultChannel, c.overwrite)
	if err != nil {
		return fmt.Errorf("error generating bundle metadata: %v", err)
	}
	return nil
}
