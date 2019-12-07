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

package catalog

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/operator-framework/operator-sdk/internal/scaffold"
	"github.com/operator-framework/operator-sdk/internal/scaffold/input"
	"github.com/operator-framework/operator-sdk/internal/util/k8sutil"
	"github.com/operator-framework/operator-sdk/internal/util/yamlutil"

	"github.com/blang/semver"
	"github.com/ghodss/yaml"
	olmapiv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	olmversion "github.com/operator-framework/operator-lifecycle-manager/pkg/lib/version"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	OLMCatalogDir     = scaffold.DeployDir + string(filepath.Separator) + "olm-catalog"
	CSVYamlFileExt    = ".clusterserviceversion.yaml"
	CSVConfigYamlFile = "csv-config.yaml"
)

var ErrNoCSVVersion = errors.New("no CSV version supplied")

type CSV struct {
	input.Input

	// ConfigFilePath is the location of a configuration file path for this
	// projects' CSV file.
	ConfigFilePath string
	// CSVVersion is the CSV current version.
	CSVVersion string
	// FromVersion is the CSV version from which to build a new CSV. A CSV
	// manifest with this version should exist at:
	// deploy/olm-catalog/{from_version}/operator-name.v{from_version}.{CSVYamlFileExt}
	FromVersion string
	// OperatorName is the operator's name, ex. app-operator
	OperatorName string

	once       sync.Once
	fs         afero.Fs // For testing, ex. afero.NewMemMapFs()
	pathPrefix string   // For testing, ex. testdata/deploy/olm-catalog
}

func (s *CSV) initFS(fs afero.Fs) {
	s.once.Do(func() {
		s.fs = fs
	})
}

func (s *CSV) getFS() afero.Fs {
	s.initFS(afero.NewOsFs())
	return s.fs
}

func (s *CSV) GetInput() (input.Input, error) {
	// A CSV version is required.
	if s.CSVVersion == "" {
		return input.Input{}, ErrNoCSVVersion
	}
	if s.Path == "" {
		operatorName := strings.ToLower(s.OperatorName)
		// Path is what the operator-registry expects:
		// {manifests -> olm-catalog}/{operator_name}/{semver}/{operator_name}.v{semver}.clusterserviceversion.yaml
		s.Path = filepath.Join(s.pathPrefix,
			OLMCatalogDir,
			operatorName,
			s.CSVVersion,
			getCSVFileName(operatorName, s.CSVVersion),
		)
	}
	if s.ConfigFilePath == "" {
		s.ConfigFilePath = filepath.Join(s.pathPrefix, OLMCatalogDir, CSVConfigYamlFile)
	}
	return s.Input, nil
}

func (s *CSV) SetFS(fs afero.Fs) { s.initFS(fs) }

// CustomRender allows a CSV to be written by marshalling
// olmapiv1alpha1.ClusterServiceVersion instead of writing to a template.
func (s *CSV) CustomRender() ([]byte, error) {
	s.initFS(afero.NewOsFs())

	// Get current CSV to update.
	csv, exists, err := s.getBaseCSVIfExists()
	if err != nil {
		return nil, err
	}
	if !exists {
		csv = &olmapiv1alpha1.ClusterServiceVersion{}
	}

	cfg, err := GetCSVConfig(s.ConfigFilePath)
	if err != nil {
		return nil, err
	}

	if err = s.updateCSVVersions(csv); err != nil {
		return nil, err
	}
	if err = s.updateCSVFromManifests(cfg, csv); err != nil {
		return nil, err
	}
	s.setCSVDefaultFields(csv)

	if fields := getEmptyRequiredCSVFields(csv); len(fields) != 0 {
		if exists {
			log.Warnf("Required csv fields not filled in file %s:%s\n", s.Path, joinFields(fields))
		} else {
			// A new csv won't have several required fields populated.
			// Report required fields to user informationally.
			log.Infof("Fill in the following required fields in file %s:%s\n", s.Path, joinFields(fields))
		}
	}

	return k8sutil.GetObjectBytes(csv, yaml.Marshal)
}

func (s *CSV) getBaseCSVIfExists() (*olmapiv1alpha1.ClusterServiceVersion, bool, error) {
	verToGet := s.CSVVersion
	if s.FromVersion != "" {
		verToGet = s.FromVersion
	}
	csv, exists, err := getCSVFromFSIfExists(s.getFS(), s.getCSVPath(verToGet))
	if err != nil {
		return nil, false, err
	}
	if !exists && s.FromVersion != "" {
		log.Warnf("FromVersion set (%s) but CSV does not exist", s.FromVersion)
	}
	return csv, exists, nil
}

func getCSVFromFSIfExists(fs afero.Fs, path string) (*olmapiv1alpha1.ClusterServiceVersion, bool, error) {
	csvBytes, err := afero.ReadFile(fs, path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	if len(csvBytes) == 0 {
		return nil, false, nil
	}

	csv := &olmapiv1alpha1.ClusterServiceVersion{}
	if err := yaml.Unmarshal(csvBytes, csv); err != nil {
		return nil, false, errors.Wrapf(err, "error unmarshalling CSV %s", path)
	}

	return csv, true, nil
}

func getCSVName(name, version string) string {
	return name + ".v" + version
}

func getCSVFileName(name, version string) string {
	return getCSVName(name, version) + CSVYamlFileExt
}

func (s *CSV) getCSVPath(ver string) string {
	lowerOperatorName := strings.ToLower(s.OperatorName)
	name := getCSVFileName(lowerOperatorName, ver)
	return filepath.Join(s.pathPrefix, OLMCatalogDir, lowerOperatorName, ver, name)
}

// setCSVDefaultFields sets all csv fields that should be populated by a user
// to sane defaults.
func (s *CSV) setCSVDefaultFields(csv *olmapiv1alpha1.ClusterServiceVersion) {
	// These fields have well-defined required values.
	csv.TypeMeta.APIVersion = olmapiv1alpha1.ClusterServiceVersionAPIVersion
	csv.TypeMeta.Kind = olmapiv1alpha1.ClusterServiceVersionKind
	csv.SetName(getCSVName(strings.ToLower(s.OperatorName), s.CSVVersion))

	// Set if empty.
	if csv.GetNamespace() == "" {
		csv.SetNamespace("placeholder")
	}
	if csv.GetAnnotations() == nil {
		csv.SetAnnotations(map[string]string{})
	}
	if caps, ok := csv.GetAnnotations()["capabilities"]; !ok || caps == "" {
		csv.GetAnnotations()["capabilities"] = "Basic Install"
	}
	if csv.Spec.Provider == (olmapiv1alpha1.AppLink{}) {
		csv.Spec.Provider = olmapiv1alpha1.AppLink{}
	}
	if len(csv.Spec.Maintainers) == 0 {
		csv.Spec.Maintainers = []olmapiv1alpha1.Maintainer{}
	}
	if len(csv.Spec.Links) == 0 {
		csv.Spec.Links = []olmapiv1alpha1.AppLink{}
	}
	if csv.Spec.DisplayName == "" {
		csv.Spec.DisplayName = k8sutil.GetDisplayName(s.OperatorName)
	}
	if csv.Spec.Description == "" {
		csv.Spec.Description = "Placeholder description"
	}
	if csv.Spec.Maturity == "" {
		csv.Spec.Maturity = "alpha"
	}
	if len(csv.Spec.InstallModes) == 0 {
		csv.Spec.InstallModes = []olmapiv1alpha1.InstallMode{
			{Type: olmapiv1alpha1.InstallModeTypeOwnNamespace, Supported: true},
			{Type: olmapiv1alpha1.InstallModeTypeSingleNamespace, Supported: true},
			{Type: olmapiv1alpha1.InstallModeTypeMultiNamespace, Supported: false},
			{Type: olmapiv1alpha1.InstallModeTypeAllNamespaces, Supported: true},
		}
	}
}

// TODO: validate that all fields from files are populated as expected
// ex. add `resources` to a CRD

func getEmptyRequiredCSVFields(csv *olmapiv1alpha1.ClusterServiceVersion) (fields []string) {
	// Metadata
	if csv.TypeMeta.APIVersion != olmapiv1alpha1.ClusterServiceVersionAPIVersion {
		fields = append(fields, "apiVersion")
	}
	if csv.TypeMeta.Kind != olmapiv1alpha1.ClusterServiceVersionKind {
		fields = append(fields, "kind")
	}
	if csv.ObjectMeta.Name == "" {
		fields = append(fields, "metadata.name")
	}
	// Spec fields
	if csv.Spec.Version.String() == "" {
		fields = append(fields, "spec.version")
	}
	if csv.Spec.DisplayName == "" {
		fields = append(fields, "spec.displayName")
	}
	if csv.Spec.Description == "" {
		fields = append(fields, "spec.description")
	}
	if len(csv.Spec.Keywords) == 0 {
		fields = append(fields, "spec.keywords")
	}
	if len(csv.Spec.Maintainers) == 0 {
		fields = append(fields, "spec.maintainers")
	}
	if csv.Spec.Provider == (olmapiv1alpha1.AppLink{}) {
		fields = append(fields, "spec.provider")
	}
	if csv.Spec.Maturity == "" {
		fields = append(fields, "spec.maturity")
	}

	return fields
}

func joinFields(fields []string) string {
	sb := &strings.Builder{}
	for _, f := range fields {
		sb.WriteString("\n\t" + f)
	}
	return sb.String()
}

// updateCSVVersions updates csv's version and data involving the version,
// ex. ObjectMeta.Name, and place the old version in the `replaces` object,
// if there is an old version to replace.
func (s *CSV) updateCSVVersions(csv *olmapiv1alpha1.ClusterServiceVersion) error {

	// Old csv version to replace, and updated csv version.
	oldVer, newVer := csv.Spec.Version.String(), s.CSVVersion
	if oldVer == newVer {
		return nil
	}

	// Replace all references to the old operator name.
	lowerOperatorName := strings.ToLower(s.OperatorName)
	oldCSVName := getCSVName(lowerOperatorName, oldVer)
	oldRe, err := regexp.Compile(fmt.Sprintf("\\b%s\\b", regexp.QuoteMeta(oldCSVName)))
	if err != nil {
		return errors.Wrapf(err, "error compiling CSV name regexp %s", oldRe.String())
	}
	b, err := yaml.Marshal(csv)
	if err != nil {
		return err
	}
	newCSVName := getCSVName(lowerOperatorName, newVer)
	b = oldRe.ReplaceAll(b, []byte(newCSVName))
	*csv = olmapiv1alpha1.ClusterServiceVersion{}
	if err = yaml.Unmarshal(b, csv); err != nil {
		return errors.Wrapf(err, "error unmarshalling CSV %s after replacing old CSV name", csv.GetName())
	}

	ver, err := semver.Parse(s.CSVVersion)
	if err != nil {
		return err
	}
	csv.Spec.Version = olmversion.OperatorVersion{Version: ver}
	csv.Spec.Replaces = oldCSVName
	return nil
}

// updateCSVFromManifestFiles gathers relevant data from generated and
// user-defined manifests and updates csv.
func (s *CSV) updateCSVFromManifests(cfg *CSVConfig, csv *olmapiv1alpha1.ClusterServiceVersion) (err error) {
	paths := append(cfg.CRDCRPaths, cfg.OperatorPath)
	paths = append(paths, cfg.RolePaths...)
	manifestGVKMap := map[schema.GroupVersionKind][][]byte{}
	crGVKSet := map[schema.GroupVersionKind]struct{}{}
	for _, path := range paths {
		info, err := s.getFS().Stat(path)
		if err != nil {
			return err
		}
		if info.IsDir() {
			continue
		}
		b, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}
		scanner := yamlutil.NewYAMLScanner(b)
		for scanner.Scan() {
			manifest := scanner.Bytes()
			typeMeta, err := k8sutil.GetTypeMetaFromBytes(manifest)
			if err != nil {
				log.Infof("No TypeMeta in %s, skipping file", path)
				continue
			}
			gvk := typeMeta.GroupVersionKind()
			manifestGVKMap[gvk] = append(manifestGVKMap[gvk], manifest)
			switch typeMeta.Kind {
			case "CustomResourceDefinition":
				// Collect CRD kinds to filter them out from unsupported manifest types.
				// The CRD type version doesn't matter as long as it has a group, kind,
				// and versions in the expected fields.
				crd := apiextv1beta1.CustomResourceDefinition{}
				if err = yaml.Unmarshal(manifest, &crd); err != nil {
					return err
				}
				for _, ver := range crd.Spec.Versions {
					crGVK := schema.GroupVersionKind{
						Group:   crd.Spec.Group,
						Version: ver.Name,
						Kind:    crd.Spec.Names.Kind,
					}
					crGVKSet[crGVK] = struct{}{}
				}
			}
		}
		if err = scanner.Err(); err != nil {
			return err
		}
	}

	crUpdaters := crs{}
	for gvk, manifests := range manifestGVKMap {
		// We don't necessarily care about sorting by a field value, more about
		// consistent ordering.
		sort.Slice(manifests, func(i int, j int) bool {
			return string(manifests[i]) < string(manifests[j])
		})
		switch gvk.Kind {
		case "Role":
			err = roles(manifests).apply(csv)
		case "ClusterRole":
			err = clusterRoles(manifests).apply(csv)
		case "Deployment":
			err = deployments(manifests).apply(csv)
		case "CustomResourceDefinition":
			err = crds(manifests).apply(csv)
		default:
			if _, ok := crGVKSet[gvk]; ok {
				crUpdaters = append(crUpdaters, manifests...)
			} else {
				log.Infof("Skipping manifest %s", gvk)
			}
		}
		if err != nil {
			return err
		}
	}
	// Re-sort CR's since they are appended in random order.
	sort.Slice(crUpdaters, func(i int, j int) bool {
		return string(crUpdaters[i]) < string(crUpdaters[j])
	})
	if err = crUpdaters.apply(csv); err != nil {
		return err
	}
	return nil
}
