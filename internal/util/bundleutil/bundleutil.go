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

package bundleutil

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/operator-framework/operator-sdk/internal/flags"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"
)

var (
	defaultMetadataDir          = "metadata"
	defaultManifestDir          = "manifests"
	defaultBundleDockerfilePath = "bundle.Dockerfile"
)

// BundleMetaData contains the required metadata to build bundles and images.
type BundleMetaData struct {
	// BundleDir refers to the directory where generated bundles are to be written.
	BundleDir string

	// The PackageName of the operator bundle.
	PackageName string

	// Channels and DefaultChannel the operator should be subscribed to.
	Channels       string
	DefaultChannel string

	// BaseImage name to build bundle image.
	BaseImage string

	// BuildCommand to run while building image.
	BuildCommand string

	// PackageManifestPath where the input manifests are present.
	PkgmanifestPath string

	// IsScoreConfigPresent when set to true includes scorecard config annotations
	// in bundle metadata.
	IsScoreConfigPresent bool

	// Other labels to be added in CSV.
	OtherLabels map[string]string
}

// values to populate bundle metadata/Dockerfile.
type annotationsValues struct {
	BundleDir                string
	PackageName              string
	Channels                 string
	DefaultChannel           string
	OtherLabels              []string
	IsScorecardConfigPresent bool
}

// GenerateMetadata scaffolds annotations.yaml and bundle.Dockerfile with the provided
// annotation values.
func (meta *BundleMetaData) GenerateMetadata() error {
	// Create output directory
	if err := os.MkdirAll(meta.BundleDir, projutil.DirMode); err != nil {
		return err
	}

	// Create annotation values for both bundle.Dockerfile and annotations.yaml, which should
	// hold the same set of values always.
	values := annotationsValues{
		BundleDir:                meta.BundleDir,
		PackageName:              meta.PackageName,
		Channels:                 meta.Channels,
		DefaultChannel:           meta.DefaultChannel,
		IsScorecardConfigPresent: meta.IsScoreConfigPresent,
	}

	for k, v := range meta.OtherLabels {
		values.OtherLabels = append(values.OtherLabels, fmt.Sprintf("%s=%s", k, v))
	}
	sort.Strings(values.OtherLabels)

	// Write each file
	metadataDir := filepath.Join(meta.BundleDir, defaultMetadataDir)
	if err := os.MkdirAll(metadataDir, projutil.DirMode); err != nil {
		return err
	}

	dockerfilePath := defaultBundleDockerfilePath
	// If migrating from packagemanifests to bundle, bundle.Dockerfile is present
	// inside bundleDir, else it's in the project directory. Hence, dockerfile
	// should have the path specified with respect to output directory of resulting bundles.
	// Remove this, when pkgman-to-bundle migrate command is removed.
	if len(meta.PkgmanifestPath) != 0 {
		dockerfilePath = filepath.Join(filepath.Dir(meta.BundleDir), "bundle.Dockerfile")
		values.BundleDir = filepath.Base(meta.BundleDir)
	}

	templateMap := map[string]*template.Template{
		dockerfilePath: dockerfileTemplate,
		filepath.Join(metadataDir, "annotations.yaml"): annotationsTemplate,
	}

	for path, tmpl := range templateMap {
		log.Info(fmt.Sprintf("Creating %s", path))
		f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
		if err != nil {
			return err
		}

		defer func() {
			if err := f.Close(); err != nil {
				log.Error(err)
			}
		}()
		if err = tmpl.Execute(f, values); err != nil {
			return err
		}
	}
	log.Infof("Bundle metadata generated successfully")
	return nil
}

// CopyOperatorManifests copies packagemanifestsDir/manifests to bundleDir/manifests.
func (meta *BundleMetaData) CopyOperatorManifests() error {
	return copyOperatorManifests(meta.PkgmanifestPath, filepath.Join(meta.BundleDir, defaultManifestDir))
}

func copyOperatorManifests(src, dest string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("error reading source directory %v", err)
	}

	if err := os.MkdirAll(dest, srcInfo.Mode()); err != nil {
		return err
	}

	srcFiles, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, f := range srcFiles {
		srcPath := filepath.Join(src, f.Name())
		destPath := filepath.Join(dest, f.Name())

		if f.IsDir() {
			// TODO(verify): we may have to log an error here instead of recursively copying
			// if there are no sub-folders allowed under manifests dir of a packagemanifest.
			if err = copyOperatorManifests(srcPath, destPath); err != nil {
				return err
			}
		} else {
			srcFile, err := os.Open(srcPath)
			if err != nil {
				return err
			}
			defer srcFile.Close()

			destFile, err := os.Create(destPath)
			if err != nil {
				return err
			}
			defer destFile.Close()

			_, err = io.Copy(destFile, srcFile)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// BuildBundleImage builds the bundle image with the provided command or using
// docker build command.
func (meta *BundleMetaData) BuildBundleImage(tag string) error {

	img := fmt.Sprintf("%s:%s", meta.BaseImage, tag)

	// switch back to current working directory, so that subsequent
	// bundle version images can be built.
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	defer func() {
		if err := os.Chdir(cwd); err != nil {
			log.Error(cwd)
		}
	}()

	if err := os.Chdir(filepath.Dir(meta.BundleDir)); err != nil {
		return err
	}

	if len(meta.BuildCommand) != 0 {
		// TODO(varsha): Make this more user friendly by accepting a template which
		// can executed in each bundle subdirectory.
		log.Infof("Using the specified command to build image")
		commandArg := strings.Split(meta.BuildCommand, " ")

		// append the tag and build context to the command
		cmd := exec.Command(commandArg[0], append(commandArg[1:], img)...)
		output, err := cmd.CombinedOutput()
		if err != nil || viper.GetBool(flags.VerboseOpt) {
			fmt.Println(string(output))
		}
		if err != nil {
			return err
		}
	} else {
		output, err := exec.Command("docker", "build", "-f", "bundle.Dockerfile", "-t", img, ".").CombinedOutput()
		if err != nil || viper.GetBool(flags.VerboseOpt) {
			fmt.Println(string(output))
		}
		if err != nil {
			return err
		}
	}
	log.Infof("Successfully built image %s", img)
	return nil
}

// WriteScorecardConfig creates the scorecard directory in the bundle and copies the
// configuration yaml to bundle.
func (meta *BundleMetaData) WriteScorecardConfig(inputConfigPath string) error {
	// If the config is already copied as a part of the manifest directory
	// then ensure that it is deleted to remove duplicates.
	_, filename := filepath.Split(inputConfigPath)
	if err := deleteExistingScorecardConfig(meta.BundleDir, filename); err != nil {
		return err
	}

	scorecardDir := filepath.Join(meta.BundleDir, "tests", "scorecard")

	// Create directory for scorecard config
	if err := os.MkdirAll(scorecardDir, projutil.DirMode); err != nil {
		return err
	}

	log.Info(fmt.Sprintf("Writing scorecard config in %s", scorecardDir))
	b, err := os.ReadFile(inputConfigPath)
	if err != nil {
		return err
	}

	err = os.WriteFile(filepath.Join(scorecardDir, "config.yaml"), b, 0644)
	if err != nil {
		return fmt.Errorf("error writing scorecard config %v", err)
	}
	return nil
}

// deleteExistingScorecardConfig checks if there is an existing scorecard config file
// in manifests/ folder, if present it deletes it.
func deleteExistingScorecardConfig(bundleDir, filename string) error {
	scorecardConfigPath := filepath.Join(bundleDir, defaultManifestDir, filename)
	return os.RemoveAll(scorecardConfigPath)
}
