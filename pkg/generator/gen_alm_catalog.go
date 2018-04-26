// Copyright 2018 The Operator-SDK Authors
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

package generator

import (
	"fmt"
	"io"
	"strings"
	"text/template"
)

const (
	// Sample catalog resource values
	// TODO: Make this configurable
	packageChannel = "alpha"
)

// CatalogPackageConfig contains the data needed to generate deploy/alm-catalog/package.yaml
type CatalogPackageConfig struct {
	PackageName string
	ChannelName string
	CurrentCSV  string
}

// renderCatalogPackage generates deploy/alm-catalog/package.yaml
func renderCatalogPackage(w io.Writer, config *Config, catalogVersion string) error {
	t := template.New(catalogPackageYaml)
	t, err := t.Parse(catalogPackageTmpl)
	if err != nil {
		return fmt.Errorf("failed to parse catalog package template: %v", err)
	}

	name := strings.ToLower(config.Kind)
	cpConfig := CatalogPackageConfig{
		PackageName: name,
		ChannelName: packageChannel,
		CurrentCSV:  getCSVName(name, catalogVersion),
	}
	return t.Execute(w, cpConfig)
}

// CRDConfig contains the data needed to generate deploy/alm-catalog/crd.yaml
type CRDConfig struct {
	Kind         string
	KindSingular string
	KindPlural   string
	GroupName    string
	Version      string
}

// renderCRD generates deploy/alm-catalog/crd.yaml
func renderCRD(w io.Writer, config *Config) error {
	t := template.New(catalogCRDYaml)
	t, err := t.Parse(crdTmpl)
	if err != nil {
		return fmt.Errorf("failed to parse catalog CRD template: %v", err)
	}

	kindSingular := strings.ToLower(config.Kind)
	crdConfig := CRDConfig{
		Kind:         config.Kind,
		KindSingular: kindSingular,
		KindPlural:   kindSingular + "s",
		GroupName:    groupName(config.APIVersion),
		Version:      version(config.APIVersion),
	}
	return t.Execute(w, crdConfig)
}

// CSVConfig contains the data needed to generate deploy/alm-catalog/csv.yaml
type CSVConfig struct {
	Kind           string
	KindSingular   string
	KindPlural     string
	GroupName      string
	CRDVersion     string
	ProjectName    string
	CSVName        string
	Image          string
	CatalogVersion string
}

// renderCatalogCSV generates deploy/alm-catalog/csv.yaml
func renderCatalogCSV(w io.Writer, config *Config, image, catalogVersion string) error {
	t := template.New(catalogCSVYaml)
	t, err := t.Parse(catalogCSVTmpl)
	if err != nil {
		return fmt.Errorf("failed to parse catalog CSV template: %v", err)
	}

	kindSingular := strings.ToLower(config.Kind)
	csvConfig := CSVConfig{
		Kind:           config.Kind,
		KindSingular:   kindSingular,
		KindPlural:     kindSingular + "s",
		GroupName:      groupName(config.APIVersion),
		CRDVersion:     version(config.APIVersion),
		CSVName:        getCSVName(kindSingular, catalogVersion),
		Image:          image,
		CatalogVersion: catalogVersion,
		ProjectName:    config.ProjectName,
	}
	return t.Execute(w, csvConfig)
}

func getCSVName(name, version string) string {
	return name + ".v" + version
}
