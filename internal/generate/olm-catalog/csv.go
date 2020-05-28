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
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/operator-framework/operator-sdk/internal/generate/clusterserviceversion"
	"github.com/operator-framework/operator-sdk/internal/generate/clusterserviceversion/bases"
	"github.com/operator-framework/operator-sdk/internal/generate/collector"
	"github.com/operator-framework/operator-sdk/internal/scaffold"
	"github.com/operator-framework/operator-sdk/internal/util/fileutil"
	"github.com/operator-framework/operator-sdk/internal/util/k8sutil"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"

	"github.com/blang/semver"
	olmapiv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/operator-framework/operator-registry/pkg/lib/bundle"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/yaml"
)

// KB_INTEGRATION_TODO(estroz): generate these using kustomize and pass
// from stdin, like 'make deploy'.

const (
	OLMCatalogChildDir = "olm-catalog"
	// OLMCatalogDir is the default location for OLM catalog directory.
	OLMCatalogDir  = scaffold.DeployDir + string(filepath.Separator) + OLMCatalogChildDir
	csvYamlFileExt = ".clusterserviceversion.yaml"
)

type BundleGenerator struct {
	// OperatorName is the operator's name, ex. app-operator
	OperatorName string
	OutputDir    string
	FromVersion  string
	// csvVersion is the CSV current version.
	CSVVersion string
	// These directories specify where to retrieve manifests from.
	DeployDir, ApisDir, CRDsDir string
	// InteractivePreference refers to the user preference to enable/disable
	// interactive prompts.
	InteractivePreference projutil.InteractiveLevel
	// updateCRDs directs the generator to also add CustomResourceDefinition
	// manifests to the bundle.
	UpdateCRDs bool
	// makeManifests directs the generator to use 'manifests' as the bundle
	// dir name.
	MakeManifests bool
	// noUpdate is for testing the generator's update capabilities.
	noUpdate bool
	// fromBundleDir is set if the generator needs to update from
	// an existing CSV bundle directory
	fromBundleDir string
	// toBundleDir is the bundle directory filepath where the CSV will be generated
	// This is set according to the generator's OutputDir
	toBundleDir string
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
		case isExist(toBundleDir):
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
	case isExist(toBundleDir):
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

func (g *BundleGenerator) setDefaults() {
	if g.DeployDir == "" {
		g.DeployDir = scaffold.DeployDir
	}
	if g.ApisDir == "" {
		g.ApisDir = scaffold.ApisDir
	}
	if g.CRDsDir == "" {
		g.CRDsDir = filepath.Join(g.DeployDir, "crds")
	}

	if g.MakeManifests {
		g.toBundleDir, g.fromBundleDir = getBundleDirs(g.OperatorName, g.CSVVersion,
			g.OutputDir, g.DeployDir)
	} else {
		g.toBundleDir, g.fromBundleDir = getBundleDirsLegacy(g.OperatorName, g.CSVVersion,
			g.FromVersion, g.OutputDir, g.DeployDir)
	}
}

// Generate allows a CSV to be written by marshalling
// olmapiv1alpha1.ClusterServiceVersion instead of writing to a template.
func (g BundleGenerator) Generate() error {
	g.setDefaults()

	fileMap, err := g.generateCSV()
	if err != nil {
		return err
	}
	if len(fileMap) == 0 {
		return errors.New("error generating CSV manifest: no generated file found")
	}

	// Write CRD's to the new or updated CSV package dir.
	if g.UpdateCRDs {
		if err := addCustomResourceDefinitionsToFileSet(g.CRDsDir, fileMap); err != nil {
			return fmt.Errorf("error collecting CustomResourceDefinitions from %s: %v", g.CRDsDir, err)
		}
	}

	if err := os.MkdirAll(g.toBundleDir, 0755); err != nil {
		return fmt.Errorf("error mkdir %s: %v", g.toBundleDir, err)
	}

	for fileName, b := range fileMap {
		path := filepath.Join(g.toBundleDir, fileName)
		log.Debugf("CSV generator writing %s", path)
		if err := ioutil.WriteFile(path, b, fileutil.DefaultFileMode); err != nil {
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

func (g BundleGenerator) generateCSV() (fileMap map[string][]byte, err error) {

	csv, err := g.getBase()
	if err != nil {
		return nil, err
	}

	if err = g.updateCSVVersions(csv); err != nil {
		return nil, err
	}

	if err = g.updateCSVFromManifests(csv); err != nil {
		return nil, err
	}

	path := ""
	if g.MakeManifests {
		path = getCSVFileName(g.OperatorName)
	} else {
		path = getCSVFileNameLegacy(g.OperatorName, g.CSVVersion)
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

// getBase either reads an existing CSV from fromBundleDir or creates a new one.
func (g BundleGenerator) getBase() (*olmapiv1alpha1.ClusterServiceVersion, error) {
	v1crds, v1beta1crds, err := k8sutil.GetCustomResourceDefinitions(g.CRDsDir)
	if err != nil {
		return nil, err
	}
	var gvks []schema.GroupVersionKind
	v1crdGVKs := k8sutil.GVKsForV1CustomResourceDefinitions(v1crds...)
	gvks = append(gvks, v1crdGVKs...)
	v1beta1crdGVKs := k8sutil.GVKsForV1beta1CustomResourceDefinitions(v1beta1crds...)
	gvks = append(gvks, v1beta1crdGVKs...)

	b := bases.ClusterServiceVersion{
		OperatorName: g.OperatorName,
		OperatorType: projutil.GetOperatorType(),
		APIsDir:      g.ApisDir,
		GVKs:         gvks,
	}

	if g.fromBundleDir != "" && !g.noUpdate {
		if g.MakeManifests {
			b.BasePath = filepath.Join(g.fromBundleDir, getCSVFileName(g.OperatorName))
		} else {
			if g.FromVersion == "" {
				b.BasePath = filepath.Join(g.fromBundleDir, getCSVFileNameLegacy(g.OperatorName, g.CSVVersion))
			} else {
				b.BasePath = filepath.Join(g.fromBundleDir, getCSVFileNameLegacy(g.OperatorName, g.FromVersion))
			}
		}
	}

	// Check if user explicitly wants an interactive prompt or has no preference.
	if (g.InteractivePreference == projutil.InteractiveSoftOff && isNotExist(b.BasePath)) ||
		g.InteractivePreference == projutil.InteractiveOnAll {
		b.Interactive = true
	}

	return b.GetBase()
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
func (g BundleGenerator) updateCSVVersions(csv *olmapiv1alpha1.ClusterServiceVersion) (err error) {

	oldVer, newVer := csv.Spec.Version.String(), g.CSVVersion
	newCSVName := getCSVName(g.OperatorName, newVer)
	oldCSVName := getCSVName(g.OperatorName, oldVer)

	// If the new version is empty, either because a CSV is only being updated or
	// a base was generated, no update is needed.
	if newVer == "0.0.0" || newVer == "" || newVer == oldVer {
		return nil
	}
	if oldVer != "0.0.0" {
		csv.Spec.Replaces = oldCSVName
	}

	csv.SetName(newCSVName)
	csv.Spec.Version.Version, err = semver.Parse(newVer)
	return err
}

// updateCSVFromManifests gathers relevant data from generated and
// user-defined manifests and updates csv.
func (g BundleGenerator) updateCSVFromManifests(csv *olmapiv1alpha1.ClusterServiceVersion) (err error) {
	// Collect all manifests in paths.
	col := &collector.Manifests{}
	if err := col.UpdateFromDirs(g.DeployDir, g.CRDsDir); err != nil {
		return err
	}

	// Apply manifests to the CSV object.
	if err = clusterserviceversion.ApplyTo(col, csv); err != nil {
		return fmt.Errorf("error building CSV: %v", err)
	}

	// Ensure WATCH_NAMESPACE is set.
	if err = checkWatchNamespaces(csv); err != nil {
		return fmt.Errorf("error checking for WATCH_NAMESPACE: %v", err)
	}

	return nil
}

// OLM places the set of target namespaces for the operator in
// "metadata.annotations['olm.targetNamespaces']". This value should be
// referenced in either:
//	- The DeploymentSpec's pod spec WATCH_NAMESPACE env variable.
//	- Some other DeploymentSpec pod spec field.
func checkWatchNamespaces(csv *olmapiv1alpha1.ClusterServiceVersion) error {
	envVarValue := clusterserviceversion.TargetNamespacesRef
	for _, dep := range csv.Spec.InstallStrategy.StrategySpec.DeploymentSpecs {
		// Make sure "olm.targetNamespaces" is referenced somewhere in dep,
		// and emit a warning of not.
		b, err := dep.Spec.Template.Marshal()
		if err != nil {
			return err
		}
		if !bytes.Contains(b, []byte(envVarValue)) {
			log.Warnf("No WATCH_NAMESPACE environment variable nor reference to %q "+
				"detected in operator Deployment %s. For compatibility between OLM and a "+
				"namespaced operator, your operator must watch namespaces defined in %q",
				envVarValue, dep.Name, envVarValue)
		}
	}
	return nil
}
