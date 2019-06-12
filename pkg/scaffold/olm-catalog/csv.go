// Copyright 2019 The Operator-SDK Authors
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

package catalog

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/operator-framework/operator-sdk/internal/pkg/scaffold"
	"github.com/operator-framework/operator-sdk/internal/pkg/scaffold/input"
	catalog "github.com/operator-framework/operator-sdk/internal/pkg/scaffold/olm-catalog"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const (
	OutputFormatJSON = "json"
	OutputFormatYAML = "yaml"
)

type GenCatalogSourceCmd struct {
	BundleDir           string
	PackageManifestPath string
	OutputFormat        string
	WriteTo             string
	Write               bool
}

func (c *GenCatalogSourceCmd) Run() error {

	if c.OutputFormat == "" {
		c.OutputFormat = OutputFormatYAML
	}

	if err := c.verify(); err != nil {
		return err
	}

	log.Infof("Generating %s CatalogSource manifest", strings.ToUpper(string(c.OutputFormat)))

	absProjectPath := projutil.MustGetwd()
	cfg := &input.Config{
		Repo:           projutil.CheckAndGetProjectGoPkg(),
		AbsProjectPath: absProjectPath,
		ProjectName:    filepath.Base(absProjectPath),
	}
	cs := &catalog.CatalogSource{
		BundleDir:           c.BundleDir,
		PackageManifestPath: c.PackageManifestPath,
	}

	if c.WriteTo != "" || c.Write {
		// Write a CatalogSource manifest to either a specified file or the default
		// in CatalogSource.
		if c.WriteTo != "" {
			cs.Path = c.WriteTo
		}
		err := (&scaffold.Scaffold{}).Execute(cfg, cs)
		if err != nil {
			return errors.Wrap(err, "failed to scaffold catalog source")
		}
	} else {
		// Print the bytes without writing.
		b, err := cs.CustomRender()
		if err != nil {
			return errors.Wrap(err, "failed to render catalog source")
		}
		if _, err := fmt.Fprintln(os.Stdout, string(b)); err != nil {
			return errors.Wrap(err, "failed to print catalog source")
		}
	}
	return nil
}

func (c *GenCatalogSourceCmd) verify() error {
	if c.OutputFormat != OutputFormatJSON && c.OutputFormat != OutputFormatYAML {
		return fmt.Errorf("output format must be one of: %s, %s", OutputFormatJSON, OutputFormatYAML)
	}
	return nil
}
