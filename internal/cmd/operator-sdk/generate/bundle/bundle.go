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
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/operator-framework/api/pkg/apis/scorecard/v1alpha3"
	"github.com/operator-framework/operator-registry/pkg/lib/bundle"
	"sigs.k8s.io/kubebuilder/pkg/model/config"
	"sigs.k8s.io/yaml"

	metricsannotations "github.com/operator-framework/operator-sdk/internal/annotations/metrics"
	scorecardannotations "github.com/operator-framework/operator-sdk/internal/annotations/scorecard"
	genutil "github.com/operator-framework/operator-sdk/internal/cmd/operator-sdk/generate/internal"
	gencsv "github.com/operator-framework/operator-sdk/internal/generate/clusterserviceversion"
	"github.com/operator-framework/operator-sdk/internal/generate/collector"
	"github.com/operator-framework/operator-sdk/internal/registry"
	"github.com/operator-framework/operator-sdk/internal/scorecard"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"
)

const (
	longHelp = `
Running 'generate bundle' is the first step to publishing your operator to a catalog and/or deploying it with OLM.
This command generates a set of bundle manifests, metadata, and a bundle.Dockerfile for your operator.
Typically one would run 'generate kustomize manifests' first to (re)generate kustomize bases consumed by this command.

Set '--version' to supply a semantic version for your bundle if you are creating one
for the first time or upgrading an existing one.

If '--output-dir' is set and you wish to build bundle images from that directory,
either manually update your bundle.Dockerfile or set '--overwrite'.

More information on bundles:
https://github.com/operator-framework/operator-registry/#manifest-format
`

	examples = `
  # Generate bundle files and build your bundle image with these 'make' recipes:
  $ make bundle
  $ export USERNAME=<your registry username>
  $ export BUNDLE_IMG=quay.io/$USERNAME/memcached-operator-bundle:v0.0.1
  $ make bundle-build BUNDLE_IMG=$BUNDLE_IMG

  # The above recipe runs the following commands manually. First it creates bundle
  # manifests, metadata, and a bundle.Dockerfile:
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
  $ kustomize build config/manifests | operator-sdk generate bundle --overwrite --version 0.0.1
  Generating bundle manifest version 0.0.1
  ...

  # After running the above commands, you should see this directory structure:
  $ tree bundle
  bundle
  ├── manifests
  │   ├── cache.my.domain_memcacheds.yaml
  │   └── memcached-operator.clusterserviceversion.yaml
  └── metadata
      └── annotations.yaml

  # Then it validates your bundle files and builds your bundle image:
  $ operator-sdk bundle validate ./bundle
  $ docker build -f bundle.Dockerfile -t $BUNDLE_IMG .
  Sending build context to Docker daemon  42.33MB
  Step 1/9 : FROM scratch
  ...

  # You can then push your bundle image:
  $ make docker-push IMG=$BUNDLE_IMG
`
)

// defaultRootDir is the default root directory in which to generate bundle files.
const defaultRootDir = "bundle"

// setDefaults sets defaults useful to all modes of this subcommand.
func (c *bundleCmd) setDefaults(cfg *config.Config) (err error) {
	if c.projectName, err = genutil.GetOperatorName(cfg); err != nil {
		return err
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

	if c.inputDir == "" {
		c.inputDir = defaultRootDir
	}
	if !c.stdout {
		if c.outputDir == "" {
			c.outputDir = defaultRootDir
		}
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
		OperatorName: c.projectName,
		OperatorType: projutil.PluginKeyToOperatorType(cfg.Layout),
		Version:      c.version,
		Collector:    col,
	}

	stdout := genutil.NewMultiManifestWriter(os.Stdout)
	opts := []gencsv.Option{
		// By not passing apisDir and turning interactive prompts on, we forcibly rely on the kustomize base
		// for UI metadata and uninferrable data.
		gencsv.WithBase(c.kustomizeDir, "", projutil.InteractiveHardOff),
	}
	if c.stdout {
		opts = append(opts, gencsv.WithWriter(stdout))
	} else {
		opts = append(opts, gencsv.WithBundleWriter(c.outputDir))
	}

	if err := csvGen.Generate(cfg, opts...); err != nil {
		return fmt.Errorf("error generating ClusterServiceVersion: %v", err)
	}

	objs := genutil.GetManifestObjects(col)
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

	// Write the scorecard config if it was passed.
	if err := writeScorecardConfig(c.outputDir, col.ScorecardConfig); err != nil {
		return fmt.Errorf("error writing bundle scorecard config: %v", err)
	}

	if !c.quiet && !c.stdout {
		fmt.Println("Bundle manifests generated successfully in", c.outputDir)
	}

	return nil
}

// writeScorecardConfig writes cfg to dir at the hard-coded config path 'config.yaml'.
func writeScorecardConfig(dir string, cfg v1alpha3.Configuration) error {
	if cfg.Metadata.Name == "" {
		return nil
	}

	b, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	cfgDir := filepath.Join(dir, filepath.FromSlash(scorecard.DefaultConfigDir))
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		return err
	}
	scorecardConfigPath := filepath.Join(cfgDir, scorecard.ConfigFileName)
	return ioutil.WriteFile(scorecardConfigPath, b, 0666)
}

// validateMetadata validates c for bundle metadata generation.
func (c bundleCmd) validateMetadata(*config.Config) (err error) {
	return nil
}

// runMetadata generates a bundle.Dockerfile and bundle metadata.
func (c bundleCmd) runMetadata(cfg *config.Config) error {

	directory := c.inputDir
	if directory == "" {
		// There may be no existing bundle at the default path, so assume manifests
		// only exist in the output directory.
		defaultDirectory := filepath.Join(defaultRootDir, bundle.ManifestsDir)
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

	return c.generateMetadata(cfg, directory, outputDir)
}

// generateMetadata wraps the operator-registry bundle Dockerfile/metadata generator.
func (c bundleCmd) generateMetadata(cfg *config.Config, manifestsDir, outputDir string) error {

	metadataExists := isMetatdataExist(outputDir, manifestsDir)
	err := bundle.GenerateFunc(manifestsDir, outputDir, c.projectName, c.channels, c.defaultChannel, c.overwrite)
	if err != nil {
		return fmt.Errorf("error generating bundle metadata: %v", err)
	}

	// Add SDK annotations/labels if metadata did not exist before or when overwrite is true.
	if c.overwrite || !metadataExists {
		bundleRoot := outputDir
		if bundleRoot == "" {
			bundleRoot = filepath.Dir(manifestsDir)
		}

		if err = updateMetadata(cfg, bundleRoot); err != nil {
			return err
		}
	}
	return nil
}

// TODO(estroz): these updates need to be atomic because the bundle's Dockerfile and annotations.yaml
// cannot be out-of-sync.
func updateMetadata(cfg *config.Config, bundleRoot string) error {
	bundleLabels := metricsannotations.MakeBundleMetadataLabels(cfg)
	for key, value := range scorecardannotations.MakeBundleMetadataLabels(scorecard.DefaultConfigDir) {
		if _, hasKey := bundleLabels[key]; hasKey {
			return fmt.Errorf("internal error: duplicate bundle annotation key %s", key)
		}
		bundleLabels[key] = value
	}

	// Write labels to bundle Dockerfile.
	if err := rewriteDockerfileLabels(bundle.DockerFile, bundleLabels); err != nil {
		return fmt.Errorf("error writing LABEL's in %s: %v", bundle.DockerFile, err)
	}
	if err := rewriteAnnotations(bundleRoot, bundleLabels); err != nil {
		return fmt.Errorf("error writing LABEL's in bundle metadata: %v", err)
	}

	// Add a COPY for the scorecard config to bundle Dockerfile.
	// TODO: change input config path to be a flag-based value.
	localScorecardConfigPath := filepath.Join(bundleRoot, filepath.FromSlash(scorecard.DefaultConfigDir))
	err := writeDockerfileCOPYScorecardConfig(bundle.DockerFile, localScorecardConfigPath)
	if err != nil {
		return fmt.Errorf("error writing scorecard config COPY in %s: %v", bundle.DockerFile, err)
	}

	return nil
}

// writeDockerfileCOPYScorecardConfig checks if bundle.Dockerfile and scorecard config exists in
// the operator project. If it does, it injects the scorecard configuration into bundle image.
func writeDockerfileCOPYScorecardConfig(dockerfileName, localConfigDir string) error {
	if isExist(bundle.DockerFile) && isExist(localConfigDir) {
		scorecardFileContent := fmt.Sprintf("COPY %s %s\n", localConfigDir, "/"+scorecard.DefaultConfigDir)
		return projutil.RewriteFileContents(dockerfileName, "COPY", scorecardFileContent)
	}
	return nil
}

// isMetatdataExist returns true if bundle.Dockerfile and metadataDir exist, if not
// it returns false.
func isMetatdataExist(outputDir, manifestsDir string) bool {
	var annotationsDir string
	if outputDir == "" {
		annotationsDir = filepath.Dir(manifestsDir) + bundle.MetadataDir
	} else {
		annotationsDir = outputDir + bundle.MetadataDir
	}

	if genutil.IsNotExist(bundle.DockerFile) || genutil.IsNotExist(annotationsDir) {
		return false
	}
	return true
}

func rewriteDockerfileLabels(dockerfileName string, kvs map[string]string) error {
	var labelStrings []string
	for key, value := range kvs {
		labelStrings = append(labelStrings, fmt.Sprintf("LABEL %s=%s\n", key, value))
	}
	sort.Strings(labelStrings)
	var newBundleLabels strings.Builder
	for _, line := range labelStrings {
		newBundleLabels.WriteString(line)
	}

	return projutil.RewriteFileContents(dockerfileName, "LABEL", newBundleLabels.String())
}

func rewriteAnnotations(bundleRoot string, kvs map[string]string) error {
	annotations, annotationsPath, err := registry.FindBundleMetadata(bundleRoot)
	if err != nil {
		return err
	}

	for key, value := range kvs {
		annotations[key] = value
	}
	annotationsFile := bundle.AnnotationMetadata{
		Annotations: annotations,
	}
	b, err := yaml.Marshal(annotationsFile)
	if err != nil {
		return err
	}

	mode := os.FileMode(0666)
	if info, err := os.Stat(annotationsPath); err == nil {
		mode = info.Mode()
	}
	return ioutil.WriteFile(annotationsPath, b, mode)
}

// isExist returns true if path exists.
func isExist(path string) bool {
	_, err := os.Stat(path)
	return err == nil || os.IsExist(err)
}
