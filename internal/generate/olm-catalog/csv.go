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

package olmcatalog

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/operator-framework/operator-sdk/internal/generate/gen"
	"github.com/operator-framework/operator-sdk/internal/scaffold"
	"github.com/operator-framework/operator-sdk/internal/util/fileutil"
	"github.com/operator-framework/operator-sdk/internal/util/k8sutil"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"
	"github.com/operator-framework/operator-sdk/internal/util/yamlutil"

	"github.com/blang/semver"
	"github.com/ghodss/yaml"
	olmapiv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	olminstall "github.com/operator-framework/operator-lifecycle-manager/pkg/controller/install"
	olmversion "github.com/operator-framework/operator-lifecycle-manager/pkg/lib/version"
	log "github.com/sirupsen/logrus"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	olmCatalogChildDir = "olm-catalog"
	OLMCatalogDir      = scaffold.DeployDir + string(filepath.Separator) + olmCatalogChildDir
	CSVYamlFileExt     = ".clusterserviceversion.yaml"

	BundleDirKey = "bundle"
	DeployDirKey = "deploy"
	APIsDirKey   = "apis"
)

type csvGenerator struct {
	gen.Config
	// csvVersion is the CSV current version.
	csvVersion string
	// fromVersion is the CSV version from which to build a new CSV. A CSV
	// manifest with this version should exist at:
	// deploy/olm-catalog/{from_version}/operator-name.v{from_version}.{CSVYamlFileExt}
	fromVersion string
}

func NewCSV(cfg gen.Config, csvVersion, fromVersion string) gen.Generator {
	g := csvGenerator{
		Config:      cfg,
		csvVersion:  csvVersion,
		fromVersion: fromVersion,
	}
	if g.Inputs == nil {
		g.Inputs = map[string]string{}
	}
	// The olm-catalog directory location depends on where the deploy (operator
	// manifests) directory is.
	olmCatalogDir := OLMCatalogDir
	if deployDir, ok := g.Inputs[DeployDirKey]; !ok || deployDir == "" {
		g.Inputs[DeployDirKey] = scaffold.DeployDir
	} else {
		// Set to non-standard location.
		olmCatalogDir = filepath.Join(g.Inputs[DeployDirKey], olmCatalogChildDir)
	}
	if bundle, ok := g.Inputs[BundleDirKey]; !ok || bundle == "" {
		// Initialize so we don't have to check for key existence elsewhere.
		g.Inputs[BundleDirKey] = ""
		parentDir := filepath.Join(olmCatalogDir, g.OperatorName)
		if isBundleDirExist(parentDir, g.fromVersion) {
			g.Inputs[BundleDirKey] = filepath.Join(olmCatalogDir, g.OperatorName, g.fromVersion)
		} else if isBundleDirExist(parentDir, g.csvVersion) {
			g.Inputs[BundleDirKey] = filepath.Join(olmCatalogDir, g.OperatorName, g.csvVersion)
		}
	}
	if apisDir, ok := g.Inputs[APIsDirKey]; !ok || apisDir == "" {
		g.Inputs[APIsDirKey] = scaffold.ApisDir
	}
	if g.OutputDir == "" {
		g.OutputDir = filepath.Join(olmCatalogDir, g.OperatorName, g.csvVersion)
	}
	return g
}

func isBundleDirExist(parentDir, version string) bool {
	// Ensure full path is constructed.
	if parentDir == "" || version == "" {
		return false
	}
	_, err := os.Stat(filepath.Join(parentDir, version))
	return err == nil || os.IsExist(err)
}

func getCSVName(name, version string) string {
	return name + ".v" + version
}

func getCSVFileName(name, version string) string {
	return getCSVName(strings.ToLower(name), version) + CSVYamlFileExt
}

// Generate allows a CSV to be written by marshalling
// olmapiv1alpha1.ClusterServiceVersion instead of writing to a template.
func (g csvGenerator) Generate() error {
	fileMap, err := g.generate()
	if err != nil {
		return err
	}
	if len(fileMap) == 0 {
		return errors.New("error generating CSV manifest: no generated file found")
	}
	if err = os.MkdirAll(g.OutputDir, fileutil.DefaultDirFileMode); err != nil {
		return fmt.Errorf("error mkdir %s: %v", g.OutputDir, err)
	}
	for fileName, b := range fileMap {
		path := filepath.Join(g.OutputDir, fileName)
		if err = ioutil.WriteFile(path, b, fileutil.DefaultFileMode); err != nil {
			return err
		}
	}
	return nil
}

func (g csvGenerator) generate() (fileMap map[string][]byte, err error) {
	bundle := g.Inputs[BundleDirKey]
	// Get current CSV to update, otherwise start with a fresh CSV.
	var csv *olmapiv1alpha1.ClusterServiceVersion
	if bundle != "" {
		if csv, err = getCSVFromDir(bundle); err != nil {
			return nil, err
		}
		// TODO: validate existing CSV.
		if err = g.updateCSVVersions(csv); err != nil {
			return nil, err
		}
	} else {
		if csv, err = newCSV(g.OperatorName, g.csvVersion); err != nil {
			return nil, err
		}
	}

	if err = g.updateCSVFromManifests(csv); err != nil {
		return nil, err
	}

	path := getCSVFileName(g.OperatorName, g.csvVersion)
	if fields := getEmptyRequiredCSVFields(csv); len(fields) != 0 {
		if bundle != "" {
			log.Warnf("Required csv fields not filled in file %s:%s\n", path, joinFields(fields))
		} else {
			// A new csv won't have several required fields populated.
			// Report required fields to user informationally.
			log.Infof("Fill in the following required fields in file %s:%s\n", path, joinFields(fields))
		}
	}

	b, err := k8sutil.GetObjectBytes(csv, yaml.Marshal)
	if err != nil {
		return nil, err
	}
	fileMap = map[string][]byte{
		path: b,
	}
	return fileMap, nil
}

func getCSVFromDir(dir string) (*olmapiv1alpha1.ClusterServiceVersion, error) {
	infos, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, info := range infos {
		path := filepath.Join(dir, info.Name())
		info, err := os.Stat(path)
		if err != nil || info.IsDir() {
			// Skip any directories or files accessed in error.
			continue
		}
		b, err := ioutil.ReadFile(path)
		if err != nil {
			return nil, err
		}
		typeMeta, err := k8sutil.GetTypeMetaFromBytes(b)
		if err != nil {
			return nil, err
		}
		if typeMeta.Kind != olmapiv1alpha1.ClusterServiceVersionKind {
			continue
		}
		csv := &olmapiv1alpha1.ClusterServiceVersion{}
		if err := yaml.Unmarshal(b, csv); err != nil {
			return nil, fmt.Errorf("error unmarshalling CSV %s: %v", path, err)
		}
		return csv, nil
	}
	return nil, fmt.Errorf("no CSV manifest in %s", dir)
}

// newCSV sets all csv fields that should be populated by a user
// to sane defaults.
func newCSV(name, version string) (*olmapiv1alpha1.ClusterServiceVersion, error) {
	ver, err := semver.Parse(version)
	if err != nil {
		return nil, err
	}
	return &olmapiv1alpha1.ClusterServiceVersion{
		TypeMeta: metav1.TypeMeta{
			APIVersion: olmapiv1alpha1.ClusterServiceVersionAPIVersion,
			Kind:       olmapiv1alpha1.ClusterServiceVersionKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      getCSVName(name, version),
			Namespace: "placeholder",
			Annotations: map[string]string{
				"capabilities": "Basic Install",
			},
		},
		Spec: olmapiv1alpha1.ClusterServiceVersionSpec{
			DisplayName: k8sutil.GetDisplayName(name),
			Description: "",
			Provider:    olmapiv1alpha1.AppLink{},
			Maintainers: []olmapiv1alpha1.Maintainer{},
			Links:       []olmapiv1alpha1.AppLink{},
			Maturity:    "alpha",
			Version:     olmversion.OperatorVersion{Version: ver},
			InstallModes: []olmapiv1alpha1.InstallMode{
				{Type: olmapiv1alpha1.InstallModeTypeOwnNamespace, Supported: true},
				{Type: olmapiv1alpha1.InstallModeTypeSingleNamespace, Supported: true},
				{Type: olmapiv1alpha1.InstallModeTypeMultiNamespace, Supported: false},
				{Type: olmapiv1alpha1.InstallModeTypeAllNamespaces, Supported: true},
			},
			InstallStrategy: olmapiv1alpha1.NamedInstallStrategy{
				StrategyName:    olminstall.InstallStrategyNameDeployment,
				StrategySpecRaw: json.RawMessage("{}"),
			},
		},
	}, nil
}

// TODO: replace with validation library.
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
func (g csvGenerator) updateCSVVersions(csv *olmapiv1alpha1.ClusterServiceVersion) error {

	// Old csv version to replace, and updated csv version.
	oldVer, newVer := csv.Spec.Version.String(), g.csvVersion
	if oldVer == newVer {
		return nil
	}

	// Replace all references to the old operator name.
	oldCSVName := getCSVName(g.OperatorName, oldVer)
	oldRe, err := regexp.Compile(fmt.Sprintf("\\b%s\\b", regexp.QuoteMeta(oldCSVName)))
	if err != nil {
		return fmt.Errorf("error compiling CSV name regexp %s: %v", oldRe.String(), err)
	}
	b, err := yaml.Marshal(csv)
	if err != nil {
		return err
	}
	newCSVName := getCSVName(g.OperatorName, newVer)
	b = oldRe.ReplaceAll(b, []byte(newCSVName))
	*csv = olmapiv1alpha1.ClusterServiceVersion{}
	if err = yaml.Unmarshal(b, csv); err != nil {
		return fmt.Errorf("error unmarshalling CSV %s after replacing old CSV name: %v", csv.GetName(), err)
	}

	ver, err := semver.Parse(g.csvVersion)
	if err != nil {
		return err
	}
	csv.Spec.Version = olmversion.OperatorVersion{Version: ver}
	csv.Spec.Replaces = oldCSVName
	return nil
}

// updateCSVFromManifests gathers relevant data from generated and
// user-defined manifests and updates csv.
func (g csvGenerator) updateCSVFromManifests(csv *olmapiv1alpha1.ClusterServiceVersion) (err error) {
	manifestsDir := filepath.Join(g.Inputs[DeployDirKey], olmCatalogChildDir)
	kindManifestMap := map[schema.GroupVersionKind][][]byte{}
	crGVKSet := map[schema.GroupVersionKind]struct{}{}
	err = filepath.Walk(g.Inputs[DeployDirKey], func(path string, info os.FileInfo, werr error) error {
		if werr != nil || info.IsDir() {
			return werr
		}
		// Ignore any paths not satisfying the filter or files already in the
		// manifests dir.
		if !g.Filters.SatisfiesAny(path) || strings.HasPrefix(path, manifestsDir) {
			return nil
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
			kindManifestMap[gvk] = append(kindManifestMap[gvk], manifest)
			switch typeMeta.Kind {
			case "CustomResourceDefinition":
				// Collect CRD kinds to filter them out from unsupported manifest types.
				// The CRD version type doesn't matter as long as it has a group, kind,
				// and versions in the expected fields.
				crd := v1beta1.CustomResourceDefinition{}
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
		return scanner.Err()
	})

	crUpdaters := crs{}
	for gvk, manifests := range kindManifestMap {
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
				crUpdaters = append(crUpdaters, crs(manifests)...)
			} else {
				log.Infof("Skipping manifest %s", gvk)
			}
		}
		if err != nil {
			return err
		}
	}
	err = updateDescriptions(csv, g.Inputs[APIsDirKey], projutil.GetOperatorType())
	if err != nil {
		return fmt.Errorf("error updating CSV customresourcedefinitions: %w", err)
	}
	// Re-sort CR's since they are appended in random order.
	if len(crUpdaters) != 0 {
		sort.Slice(crUpdaters, func(i int, j int) bool {
			return string(crUpdaters[i]) < string(crUpdaters[j])
		})
		if err = crUpdaters.apply(csv); err != nil {
			return err
		}
	}
	return nil
}
