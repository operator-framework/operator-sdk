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
	"github.com/operator-framework/operator-sdk/internal/scaffold/kustomize"
)

// TODO: make these paths relative to inputDir
const kubebuilderKustomization = `resources:
- ../default
- ../samples
`

func (c *packagemanifestsCmd) setDefaults(cfg *config.Config) {
	if c.operatorName == "" {
		c.operatorName = filepath.Base(cfg.Repo)
	}
}

func (c packagemanifestsCmd) runKustomize(cfg *config.Config) error {

	if !c.quiet {
		fmt.Println("Generating package manifests format kustomize bases")
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
		gencsv.WithBase(c.inputDir, c.apisDir),
		gencsv.WithBaseWriter(c.outputDir),
	}
	if err := csvGen.Generate(cfg, opts...); err != nil {
		return fmt.Errorf("error generating ClusterServiceVersion: %v", err)
	}

	if err := kustomize.WriteIfNotExist(c.outputDir, kubebuilderKustomization); err != nil {
		return err
	}

	if !c.quiet {
		fmt.Println("Bases generated successfully")
	}

	return nil
}

func (c packagemanifestsCmd) validateManifests() error {

	if err := genutil.ValidateVersion(c.version); err != nil {
		return err
	}

	if c.fromVersion != "" {
		return errors.New("--from-version cannot be set for PROJECT configured projects")
	}

	if !genutil.IsPipeReader() {
		if c.manifestRoot == "" {
			return errors.New("--manifest-root must be set if not reading from stdin")
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

func (c packagemanifestsCmd) runManifests(cfg *config.Config) error {

	if !c.quiet {
		fmt.Printf("Generating package version %s\n", c.version)
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
		gencsv.WithBase(c.inputDir, c.apisDir),
	}
	if c.stdout {
		opts = append(opts, gencsv.WithWriter(stdout))
	} else {
		opts = append(opts, gencsv.WithPackageWriter(c.outputDir))
	}

	if err := csvGen.Generate(cfg, opts...); err != nil {
		return fmt.Errorf("error generating ClusterServiceVersion: %v", err)
	}

	if c.stdout {
		if err := genutil.WriteCRDs(stdout, col.CustomResourceDefinitions...); err != nil {
			return err
		}
	} else if c.updateCRDs {
		dir := filepath.Join(c.outputDir, c.version)
		if err := genutil.WriteCRDFiles(dir, col.CustomResourceDefinitions...); err != nil {
			return err
		}
	}

	if !c.quiet {
		fmt.Println("Package generated successfully")
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
		genpkg.WithGetBase(c.inputDir),
		genpkg.WithFileWriter(c.outputDir),
	}
	if err := pkgGen.Generate(opts...); err != nil {
		return err
	}
	return nil
}
