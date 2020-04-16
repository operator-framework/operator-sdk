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
	"strings"

	"github.com/operator-framework/operator-registry/pkg/lib/bundle"
	"sigs.k8s.io/kubebuilder/pkg/model/config"

	genutil "github.com/operator-framework/operator-sdk/cmd/operator-sdk/generate/internal"
	gencsv "github.com/operator-framework/operator-sdk/internal/generate/clusterserviceversion"
	"github.com/operator-framework/operator-sdk/internal/generate/collector"
)

const (
	//nolint:lll
	examples = `
  # Using the example 'memcached-operator' and assuming a directory structure
  # similar to the following exists:
  $ tree api/ config/
  api/
  └── v1alpha1
      ├── groupversion_info.go
      ├── memcached_types.go
      └── zz_generated.deepcopy.go
  config/
  ├── bundle
  │   └── kustomization.yaml
  ├── crd
  │   ├── bases
  │   │   └── cache.my.domain_memcacheds.yaml
  │   ├── kustomization.yaml
  │   ├── kustomizeconfig.yaml
  │   ...
  ├── default
  │   ├── kustomization.yaml
  │   ...
  ├── manager
  │   ├── kustomization.yaml
  │   └── manager.yaml
  ...

  # Generate bundle files and build your bundle image with these 'make' recipes:
  $ make bundle
  $ export USERNAME=<your registry username>
  $ export BUNDLE_IMG=quay.io/$USERNAME/memcached-operator-bundle:v0.0.1
  $ make bundle-build BUNDLE_IMG=$BUNDLE_IMG

  # The above recipe runs the following commands manually. First it creates bundle
  # manifests, metadata, and a bundle.Dockerfile:
  $ make manifests
  /home/user/go/bin/controller-gen "crd:trivialVersions=true" rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases
  $ operator-sdk generate bundle -q --kustomize

  Display name for the operator (required):
  > memcached-operator
  ...

  $ kustomize build config/bundle | operator-sdk generate bundle --manifests --metadata --overwrite --version 0.0.1
  Generating bundle manifest version 0.0.1
  ...

  # After running the above commands, you should see:
  $ tree config/bundle
  config/bundle
  ├── bases
  │   └── memcached-operator.clusterserviceversion.yaml
  ├── kustomization.yaml
  ├── manifests
  │   ├── cache.my.domain_memcacheds.yaml
  │   └── memcached-operator.clusterserviceversion.yaml
  └── metadata
      └── annotations.yaml

  # Then it validates your bundle files and builds your bundle image:
  $ operator-sdk bundle validate config/bundle
  $ docker build -f bundle.Dockerfile -t $BUNDLE_IMG .
  Sending build context to Docker daemon  42.33MB
  Step 1/9 : FROM scratch
  ...

  # You can then push your bundle image:
  $ make docker-push IMG=$BUNDLE_IMG
`
)

// setCommonDefaults sets defaults useful to all modes of this subcommand.
func (c *bundleCmd) setCommonDefaults(cfg *config.Config) {
	if c.operatorName == "" {
		c.operatorName = filepath.Base(cfg.Repo)
	}
	// A default channel can be inferred if there is only one channel. Don't infer
	// default otherwise; the user must set this value.
	if c.defaultChannel == "" && strings.Count(c.channels, ",") == 0 {
		c.defaultChannel = c.channels
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

// validateMetadata validates c for bundle metadata generation.
func (c bundleCmd) validateMetadata(*config.Config) (err error) {
	// Ensure a default channel is present.
	if c.defaultChannel == "" {
		return fmt.Errorf("--default-channel must be set if setting multiple channels")
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
