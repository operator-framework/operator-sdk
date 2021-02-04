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

	metricsannotations "github.com/operator-framework/operator-sdk/internal/annotations/metrics"
	genutil "github.com/operator-framework/operator-sdk/internal/cmd/operator-sdk/generate/internal"
	gencsv "github.com/operator-framework/operator-sdk/internal/generate/clusterserviceversion"
	"github.com/operator-framework/operator-sdk/internal/generate/clusterserviceversion/bases"
	"github.com/operator-framework/operator-sdk/internal/generate/collector"
	genpkg "github.com/operator-framework/operator-sdk/internal/generate/packagemanifest"
)

const (
	longHelp = `
Note: while the package manifests format is not yet deprecated, the operator-framework is migrated
towards using bundles by default. Run 'operator-sdk generate bundle -h' for more information.

Running 'generate packagemanifests' is the first step to publishing your operator to a catalog and/or deploying
it with OLM. This command generates a set of manifests in a versioned directory and a package manifest file for
your operator. Typically one would run 'generate kustomize manifests' first to (re)generate kustomize bases
consumed by this command.

Set '--version' to supply a semantic version for your new package.

More information on the package manifests format:
https://github.com/operator-framework/operator-registry/#manifest-format
`

	examples = `
  # Generate manifests then create the package manifests base:
  $ make manifests
  /home/user/go/bin/controller-gen "crd:trivialVersions=true" rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases
  $ operator-sdk generate kustomize manifests

  Display name for the operator (required):
  > memcached-operator
  ...

  $ tree config/manifests
  config/manifests
  ├── bases
  │   └── memcached-operator.clusterserviceversion.yaml
  └── kustomization.yaml
  $ kustomize build config/manifests | operator-sdk generate packagemanifests --version 0.0.1
  Generating package manifests version 0.0.1
  ...

  # After running the above commands, you should see this directory structure:
  $ tree packagemanifests
  packagemanifests
  ├── 0.0.1
  │   ├── cache.my.domain_memcacheds.yaml
  │   └── memcached-operator.clusterserviceversion.yaml
  └── memcached-operator.package.yaml
`
)

// defaultRootDir is the default root directory in which to generate package manifests files.
const defaultRootDir = "packagemanifests"

// setDefaults sets command defaults.
func (c *packagemanifestsCmd) setDefaults() (err error) {
	if c.packageName, c.layout, err = genutil.GetPackageNameAndLayout(c.packageName); err != nil {
		return err
	}

	if c.inputDir == "" {
		c.inputDir = defaultRootDir
	}
	if !c.stdout {
		if c.outputDir == "" {
			c.outputDir = defaultRootDir
		}
	}
	c.generator = genpkg.NewGenerator()

	return nil
}

// validate validates c for package manifests generation.
func (c packagemanifestsCmd) validate() error {

	if c.version != "" {
		if err := genutil.ValidateVersion(c.version); err != nil {
			return err
		}
	} else {
		return errors.New("--version must be set")
	}

	if c.fromVersion != "" {
		if err := genutil.ValidateVersion(c.fromVersion); err != nil {
			return err
		}
	}

	if c.inputDir == "" {
		return errors.New("--input-dir must be set")
	}
	if c.kustomizeDir == "" {
		return errors.New("--kustomize-dir must be set")
	}

	if !genutil.IsPipeReader() {
		if c.deployDir == "" {
			return errors.New("--deploy-dir must be set if not reading from stdin")
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

	if c.isDefaultChannel && c.channelName == "" {
		return fmt.Errorf("--default-channel can only be set if --channel is set")
	}

	return nil
}

// run generates package manifests.
func (c packagemanifestsCmd) run() error {

	if !c.quiet && !c.stdout {
		fmt.Println("Generating package manifests version", c.version)
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

	// If no CSV is passed via stdin, an on-disk base is expected at basePath.
	if len(col.ClusterServiceVersions) == 0 {
		basePath := filepath.Join(c.kustomizeDir, "bases", c.packageName+".clusterserviceversion.yaml")
		if genutil.IsExist(basePath) {
			base, err := bases.ClusterServiceVersion{BasePath: basePath}.GetBase()
			if err != nil {
				return fmt.Errorf("error reading CSV base: %v", err)
			}
			col.ClusterServiceVersions = append(col.ClusterServiceVersions, *base)
		}
	}

	var opts []gencsv.Option
	stdout := genutil.NewMultiManifestWriter(os.Stdout)
	if c.stdout {
		opts = append(opts, gencsv.WithWriter(stdout))
	} else {
		opts = append(opts, gencsv.WithPackageWriter(c.outputDir))
	}

	csvGen := gencsv.Generator{
		OperatorName: c.packageName,
		Version:      c.version,
		FromVersion:  c.fromVersion,
		Collector:    col,
		Annotations:  metricsannotations.MakeBundleObjectAnnotations(c.layout),
	}
	if err := csvGen.Generate(opts...); err != nil {
		return fmt.Errorf("error generating ClusterServiceVersion: %v", err)
	}

	if c.updateObjects {
		objs := genutil.GetManifestObjects(col)
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
	//copy of genpkg withfilewriter()
	//move out of internal util pkg?
	if err := os.MkdirAll(c.outputDir, 0755); err != nil {
		return err
	}

	opts := genpkg.Options{
		BaseDir:          c.inputDir,
		ChannelName:      c.channelName,
		IsDefaultChannel: c.isDefaultChannel,
	}

	if err := c.generator.Generate(c.packageName, c.version, c.outputDir, opts); err != nil {
		return err
	}
	return nil
}
