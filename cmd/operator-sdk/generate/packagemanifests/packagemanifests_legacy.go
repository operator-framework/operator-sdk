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
	"fmt"
	"path/filepath"

	log "github.com/sirupsen/logrus"

	genutil "github.com/operator-framework/operator-sdk/cmd/operator-sdk/generate/internal"
	gencsv "github.com/operator-framework/operator-sdk/internal/generate/clusterserviceversion"
	"github.com/operator-framework/operator-sdk/internal/generate/collector"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"
)

func (c *packagemanifestsCmd) setDefaultsLegacy() {
	if c.operatorName == "" {
		c.operatorName = filepath.Base(projutil.MustGetwd())
	}
}

func (c packagemanifestsCmd) validateLegacy() error {
	if err := genutil.ValidateVersion(c.version); err != nil {
		return err
	}
	if c.fromVersion != "" {
		if err := genutil.ValidateVersion(c.fromVersion); err != nil {
			return err
		}
		if c.version == c.fromVersion {
			return fmt.Errorf("--from-version (%s) cannot equal --version; set --version instead", c.fromVersion)
		}
	}

	if c.isDefaultChannel && c.channelName == "" {
		return fmt.Errorf("--default-channel can only be set if --channel is set")
	}

	return nil
}

func (c packagemanifestsCmd) runLegacy() error {
	log.Infof("Generating package version %s", c.version)

	if c.manifestRoot == "" {
		c.manifestRoot = "deploy"
	}
	if c.crdsDir == "" {
		c.crdsDir = filepath.Join(c.manifestRoot, "crds")
	}
	if c.apisDir == "" {
		c.apisDir = filepath.Join("pkg", "apis")
	}
	defaultDir := filepath.Join(c.manifestRoot, "olm-catalog", c.operatorName)
	if c.inputDir == "" {
		c.inputDir = defaultDir
	}
	if c.outputDir == "" {
		c.outputDir = defaultDir
	}

	if err := c.generatePackageManifest(); err != nil {
		return err
	}

	col := &collector.Manifests{}
	if err := col.UpdateFromDirs(c.manifestRoot, c.crdsDir); err != nil {
		return err
	}

	csvGen := gencsv.Generator{
		OperatorName: c.operatorName,
		OperatorType: projutil.GetOperatorType(),
		Version:      c.version,
		FromVersion:  c.fromVersion,
		Collector:    col,
	}

	opts := []gencsv.LegacyOption{
		gencsv.WithPackageBase(c.inputDir, c.apisDir),
		gencsv.WithPackageWriterLegacy(c.outputDir),
	}
	if err := csvGen.GenerateLegacy(opts...); err != nil {
		return err
	}

	if c.updateCRDs {
		dir := filepath.Join(c.outputDir, c.version)
		if err := genutil.WriteCRDFiles(dir, col.CustomResourceDefinitions...); err != nil {
			return err
		}
	}

	log.Info("Package generated successfully")

	return nil
}
