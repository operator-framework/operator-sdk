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
	"errors"
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
	"github.com/operator-framework/operator-registry/pkg/lib/bundle"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// KB_INTEGRATION_TODO(estroz): generate these using kustomize and pass
// from stdin, like 'make deploy'.

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

type bundleGenerator struct {
	operatorName string
	// csvVersion is the CSV current version.
	csvVersion string
	// fromBundleDir is set if the generator needs to update from
	// an existing CSV bundle directory
	fromBundleDir string
	// toBundleDir is the bundle directory filepath where the CSV will be generated
	// This is set according to the generator's OutputDir
	toBundleDir string
	// These directories specify where to retrieve manifests from.
	deployDir, apisDir, crdsDir string
	// updateCRDs directs the generator to also add CustomResourceDefinition
	// manifests to the bundle.
	updateCRDs bool
	// makeManifests directs the generator to use 'manifests' as the bundle
	// dir name.
	makeManifests bool

	// noUpdate is for testing the generator's update capabilities.
	noUpdate bool
}

// NewBundle creates a new bundle generator.
func NewBundle(cfg gen.Config, csvVersion, fromVersion string, updateCRDs, makeManifests bool) gen.Generator {
	deployDir, apisDir, crdsDir := cfg.Inputs[DeployDirKey], cfg.Inputs[APIsDirKey], cfg.Inputs[CRDsDirKey]
	if deployDir == "" {
		deployDir = scaffold.DeployDir
	}
	if apisDir == "" {
		apisDir = scaffold.ApisDir
	}
	if crdsDir == "" {
		crdsDir = filepath.Join(deployDir, "crds")
	}

	g := bundleGenerator{
		operatorName:  cfg.OperatorName,
		csvVersion:    csvVersion,
		deployDir:     deployDir,
		apisDir:       apisDir,
		crdsDir:       crdsDir,
		updateCRDs:    updateCRDs,
		makeManifests: makeManifests,
	}

	if makeManifests {
		g.toBundleDir, g.fromBundleDir = getBundleDirs(cfg.OperatorName, csvVersion,
			cfg.OutputDir, deployDir)
	} else {
		g.toBundleDir, g.fromBundleDir = getBundleDirsLegacy(cfg.OperatorName, csvVersion,
			fromVersion, cfg.OutputDir, deployDir)
	}
	return g
}

// getBundleDirs gets directory names of the new bundle and, if it exists,
// a bundle to update the new bundle from.
// getBundleDirs is aware of 'manifests' directories and will update the
// new bundle from an existing 'manifests' directory if it exists.
func getBundleDirs(operatorName, csvVersion, outputDir, deployDir string) (toBundleDir, fromBundleDir string) {

	defaultOperatorDir := filepath.Join(deployDir, OLMCatalogChildDir, operatorName)

	// If outputDir was set, first check this dir for existing bundles. Otherwise
	// check the default location.
	if outputDir == "" {
		toBundleDir = filepath.Join(defaultOperatorDir, bundle.ManifestsDir)
	} else {
		toBundleDir = filepath.Join(outputDir, bundle.ManifestsDir)
		outputOperatorDir := filepath.Join(outputDir, OLMCatalogChildDir, operatorName)
		switch {
		case isBundleDirExist(outputOperatorDir, bundle.ManifestsDir):
			fromBundleDir = filepath.Join(outputOperatorDir, bundle.ManifestsDir)
		case isDirExist(toBundleDir):
			fromBundleDir = toBundleDir
		case isBundleDirExist(outputOperatorDir, csvVersion):
			// Updating an existing CSV version
			fromBundleDir = filepath.Join(outputOperatorDir, csvVersion)
		}
		if fromBundleDir != "" {
			return toBundleDir, fromBundleDir
		}
	}

	switch {
	case isBundleDirExist(defaultOperatorDir, bundle.ManifestsDir):
		fromBundleDir = filepath.Join(defaultOperatorDir, bundle.ManifestsDir)
	case isDirExist(toBundleDir):
		fromBundleDir = toBundleDir
	case isBundleDirExist(defaultOperatorDir, csvVersion):
		// Updating an existing CSV version
		fromBundleDir = filepath.Join(defaultOperatorDir, csvVersion)
	}

	return toBundleDir, fromBundleDir
}

// getBundleDirsLegacy gets directory names of the new bundle and, if it
// exists, a bundle to update the new bundle from.
// getBundleDirsLegacy assumes a 'manifests' directory does not exist and
// will not update the new bundle from an existing 'manifests' directory.
func getBundleDirsLegacy(operatorName, csvVersion, fromVersion, outputDir,
	deployDir string) (toBundleDir, fromBundleDir string) {

	defaultOperatorDir := filepath.Join(deployDir, OLMCatalogChildDir, operatorName)

	// If outputDir was set, first check this dir for existing bundles. Otherwise
	// check the default location.
	if outputDir == "" {
		toBundleDir = filepath.Join(defaultOperatorDir, csvVersion)
	} else {
		outputOperatorDir := filepath.Join(outputDir, OLMCatalogChildDir, operatorName)
		toBundleDir = filepath.Join(outputOperatorDir, csvVersion)
		switch {
		case isBundleDirExist(outputOperatorDir, fromVersion):
			// Upgrading a new CSV from previous CSV version
			fromBundleDir = filepath.Join(outputOperatorDir, fromVersion)
		case isBundleDirExist(outputOperatorDir, csvVersion):
			// Updating an existing CSV version
			fromBundleDir = filepath.Join(outputOperatorDir, csvVersion)
		}
		if fromBundleDir != "" {
			return toBundleDir, fromBundleDir
		}
	}

	switch {
	case isBundleDirExist(defaultOperatorDir, fromVersion):
		// Upgrading a new CSV from previous CSV version
		fromBundleDir = filepath.Join(defaultOperatorDir, fromVersion)
	case isBundleDirExist(defaultOperatorDir, csvVersion):
		// Updating an existing CSV version
		fromBundleDir = filepath.Join(defaultOperatorDir, csvVersion)
	}

	return toBundleDir, fromBundleDir
}

// Generate allows a CSV to be written by marshalling
// olmapiv1alpha1.ClusterServiceVersion instead of writing to a template.
func (g bundleGenerator) Generate() error {
	fileMap, err := g.generateCSV()
	if err != nil {
		return err
	}
	if len(fileMap) == 0 {
		return errors.New("error generating CSV manifest: no generated file found")
	}

	// Write CRD's to the new or updated CSV package dir.
	if g.updateCRDs {
		if err := addCustomResourceDefinitionsToFileSet(g.crdsDir, fileMap); err != nil {
			return fmt.Errorf("error collecting CustomResourceDefinitions from %s: %v", g.crdsDir, err)
		}
	}

	if err = os.MkdirAll(g.toBundleDir, fileutil.DefaultDirFileMode); err != nil {
		return fmt.Errorf("error mkdir %s: %v", g.toBundleDir, err)
	}
	for fileName, b := range fileMap {
		path := filepath.Join(g.toBundleDir, fileName)
		log.Debugf("CSV generator writing %s", path)
		if err = ioutil.WriteFile(path, b, fileutil.DefaultFileMode); err != nil {
			return fmt.Errorf("error writing bundle file: %v", err)
		}
	}
	return nil
}

func getCSVName(name, version string) string {
	return name + ".v" + version
}

func getCSVFileName(name string) string {
	return strings.ToLower(name) + csvYamlFileExt
}

func getCSVFileNameLegacy(name, version string) string {
	return getCSVName(strings.ToLower(name), version) + csvYamlFileExt
}

func (g bundleGenerator) generateCSV() (fileMap map[string][]byte, err error) {
	// Get current CSV to update, otherwise start with a fresh CSV.
	var csv *olmapiv1alpha1.ClusterServiceVersion
	if g.fromBundleDir != "" && !g.noUpdate {
		// TODO: If bundle dir exists, but the CSV file does not
		// then we should create a new one and not return an error.
		if csv, err = getCSVFromDir(g.fromBundleDir); err != nil {
			return nil, err
		}
		// TODO: validate existing CSV.
		if err = g.updateCSVVersions(csv); err != nil {
			return nil, err
		}
	} else {
		if csv, err = newCSV(g.operatorName, g.csvVersion); err != nil {
			return nil, err
		}
	}

	if err = g.updateCSVFromManifests(csv); err != nil {
		return nil, err
	}

	path := ""
	if g.makeManifests {
		path = getCSVFileName(g.operatorName)
	} else {
		path = getCSVFileNameLegacy(g.operatorName, g.csvVersion)
	}
	// TODO(estroz): replace with CSV validator from API library.
	if fields := getEmptyRequiredCSVFields(csv); len(fields) != 0 {
		if g.fromBundleDir != "" {
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

// updateCSVVersions updates csv's version and data involving the version,
// ex. ObjectMeta.Name, and place the old version in the `replaces` object,
// if there is an old version to replace.
func (g bundleGenerator) updateCSVVersions(csv *olmapiv1alpha1.ClusterServiceVersion) error {
	// If csvVersion is the same as the current version, or empty
	// bevause manifests/ is being updated, no version update is needed.
	oldVer, newVer := csv.Spec.Version.String(), g.csvVersion
	if newVer == "" || oldVer == newVer {
		return nil
	}

	// Replace all references to the old operator name.
	oldCSVName := getCSVName(g.operatorName, oldVer)
	oldRe, err := regexp.Compile(fmt.Sprintf("\\b%s\\b", regexp.QuoteMeta(oldCSVName)))
	if err != nil {
		return fmt.Errorf("error compiling CSV name regexp %s: %v", oldRe, err)
	}
	b, err := yaml.Marshal(csv)
	if err != nil {
		return err
	}
	newCSVName := getCSVName(g.operatorName, newVer)
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
func (g bundleGenerator) updateCSVFromManifests(csv *olmapiv1alpha1.ClusterServiceVersion) (err error) {
	// Collect all manifests in paths.
	collection := manifestCollection{}
	err = filepath.Walk(g.deployDir, func(path string, info os.FileInfo, err error) error {
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
	if isDirExist(g.crdsDir) {
		collection.CustomResourceDefinitions, err = k8sutil.GetCustomResourceDefinitions(g.crdsDir)
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
		err = updateDescriptions(csv, g.apisDir)
		if err != nil {
			return fmt.Errorf("error updating CSV customresourcedefinitions: %w", err)
		}
	}

	// Finally sort all updated fields.
	sortUpdates(csv)

	return nil
}
