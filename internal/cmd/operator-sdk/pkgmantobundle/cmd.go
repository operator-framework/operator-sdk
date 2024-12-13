// Copyright 2021 The Operator-SDK Authors
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

package pkgmantobundle

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/operator-framework/api/pkg/apis/scorecard/v1alpha3"
	apimanifests "github.com/operator-framework/api/pkg/manifests"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/operator-framework/operator-sdk/internal/annotations/metrics"
	"github.com/operator-framework/operator-sdk/internal/util/bundleutil"
	"github.com/operator-framework/operator-sdk/internal/util/k8sutil"
)

const (
	longHelp = `
'pkgman-to-bundle' command helps in migrating OLM packagemanifests to bundles which is the preferred OLM packaging format.
This command takes an input packagemanifest directory and generates bundles for each of the versions of manifests present in
the input directory. Additionally, it also provides the flexibility to build bundle images for each of the generated bundles.

The generated bundles are always written on disk. Location for the generated bundles can be specified using '--output-dir'. If not
specified, the default location would be 'bundle/' directory.

The base container image name for the bundles can be provided using '--image-tag-base' flag. This should be provided without the tag, since the tag
for the images would be the bundle version, (ie) image names will be in the format <base_image>:<bundle_version>.

Specify the build command for building container images using '--build-cmd' flag. The default build command is 'docker build'. The command will
need to be in the 'PATH' or fully qualified path name should be provided.
`

	examples = `

# Provide the packagemanifests directory as input to the command. Consider the packagemanifests directory to have the following
# structure:

$ tree packagemanifests/
packagemanifests
└── etcd
    ├── 0.0.1
    │   ├── etcdcluster.crd.yaml
    │   └── etcdoperator.clusterserviceversion.yaml
    ├── 0.0.2
    │   ├── etcdbackup.crd.yaml
    │   ├── etcdcluster.crd.yaml
    │   ├── etcdoperator.v0.0.2.clusterserviceversion.yaml
    │   └── etcdrestore.crd.yaml
    └── etcd.package.yaml

# Run the following command to generate bundles in the default 'bundle/' directory with the base-container image name
# to be 'quay.io/example/etcd'
$ operator-sdk pkgman-to-bundle packagemanifests --image-tag-base quay.io/example/etcd
INFO[0000] Packagemanifests will be migrated to bundles in bundle directory
INFO[0000] Creating bundle/bundle-0.0.1/bundle.Dockerfile
INFO[0000] Creating bundle/bundle-0.0.1/metadata/annotations.yaml
...

# After running the above command, the bundles will be generated in 'bundles/' directory.
$ tree bundles/
bundles/
├── bundle-0.0.1
│   ├── bundle
│   │   ├── manifests
│   │   │   ├── etcdcluster.crd.yaml
│   │   │   ├── etcdoperator.clusterserviceversion.yaml
│   │   ├── metadata
│   │   │   └── annotations.yaml
│   │   └── tests
│   │       └── scorecard
│   │           └── config.yaml
│   └── bundle.Dockerfile
└── bundle-0.0.2
    ├── bundle
    │   ├── manifests
    │   │   ├── etcdbackup.crd.yaml
    │   │   ├── etcdcluster.crd.yaml
    │   │   ├── etcdoperator.v0.0.2.clusterserviceversion.yaml
    │   │   ├── etcdrestore.crd.yaml
    │   └── metadata
    │       └── annotations.yaml
    └── bundle.Dockerfile

A custom command to build bundle images can also be specified using the '--build-cmd' flag. For example,

$ operator-sdk pkgman-to-bundle packagemanifests --image-tag-base quay.io/example/etcd --build-cmd "podman build -f bundle.Dockerfile . -t"

Images for the both the bundles will be built with the following names: quay.io/example/etcd:0.0.1 and quay.io/example/etcd:0.0.2.
`
)

var defaultSubBundleDir = "bundle"

type pkgManToBundleCmd struct {
	// Input packagemanifest directory.
	pkgmanifestDir string

	// Optional flags for generating and building bundles.
	outputDir string
	baseImg   string
	buildCmd  string
}

// NewCmd returns the pkgManToBundleCmd configured with the provided input options.
func NewCmd() *cobra.Command {
	p := pkgManToBundleCmd{}

	pkgManToBundleCmd := &cobra.Command{
		Use:     "pkgman-to-bundle <packagemanifestdir>",
		Short:   "Migrates packagemanifests to bundles",
		Long:    longHelp,
		Example: examples,
		PreRunE: func(_ *cobra.Command, args []string) (err error) {
			return p.validate(args)
		},
		RunE: func(_ *cobra.Command, args []string) (err error) {
			p.pkgmanifestDir = args[0]
			return p.run()
		},
	}

	pkgManToBundleCmd.Flags().StringVar(&p.outputDir, "output-dir", "bundles", "Directory to write bundle to.")
	pkgManToBundleCmd.Flags().StringVar(&p.baseImg, "image-tag-base", "", "Base container image name for bundle image tags, "+
		"ex. my.reg/foo/bar-operator-bundle will become my.reg/foo/bar-operator-bundle:${package-dir-name} for each child directory name in the packagemanifests directory")

	// TODO(varsha): enable users to provide a template to the command so that it can be run in all child directories to build image.
	pkgManToBundleCmd.Flags().StringVar(&p.buildCmd, "build-cmd", "", "Build command to be run for building images. By default 'docker build' is run.")

	return pkgManToBundleCmd
}

// Generate the bundles from the provided packagemanifest directory.
func (p *pkgManToBundleCmd) run() (err error) {

	// error if output bundle directory already exists.
	if _, err = os.Stat(p.outputDir); !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("output directory: %s for bundles already exists", p.outputDir)
	}

	log.Infof("Packagemanifests will be migrated to bundles in %s directory", p.outputDir)

	// Skipping bundles here, since that's not required and could be empty for a manifest directory.
	packages, _, err := apimanifests.GetManifestsDir(p.pkgmanifestDir)
	if err != nil {
		return err
	}

	if packages.IsEmpty() {
		return fmt.Errorf("no packages found in the directory %s", p.pkgmanifestDir)
	}

	// get package metadata required for annotations.yaml and bundle.Dockerfile.
	packageName, defaultChannel, channelsByCSV, err := getPackageMetadata(packages)
	if err != nil {
		return fmt.Errorf("error obtaining metadata from directory %s: %v", p.pkgmanifestDir, err)
	}

	directories, err := os.ReadDir(p.pkgmanifestDir)
	if err != nil {
		return err
	}

	// iterate through each of the subdirectories to generate respective bundles for each of them.
	for _, dir := range directories {
		if dir.IsDir() {
			// this is required to extract project layout and SDK version information.
			otherLabels, channels, err := getSDKStampsAndChannels(filepath.Join(p.pkgmanifestDir, dir.Name()), defaultChannel, channelsByCSV)
			if err != nil {
				return fmt.Errorf("error getting CSV from provided packagemanifest %v", err)
			}

			bundleMetaData := bundleutil.BundleMetaData{
				BundleDir:       filepath.Join(p.outputDir, "bundle-"+dir.Name(), defaultSubBundleDir),
				PackageName:     packageName,
				Channels:        channels,
				DefaultChannel:  defaultChannel,
				PkgmanifestPath: filepath.Join(p.pkgmanifestDir, dir.Name()),
				OtherLabels:     otherLabels,
				BaseImage:       p.baseImg,
				BuildCommand:    p.buildCmd,
			}

			if err := bundleMetaData.CopyOperatorManifests(); err != nil {
				return err
			}

			// get the location of scorecard config file from the current packagemanifest directory.
			scorecardConfigPath, err := getScorecardConfigPath(bundleMetaData.PkgmanifestPath)
			if err != nil {
				return err
			}

			// if scorecard config is present, then copy it to tests/scorecard directory
			// in bundle.
			if scorecardConfigPath != "" {
				bundleMetaData.IsScoreConfigPresent = true
				if err := bundleMetaData.WriteScorecardConfig(scorecardConfigPath); err != nil {
					return err
				}
			}

			if err := bundleMetaData.GenerateMetadata(); err != nil {
				return err
			}

			// build image when base image name is provided.
			if p.baseImg != "" {
				if err := bundleMetaData.BuildBundleImage(dir.Name()); err != nil {
					return err
				}
			}

		}
	}
	return nil
}

// getScorecardConfigPath looks for the path of scorecard config file in the directory.
func getScorecardConfigPath(inputDir string) (string, error) {
	var scorecardConfigPath string

	err := filepath.Walk(inputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			b, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			typeMeta, err := k8sutil.GetTypeMetaFromBytes(b)
			if err != nil {
				return err
			}

			if typeMeta.Kind == v1alpha3.ConfigurationKind {
				if len(scorecardConfigPath) != 0 {
					return fmt.Errorf("multiple scorrecard config files found in packagemanifest directory %s", inputDir)
				}
				scorecardConfigPath = path
			}
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return scorecardConfigPath, nil
}

func getSDKStampsAndChannels(path, defaultChannel string, channelsByCSV map[string][]string) (map[string]string, string, error) {
	bundle, err := apimanifests.GetBundleFromDir(path)
	if err != nil {
		return nil, "", err
	}

	sdkLabels, err := getSDKStamps(bundle)
	if err != nil {
		return nil, "", err
	}

	// Find channels matching the CSV names
	channels := getChannelsByCSV(bundle, channelsByCSV, defaultChannel)

	return sdkLabels, channels, nil
}

// getSDKStamps parses the CSV and gets SDK stamps.
func getSDKStamps(bundle *apimanifests.Bundle) (map[string]string, error) {
	if bundle.CSV == nil {
		return nil, fmt.Errorf("cannot find CSV from manifests package")
	}

	// Extract SDK layout and version from CSV annotations.
	csvAnnotations := bundle.CSV.GetAnnotations()
	sdkLabels := make(map[string]string)

	for key, value := range csvAnnotations {
		if key == metrics.BuilderObjectAnnotation {
			sdkLabels[key] = value
		}

		if key == metrics.LayoutObjectAnnotation {
			sdkLabels[key] = value
		}
	}

	return sdkLabels, nil
}

// getChannelsByCSV creates a list for channels for the currentCSV. For other versions of manifests which
// are not present in the manifest, the defaultChannel is added.
func getChannelsByCSV(bundle *apimanifests.Bundle, channelsByCSV map[string][]string, defaultChannel string) (channels string) {
	// Find channels matching the CSV names
	channelNames := channelsByCSV[bundle.CSV.GetName()]
	channels = strings.Join(channelNames, ",")

	// TODO: verify if we have to add this validation since while building bundles if channel is not specified
	// we add the default channel.
	if channels == "" {
		channels = defaultChannel
		log.Infof("Supported channels cannot be identified from CSV %s, using default channel %s", bundle.CSV.GetName(), defaultChannel)
	}

	return channels
}

func getPackageMetadata(pkg *apimanifests.PackageManifest) (packagename, defaultChannel string, channelsByCSV map[string][]string, err error) {
	packagename = pkg.PackageName
	if packagename == "" {
		err = fmt.Errorf("cannot find packagename from the manifest directory")
		return
	}

	defaultChannel = pkg.DefaultChannelName
	if defaultChannel == "" {
		err = fmt.Errorf("cannot find the default channel for package %q", packagename)
		return
	}

	channelsByCSV = make(map[string][]string)

	for _, p := range pkg.Channels {
		if _, ok := channelsByCSV[p.CurrentCSVName]; !ok {
			channelsByCSV[p.CurrentCSVName] = make([]string, 0)
		}
		channelsByCSV[p.CurrentCSVName] = append(channelsByCSV[p.CurrentCSVName], p.Name)
	}

	return
}

func (p *pkgManToBundleCmd) validate(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("a package manifest directory argument is required")
	}

	if len(p.baseImg) == 0 && len(p.buildCmd) != 0 {
		return fmt.Errorf("base image needs to be specified to build bundle image")
	}
	return nil
}
