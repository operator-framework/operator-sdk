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
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/operator-framework/operator-sdk/internal/util/k8sutil"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"
	"github.com/operator-framework/operator-sdk/internal/util/yamlutil"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const (
	OutputFormatJSON = "json"
	OutputFormatYAML = "yaml"
)

// GenCatalogSourceCmd holds arguments used to configure CatalogSource
// generation.
type GenCatalogSourceCmd struct {
	// Namespace is the cluster namespace set in the CatalogSource metadata.
	Namespace string
	// BundleDir contains CRD's, CSV's, and optionally a package manifest.
	BundleDir string
	// PackageManifestPath is the path to the package manifest describing
	// the operator's CSV's.
	// This field is required if no package manifest exists in BundleDir.
	PackageManifestPath string
	// CatalogSourcePath is the path to a CatalogSource manifest to include
	// in output data.
	// This field is optional. Run() will create a CatalogSource using provided
	// data in BundleDir.
	CatalogSourcePath string
	// OutputFormat controls what format data is output in.
	// Options are: "json", "yaml".
	OutputFormat string
	// Write bytes to Writer.
	Writer io.Writer
}

// Run generates a CatalogSource from data in c. Run will write the resulting
// bytes in the format specified by c.OutputFormat to c.Writer, which default
// to "yaml" and stdout.
func (c *GenCatalogSourceCmd) Run() error {
	if c.Writer == nil {
		c.Writer = os.Stdout
	}
	if c.OutputFormat == "" {
		c.OutputFormat = OutputFormatYAML
	}

	if err := c.verify(); err != nil {
		return err
	}

	log.Infof("Generating %s CatalogSource and ConfigMap manifest", strings.ToUpper(c.OutputFormat))

	cs := &CatalogSourceBundle{
		ProjectName:         filepath.Base(projutil.MustGetwd()),
		Namespace:           c.Namespace,
		BundleDir:           c.BundleDir,
		PackageManifestPath: c.PackageManifestPath,
		CatalogSourcePath:   c.CatalogSourcePath,
	}
	configMap, catsrc, err := cs.ToConfigMapAndCatalogSource()
	if err != nil {
		return errors.Wrap(err, "failed to get CatalogSource and ConfigMap")
	}

	var m k8sutil.MarshalFunc = yaml.Marshal
	if c.OutputFormat == OutputFormatJSON {
		m = json.Marshal
	}
	cmb, err := k8sutil.GetObjectBytes(configMap, m)
	if err != nil {
		return errors.Wrap(err, "failed to get ConfigMap bytes")
	}
	csb, err := k8sutil.GetObjectBytes(catsrc, m)
	if err != nil {
		return errors.Wrap(err, "failed to get CatalogSource bytes")
	}
	b := yamlutil.CombineManifests(csb, cmb)
	if _, err := fmt.Fprintln(c.Writer, string(b)); err != nil {
		return errors.Wrap(err, "failed to write CatalogSource and ConfigMap")
	}
	return nil
}

func (c *GenCatalogSourceCmd) verify() error {
	if c.OutputFormat != OutputFormatJSON && c.OutputFormat != OutputFormatYAML {
		return errors.Errorf("output format must be one of: %s, %s", OutputFormatJSON, OutputFormatYAML)
	}
	if c.BundleDir == "" {
		return errors.Errorf("bundle dir must be set")
	}
	return nil
}
