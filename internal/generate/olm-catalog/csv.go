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
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/operator-framework/operator-sdk/internal/generate/gen"
	"github.com/operator-framework/operator-sdk/internal/scaffold"
	"github.com/operator-framework/operator-sdk/internal/util/fileutil"
	"github.com/operator-framework/operator-sdk/internal/util/k8sutil"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"

	"github.com/blang/semver"
	"github.com/ghodss/yaml"
	olmapiv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	olmversion "github.com/operator-framework/operator-lifecycle-manager/pkg/lib/version"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	OLMCatalogChildDir = "olm-catalog"
	// OLMCatalogDir is the default location for OLM catalog directory.
	OLMCatalogDir  = scaffold.DeployDir + string(filepath.Separator) + OLMCatalogChildDir
	csvYamlFileExt = ".clusterserviceversion.yaml"

	// Input keys for CSV generator whose values are the filepaths for the respective input directories

	// DeployDirKey is for the location of the operator manifests directory e.g "deploy/production"
	// The Deployment and RBAC manifests from this directory will be used to populate the CSV
	// install strategy: spec.install
	DeployDirKey = "deploy"
	// APIsDirKey is for the location of the API types directory e.g "pkg/apis"
	// The CSV annotation comments will be parsed from the types under this path.
	APIsDirKey = "apis"
	// CRDsDirKey is for the location of the CRD manifests directory e.g "deploy/crds"
	// Both the CRD and CR manifests from this path will be used to populate CSV fields
	// metadata.annotations.alm-examples for CR examples
	// and spec.customresourcedefinitions.owned for owned CRDs
	CRDsDirKey = "crds"
)

type csvGenerator struct {
	gen.Config
	// csvVersion is the CSV current version.
	csvVersion string
	// fromVersion is the CSV version from which to build a new CSV. A CSV
	// manifest with this version should exist at:
	// deploy/olm-catalog/{from_version}/operator-name.v{from_version}.{csvYamlFileExt}
	fromVersion string
	// existingCSVBundleDir is set if the generator needs to update from
	// an existing CSV bundle directory
	existingCSVBundleDir string
	// csvOutputDir is the bundle directory filepath where the CSV will be generated
	// This is set according to the generator's OutputDir
	csvOutputDir string
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

	// The olm-catalog directory location depends on where the output directory is set.
	if g.OutputDir == "" {
		g.OutputDir = scaffold.DeployDir
	}
	// Set the CSV bundle dir output path under the generator's OutputDir
	olmCatalogDir := filepath.Join(g.OutputDir, OLMCatalogChildDir)
	g.csvOutputDir = filepath.Join(olmCatalogDir, g.OperatorName, g.csvVersion)

	bundleParentDir := filepath.Join(olmCatalogDir, g.OperatorName)
	if isBundleDirExist(bundleParentDir, g.fromVersion) {
		// Upgrading a new CSV from previous CSV version
		g.existingCSVBundleDir = filepath.Join(bundleParentDir, g.fromVersion)
	} else if isBundleDirExist(bundleParentDir, g.csvVersion) {
		// Updating an existing CSV version
		g.existingCSVBundleDir = filepath.Join(bundleParentDir, g.csvVersion)
	}

	if deployDir, ok := g.Inputs[DeployDirKey]; !ok || deployDir == "" {
		g.Inputs[DeployDirKey] = scaffold.DeployDir
	}

	if apisDir, ok := g.Inputs[APIsDirKey]; !ok || apisDir == "" {
		g.Inputs[APIsDirKey] = scaffold.ApisDir
	}

	if crdsDir, ok := g.Inputs[CRDsDirKey]; !ok || crdsDir == "" {
		g.Inputs[CRDsDirKey] = filepath.Join(g.Inputs[DeployDirKey], "crds")
	}

	return g
}

func isBundleDirExist(parentDir, version string) bool {
	// Ensure full path is constructed.
	if parentDir == "" || version == "" {
		return false
	}
	bundleDir := filepath.Join(parentDir, version)
	_, err := os.Stat(bundleDir)
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}
		// TODO: return and handle this error
		log.Fatalf("Failed to stat existing bundle directory %s: %v", bundleDir, err)
	}
	return true
}

func getCSVName(name, version string) string {
	return name + ".v" + version
}

func getCSVFileName(name, version string) string {
	return getCSVName(strings.ToLower(name), version) + csvYamlFileExt
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

	if err = os.MkdirAll(g.csvOutputDir, fileutil.DefaultDirFileMode); err != nil {
		return errors.Wrapf(err, "error mkdir %s", g.csvOutputDir)
	}
	for fileName, b := range fileMap {
		path := filepath.Join(g.csvOutputDir, fileName)
		log.Debugf("CSV generator writing %s", path)
		if err = ioutil.WriteFile(path, b, fileutil.DefaultFileMode); err != nil {
			return err
		}
	}
	return nil
}

func (g csvGenerator) generate() (fileMap map[string][]byte, err error) {
	// Get current CSV to update, otherwise start with a fresh CSV.
	var csv *olmapiv1alpha1.ClusterServiceVersion
	if g.existingCSVBundleDir != "" {
		// TODO: If bundle dir exists, but the CSV file does not
		// then we should create a new one and not return an error.
		if csv, err = getCSVFromDir(g.existingCSVBundleDir); err != nil {
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

	// TODO(estroz): replace with CSV validator from API library.
	path := getCSVFileName(g.OperatorName, g.csvVersion)
	if fields := getEmptyRequiredCSVFields(csv); len(fields) != 0 {
		if g.existingCSVBundleDir != "" {
			// An existing csv should have several required fields populated.
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
		// Only read manifest from files, not directories
		if !info.IsDir() {
			b, err := ioutil.ReadFile(path)
			if err != nil {
				return nil, fmt.Errorf("error reading manifest %s: %v", path, err)
			}
			csv := &olmapiv1alpha1.ClusterServiceVersion{}
			if err := yaml.Unmarshal(b, csv); err != nil {
				log.Debugf("Skipping manifest %s: %v", path, err)
				continue
			}
			return csv, nil
		}
	}
	return nil, fmt.Errorf("no CSV manifest in %s", dir)
}

// newCSV sets all csv fields that should be populated by a user
// to sane defaults.
func newCSV(name, version string) (*olmapiv1alpha1.ClusterServiceVersion, error) {
	csv := &olmapiv1alpha1.ClusterServiceVersion{
		TypeMeta: metav1.TypeMeta{
			APIVersion: olmapiv1alpha1.ClusterServiceVersionAPIVersion,
			Kind:       olmapiv1alpha1.ClusterServiceVersionKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      getCSVName(name, version),
			Namespace: "placeholder",
			Annotations: map[string]string{
				"capabilities": "Basic Install",
				"alm-examples": "[]",
			},
		},
		Spec: olmapiv1alpha1.ClusterServiceVersionSpec{
			DisplayName: k8sutil.GetDisplayName(name),
			Provider:    olmapiv1alpha1.AppLink{},
			Maintainers: make([]olmapiv1alpha1.Maintainer, 1),
			Links:       []olmapiv1alpha1.AppLink{},
			Maturity:    "alpha",
			Icon:        make([]olmapiv1alpha1.Icon, 1),
			Keywords:    make([]string, 1),
			InstallModes: []olmapiv1alpha1.InstallMode{
				{Type: olmapiv1alpha1.InstallModeTypeOwnNamespace, Supported: true},
				{Type: olmapiv1alpha1.InstallModeTypeSingleNamespace, Supported: true},
				{Type: olmapiv1alpha1.InstallModeTypeMultiNamespace, Supported: false},
				{Type: olmapiv1alpha1.InstallModeTypeAllNamespaces, Supported: true},
			},
			InstallStrategy: olmapiv1alpha1.NamedInstallStrategy{
				StrategyName: olmapiv1alpha1.InstallStrategyNameDeployment,
				StrategySpec: olmapiv1alpha1.StrategyDetailsDeployment{
					Permissions:        []olmapiv1alpha1.StrategyDeploymentPermissions{},
					ClusterPermissions: []olmapiv1alpha1.StrategyDeploymentPermissions{},
					DeploymentSpecs:    []olmapiv1alpha1.StrategyDeploymentSpec{},
				},
			},
		},
	}

	// An empty version string will evaluate to "v0.0.0".
	if version != "" {
		ver, err := semver.Parse(version)
		if err != nil {
			return nil, err
		}
		csv.Spec.Version = olmversion.OperatorVersion{Version: ver}
	}

	return csv, nil
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
	if len(csv.Spec.Keywords) == 0 || len(csv.Spec.Keywords[0]) == 0 {
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
		return errors.Wrapf(err, "error compiling CSV name regexp %s", oldRe.String())
	}
	b, err := yaml.Marshal(csv)
	if err != nil {
		return err
	}
	newCSVName := getCSVName(g.OperatorName, newVer)
	b = oldRe.ReplaceAll(b, []byte(newCSVName))
	*csv = olmapiv1alpha1.ClusterServiceVersion{}
	if err = yaml.Unmarshal(b, csv); err != nil {
		return errors.Wrapf(err, "error unmarshalling CSV %s after replacing old CSV name", csv.GetName())
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
	// Collect all manifests in paths.
	collection := manifestCollection{}
	err = filepath.Walk(g.Inputs[DeployDirKey], func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// Only read manifest from files, not directories
		if info.IsDir() {
			return nil
		}

		b, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}
		scanner := k8sutil.NewYAMLScanner(b)
		for scanner.Scan() {
			manifest := scanner.Bytes()
			typeMeta, err := k8sutil.GetTypeMetaFromBytes(manifest)
			if err != nil {
				log.Debugf("No TypeMeta in %s, skipping file", path)
				continue
			}
			switch typeMeta.GroupVersionKind().Kind {
			case "Role":
				err = collection.addRoles(manifest)
			case "ClusterRole":
				err = collection.addClusterRoles(manifest)
			case "Deployment":
				err = collection.addDeployments(manifest)
			case "CustomResourceDefinition":
				// Skip for now and add explicitly from CRDsDir input.
			default:
				err = collection.addOthers(manifest)
			}
			if err != nil {
				return err
			}
		}
		return scanner.Err()
	})
	if err != nil {
		return fmt.Errorf("failed to walk manifests directory for CSV updates: %v", err)
	}

	// Add CRDs from input.
	crdsDir := g.Inputs[CRDsDirKey]
	if _, err := os.Stat(crdsDir); err == nil || os.IsExist(err) {
		collection.CustomResourceDefinitions, err = k8sutil.GetCustomResourceDefinitions(crdsDir)
		if err != nil {
			return err
		}
	}

	// Filter the collection based on data collected.
	collection.filter()

	// Remove duplicate manifests.
	if err = collection.deduplicate(); err != nil {
		return fmt.Errorf("error removing duplicate manifests: %v", err)
	}

	// Apply manifests to the CSV object.
	if err = collection.apply(csv); err != nil {
		return fmt.Errorf("error building CSV: %v", err)
	}

	// Update descriptions from the APIs dir.
	// FEAT(estroz): customresourcedefinition should not be updated for
	// Ansible and Helm CSV's until annotated updates are implemented.
	if projutil.IsOperatorGo() {
		err = updateDescriptions(csv, g.Inputs[APIsDirKey])
		if err != nil {
			return fmt.Errorf("error updating CSV customresourcedefinitions: %w", err)
		}
	}

	// Finally sort all updated fields.
	sortUpdates(csv)

	return nil
}
