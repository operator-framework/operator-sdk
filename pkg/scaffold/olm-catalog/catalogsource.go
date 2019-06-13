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

	"github.com/operator-framework/operator-sdk/internal/util/projutil"
	"github.com/operator-framework/operator-sdk/pkg/scaffold/olm-catalog/internal"

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

	cs := &internal.CatalogSource{
		ProjectName:         filepath.Base(projutil.MustGetwd()),
		BundleDir:           c.BundleDir,
		PackageManifestPath: c.PackageManifestPath,
	}
	b, err := cs.Bytes()
	if err != nil {
		return errors.Wrap(err, "failed to get catalog source bytes")
	}

	// TODO(estroz): respect c.OutputFormat.
	if _, err := fmt.Fprintln(os.Stdout, string(b)); err != nil {
		return errors.Wrap(err, "failed to print catalog source")
	}
	return nil
}

func (c *GenCatalogSourceCmd) verify() error {
	if c.OutputFormat != OutputFormatJSON && c.OutputFormat != OutputFormatYAML {
		return fmt.Errorf("output format must be one of: %s, %s", OutputFormatJSON, OutputFormatYAML)
	}
	return nil
}
