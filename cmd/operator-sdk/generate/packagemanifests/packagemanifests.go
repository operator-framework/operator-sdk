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
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"sigs.k8s.io/kubebuilder/pkg/model/config"

	genutil "github.com/operator-framework/operator-sdk/cmd/operator-sdk/generate/internal"
	gencsv "github.com/operator-framework/operator-sdk/internal/generate/clusterserviceversion"
	"github.com/operator-framework/operator-sdk/internal/generate/collector"
	genpkg "github.com/operator-framework/operator-sdk/internal/generate/packagemanifest"
)

// setCommonDefaults sets defaults useful to all modes of this subcommand.
func (c *packagemanifestsCmd) setCommonDefaults(cfg *config.Config) {
	if c.operatorName == "" {
		c.operatorName = filepath.Base(cfg.Repo)
	}
}

// runKustomize generates kustomize package bases.
func (c packagemanifestsCmd) runKustomize(cfg *config.Config) error {

	if !c.quiet {
		fmt.Println("Generating package manifests kustomize bases")
	}

	defaultDir := filepath.Join("config", "packages")
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

// validateManifests validates c for package manifests generation.
func (c packagemanifestsCmd) validateManifests() error {

	if err := genutil.ValidateVersion(c.version); err != nil {
		return err
	}

	if c.fromVersion != "" {
		return errors.New("--from-version cannot be set for PROJECT configured projects")
	}

	if !genutil.IsPipeReader() {
		if c.deployDir == "" {
			return errors.New("--deploy-dir must be set if not reading from stdin")
		}
		if c.crdsDir == "" {
			return errors.New("--crd-dir must be set if not reading from stdin")
		}
	}

	if c.stdout {
		if c.outputDir != "" {
			return errors.New("--output-dir cannot be set if writing to stdout")
		}
	}

	if c.isDefaultChannel && c.channelName == "" {
		return fmt.Errorf("--default-channel can only be set if --channel is set")
	}

	return nil
}

// runManifests generates package manifests.
func (c packagemanifestsCmd) runManifests(cfg *config.Config) error {

	if !c.quiet && !c.stdout {
		fmt.Println("Generating package manifests version", c.version)
	}

	defaultDir := filepath.Join("config", "packages")
	if c.inputDir == "" {
		c.inputDir = defaultDir
	}
	if !c.stdout {
		if c.outputDir == "" {
			c.outputDir = defaultDir
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

	if err := c.generatePackageManifest(); err != nil {
		return err
	}

	col := &collector.Manifests{}
	if genutil.IsPipeReader() {
		if err := col.UpdateFromReader(os.Stdin); err != nil {
			return err
		}
	}
	if c.deployDir != "" {
		if err := col.UpdateFromDirs(c.deployDir, c.crdsDir); err != nil {
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
		opts = append(opts, gencsv.WithPackageWriter(c.outputDir))
	}

	if err := csvGen.Generate(cfg, opts...); err != nil {
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
		if c.stdout {
			if err := genutil.WriteObjects(stdout, objs...); err != nil {
				return err
			}
		} else {
			dir := filepath.Join(c.outputDir, c.version)
			if err := genutil.WriteObjectsToFiles(dir, objs...); err != nil {
				return err
			}
		}
	}

	if !c.quiet && !c.stdout {
		fmt.Println("Package manifests generated successfully in", c.outputDir)
	}

	return nil
}

func (c packagemanifestsCmd) generatePackageManifest() error {
	pkgGen := genpkg.Generator{
		OperatorName:     c.operatorName,
		Version:          c.version,
		ChannelName:      c.channelName,
		IsDefaultChannel: c.isDefaultChannel,
	}
	opts := []genpkg.Option{
		genpkg.WithBase(c.inputDir),
		genpkg.WithFileWriter(c.outputDir),
	}
	if err := pkgGen.Generate(opts...); err != nil {
		return err
	}
	return nil
}
