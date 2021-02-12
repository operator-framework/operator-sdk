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
	"strings"
	"text/template"

	"github.com/operator-framework/api/pkg/apis/scorecard/v1alpha3"
	"github.com/operator-framework/operator-registry/pkg/lib/bundle"
	log "github.com/sirupsen/logrus"
	"sigs.k8s.io/yaml"

	metricsannotations "github.com/operator-framework/operator-sdk/internal/annotations/metrics"
	genutil "github.com/operator-framework/operator-sdk/internal/cmd/operator-sdk/generate/internal"
	gencsv "github.com/operator-framework/operator-sdk/internal/generate/clusterserviceversion"
	"github.com/operator-framework/operator-sdk/internal/generate/clusterserviceversion/bases"
	"github.com/operator-framework/operator-sdk/internal/generate/collector"
	"github.com/operator-framework/operator-sdk/internal/registry"
	"github.com/operator-framework/operator-sdk/internal/scorecard"
)

const (
	longHelp = `
Running 'generate bundle' is the first step to publishing your operator to a catalog and/or deploying it with OLM.
This command generates a set of bundle manifests, metadata, and a bundle.Dockerfile for your operator.
A ClusterServiceVersion manifest will be generated from the set of manifests passed to this command (see below)
using either an existing base at '<kustomize-dir>/bases/<package-name>.clusterserviceversion.yaml',
typically containing metadata added by 'generate kustomize manifests' or by hand, or from scratch if that base
does not exist. All non-metadata values in a base will be overwritten.

There are two ways to pass cluster-ready manifests to this command: stdin via a Unix pipe,
or in a directory using '--input-dir'. See command help for more information on these modes.
Passing a directory is useful for running this command outside of a project, as kustomize
config files are likely not present and/or only cluster-ready manifests are available.

Set '--version' to supply a semantic version for your bundle if you are creating one
for the first time or upgrading an existing one.

If '--output-dir' is set and you wish to build bundle images from that directory,
either manually update your bundle.Dockerfile or set '--overwrite'.

More information on bundles:
https://github.com/operator-framework/operator-registry/#manifest-format
`

	examples = `
  # If running within a project or in a project that uses kustomize to generate manifests,
	# make sure a kustomize directory exists that looks like the following 'config/manifests' directory:
  $ tree config/manifests
  config/manifests
  ├── bases
  │   └── memcached-operator.clusterserviceversion.yaml
  └── kustomization.yaml

  # Generate a 0.0.1 bundle by passing manifests to stdin:
  $ kustomize build config/manifests | operator-sdk generate bundle --version 0.0.1
  Generating bundle version 0.0.1
  ...

  # If running outside of a project or in a project that does not use kustomize to generate manifests,
	# make sure cluster-ready manifests are available on disk:
  $ tree deploy/
  deploy/
  ├── crds
  │   └── cache.my.domain_memcacheds.yaml
  ├── deployment.yaml
  ├── role.yaml
  ├── role_binding.yaml
  ├── service_account.yaml
  └── webhooks.yaml

  # Generate a 0.0.1 bundle by passing manifests by dir:
  $ operator-sdk generate bundle --input-dir deploy --version 0.0.1
  Generating bundle version 0.0.1
  ...

  # After running in either of the above modes, you should see this directory structure:
  $ tree bundle/
  bundle/
  ├── manifests
  │   ├── cache.my.domain_memcacheds.yaml
  │   └── memcached-operator.clusterserviceversion.yaml
  └── metadata
      └── annotations.yaml
`
)

// defaultRootDir is the default root directory in which to generate bundle files.
const defaultRootDir = "bundle"

// setDefaults sets defaults useful to all modes of this subcommand.
func (c *bundleCmd) setDefaults() (err error) {
	if c.packageName, c.layout, err = genutil.GetPackageNameAndLayout(c.packageName); err != nil {
		return err
	}
	return nil
}

// validateManifests validates c for bundle manifests generation.
func (c bundleCmd) validateManifests() (err error) {
	if c.version != "" {
		if err := genutil.ValidateVersion(c.version); err != nil {
			return err
		}
	}

	// The three possible usage modes (stdin, inputDir, and legacy dirs) are mutually exclusive
	// and one must be chosen.
	isPipeReader := genutil.IsPipeReader()
	isInputDir := c.inputDir != ""
	isLegacyDirs := c.deployDir != "" || c.crdsDir != ""
	switch {
	case !(isPipeReader || isInputDir || isLegacyDirs):
		return errors.New("one of stdin, --input-dir, or --deploy-dir (and optionally --crds-dir) must be set")
	case isPipeReader && (isInputDir || isLegacyDirs):
		return errors.New("none of --input-dir, --deploy-dir, or --crds-dir may be set if reading from stdin")
	case isInputDir && isLegacyDirs:
		return errors.New("only one of --input-dir or --deploy-dir (and optionally --crds-dir) may be set if not reading from stdin")
	}

	if c.stdout {
		if c.outputDir != "" {
			return errors.New("--output-dir cannot be set if writing to stdout")
		}
	}

	return nil
}

// runManifests generates bundle manifests.
func (c bundleCmd) runManifests() (err error) {

	c.println("Generating bundle manifests")

	if !c.stdout && c.outputDir == "" {
		c.outputDir = defaultRootDir
	}

	col := &collector.Manifests{}
	switch {
	case genutil.IsPipeReader():
		err = col.UpdateFromReader(os.Stdin)
	case c.deployDir != "" && c.crdsDir != "":
		err = col.UpdateFromDirs(c.deployDir, c.crdsDir)
	case c.deployDir != "": // If only deployDir is set, use as input dir.
		c.inputDir = c.deployDir
		fallthrough
	case c.inputDir != "":
		err = col.UpdateFromDir(c.inputDir)
	}
	if err != nil {
		return err
	}

	// If no CSV was initially read, a kustomize base can be used at the default base path.
	// Only read from kustomizeDir if a base exists so users can still generate a barebones CSV.
	baseCSVPath := filepath.Join(c.kustomizeDir, "bases", c.packageName+".clusterserviceversion.yaml")
	if len(col.ClusterServiceVersions) == 0 && genutil.IsExist(baseCSVPath) {
		base, err := bases.ClusterServiceVersion{BasePath: baseCSVPath}.GetBase()
		if err != nil {
			return fmt.Errorf("error reading CSV base: %v", err)
		}
		col.ClusterServiceVersions = append(col.ClusterServiceVersions, *base)
	} else {
		c.println("Building a ClusterServiceVersion without an existing base")
	}

	var opts []gencsv.Option
	stdout := genutil.NewMultiManifestWriter(os.Stdout)
	if c.stdout {
		opts = append(opts, gencsv.WithWriter(stdout))
	} else {
		opts = append(opts, gencsv.WithBundleWriter(c.outputDir))
	}

	csvGen := gencsv.Generator{
		OperatorName: c.packageName,
		Version:      c.version,
		Collector:    col,
		Annotations:  metricsannotations.MakeBundleObjectAnnotations(c.layout),
	}
	if err := csvGen.Generate(opts...); err != nil {
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

	c.println("Bundle manifests generated successfully in", c.outputDir)

	return nil
}

// writeScorecardConfig writes cfg to dir at the hard-coded config path 'config.yaml'.
func writeScorecardConfig(dir string, cfg v1alpha3.Configuration) error {
	// Skip writing if config is empty.
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

// runMetadata generates a bundle.Dockerfile and bundle metadata.
func (c bundleCmd) runMetadata() error {

	c.println("Generating bundle metadata")

	if c.outputDir == "" {
		c.outputDir = defaultRootDir
	}
	if err := os.MkdirAll(c.outputDir, 0755); err != nil {
		return err
	}

	// If metadata already exists, only overwrite it if directed to.
	bundleRoot := c.inputDir
	if bundleRoot == "" {
		bundleRoot = c.outputDir
	}
	if _, _, err := registry.FindBundleMetadata(bundleRoot); err != nil {
		merr := registry.MetadataNotFoundError("")
		if !errors.As(err, &merr) {
			return err
		}
	} else if !c.overwrite {
		return nil
	}

	// Create annotation values for both bundle.Dockerfile and annotations.yaml, which should
	// hold the same set of values always.
	values := annotationsValues{
		BundleDir:      c.outputDir,
		PackageName:    c.packageName,
		Channels:       c.channels,
		DefaultChannel: c.defaultChannel,
	}
	for key, value := range metricsannotations.MakeBundleMetadataLabels(c.layout) {
		values.OtherLabels = append(values.OtherLabels, fmt.Sprintf("%s=%s", key, value))
	}

	// Write each file.
	metadataDir := filepath.Join(c.outputDir, "metadata")
	if err := os.MkdirAll(metadataDir, 0755); err != nil {
		return err
	}
	templateMap := map[string]*template.Template{
		"bundle.Dockerfile":                            dockerfileTemplate,
		filepath.Join(metadataDir, "annotations.yaml"): annotationsTemplate,
	}
	for path, tmpl := range templateMap {
		c.println("Creating", path)
		f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
		if err != nil {
			return err
		}
		defer func() {
			if err := f.Close(); err != nil {
				log.Error(err)
			}
		}()
		if err := tmpl.Execute(f, values); err != nil {
			return err
		}
	}

	c.println("Bundle metadata generated successfully")

	return nil
}

// values to populate bundle metadata/Dockerfile.
type annotationsValues struct {
	BundleDir      string
	PackageName    string
	Channels       string
	DefaultChannel string
	OtherLabels    []string
}

// Transform a Dockerfile label to a YAML kv.
var funcs = template.FuncMap{
	"toYAML": func(s string) string { return strings.ReplaceAll(s, "=", ": ") },
}

// Template for bundle.Dockerfile, containing scorecard labels.
var dockerfileTemplate = template.Must(template.New("").Funcs(funcs).Parse(`FROM scratch

# Core bundle labels.
LABEL operators.operatorframework.io.bundle.mediatype.v1=registry+v1
LABEL operators.operatorframework.io.bundle.manifests.v1=manifests/
LABEL operators.operatorframework.io.bundle.metadata.v1=metadata/
LABEL operators.operatorframework.io.bundle.package.v1={{ .PackageName }}
LABEL operators.operatorframework.io.bundle.channels.v1={{ .Channels }}
{{- if .DefaultChannel }}
LABEL operators.operatorframework.io.bundle.channel.default.v1={{ .DefaultChannel }}
{{- end }}
{{- range $i, $l := .OtherLabels }}
LABEL {{ $l }}
{{- end }}

# Labels for testing.
LABEL operators.operatorframework.io.test.mediatype.v1=scorecard+v1
LABEL operators.operatorframework.io.test.config.v1=tests/scorecard/

# Copy files to locations specified by labels.
COPY {{ .BundleDir }}/manifests /manifests/
COPY {{ .BundleDir }}/metadata /metadata/
COPY {{ .BundleDir }}/tests/scorecard /tests/scorecard/
`))

// Template for annotations.yaml, containing scorecard labels.
var annotationsTemplate = template.Must(template.New("").Funcs(funcs).Parse(`annotations:
  # Core bundle annotations.
  operators.operatorframework.io.bundle.mediatype.v1: registry+v1
  operators.operatorframework.io.bundle.manifests.v1: manifests/
  operators.operatorframework.io.bundle.metadata.v1: metadata/
  operators.operatorframework.io.bundle.package.v1: {{ .PackageName }}
  operators.operatorframework.io.bundle.channels.v1: {{ .Channels }}
  {{- if .DefaultChannel }}
  operators.operatorframework.io.bundle.channel.default.v1: {{ .DefaultChannel }}
  {{- end }}
  {{- range $i, $l := .OtherLabels }}
  {{ toYAML $l }}
  {{- end }}

  # Annotations for testing.
  operators.operatorframework.io.test.mediatype.v1: scorecard+v1
  operators.operatorframework.io.test.config.v1: tests/scorecard/
`))
