package olm

import (
	"fmt"
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"regexp"

	. "github.com/onsi/ginkgo"
	"github.com/operator-framework/operator-sdk/internal/annotations/metrics"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"
	"github.com/operator-framework/operator-sdk/testutils/sample"
	kbutil "sigs.k8s.io/kubebuilder/v3/pkg/plugin/util"
)

const (
	OlmVersionForTestSuite = "0.20.0"
)

// AddPackagemanifestsTarget will append the packagemanifests target to the makefile
// in order to test the steps described in the docs.
// More info:  https://v1-0-x.sdk.operatorframework.io/docs/olm-integration/generation/#package-manifests-formats
func AddPackagemanifestsTarget(sample sample.Sample, operatorType projutil.OperatorType) error {
	var makefilePackagemanifestsFragment = `# Options for "packagemanifests".
ifneq ($(origin FROM_VERSION), undefined)
PKG_FROM_VERSION := --from-version=$(FROM_VERSION)
endif
ifneq ($(origin CHANNEL), undefined)
PKG_CHANNELS := --channel=$(CHANNEL)
endif
ifeq ($(IS_CHANNEL_DEFAULT), 1)
PKG_IS_DEFAULT_CHANNEL := --default-channel
endif
PKG_MAN_OPTS ?= $(PKG_FROM_VERSION) $(PKG_CHANNELS) $(PKG_IS_DEFAULT_CHANNEL)
	
# Generate package manifests.
packagemanifests: kustomize %s
	operator-sdk generate kustomize manifests -q --interactive=false
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(IMG)
	$(KUSTOMIZE) build config/manifests | operator-sdk generate packagemanifests -q --version $(VERSION) $(PKG_MAN_OPTS)
`

	makefileBytes, err := ioutil.ReadFile(filepath.Join(sample.Dir(), "Makefile"))
	if err != nil {
		return err
	}

	// add the manifests target when is a Go project.
	replaceTarget := ""
	if operatorType == projutil.OperatorTypeGo {
		replaceTarget = "manifests"
	}
	makefilePackagemanifestsFragment = fmt.Sprintf(makefilePackagemanifestsFragment, replaceTarget)

	// update makefile by adding the packagemanifests target
	makefileBytes = append([]byte(makefilePackagemanifestsFragment), makefileBytes...)
	err = ioutil.WriteFile(filepath.Join(sample.Dir(), "Makefile"), makefileBytes, 0644)
	if err != nil {
		return err
	}
	return nil
}

// DisableOLMBundleInterativeMode will update the Makefile to disable the interactive mode
func DisableManifestsInteractiveMode(sample sample.Sample) error {
	// Todo: check if we cannot improve it since the replace/content will exists in the
	// pkgmanifest target if it be scaffolded before this call
	content := "operator-sdk generate kustomize manifests"
	replace := content + " --interactive=false"
	return kbutil.ReplaceInFile(filepath.Join(sample.Dir(), "Makefile"), content, replace)
}

// GenerateBundle runs all commands to create an operator bundle.
func GenerateBundle(sample sample.Sample, image string) error {
	if err := DisableManifestsInteractiveMode(sample); err != nil {
		return err
	}

	cmd := exec.Command("make", "bundle", "IMG="+image)
	if _, err := sample.CommandContext().Run(cmd, sample.Name()); err != nil {
		return err
	}

	return nil
}

// InstallOLM runs 'operator-sdk olm install' for specific version
// and returns any errors emitted by that command.
func InstallOLMVersion(sample sample.Sample, version string) error {
	cmd := exec.Command(sample.Binary(), "olm", "install", "--version", version, "--timeout", "4m")
	_, err := sample.CommandContext().Run(cmd)
	return err
}

// InstallOLM runs 'operator-sdk olm uninstall' and logs any errors emitted by that command.
func UninstallOLM(sample sample.Sample) {
	cmd := exec.Command(sample.Binary(), "olm", "uninstall")
	if _, err := sample.CommandContext().Run(cmd); err != nil {
		fmt.Fprintln(GinkgoWriter, "warning: error when uninstalling OLM:", err)
	}
}

// StripBundleAnnotations removes all annotations applied to bundle manifests and metadata
// by operator-sdk/internal/annotations/metrics annotators. Doing so decouples samples
// from which operator-sdk version they were build with, as this information is already
// available in git history.
func StripBundleAnnotations(sample sample.Sample) (err error) {
	// Remove metadata labels.
	metadataAnnotations := metrics.MakeBundleMetadataLabels("")
	metadataFiles := []string{
		filepath.Join(sample.Dir(), "bundle", "metadata", "annotations.yaml"),
		filepath.Join(sample.Dir(), "bundle.Dockerfile"),
	}
	if err = removeAllAnnotationLines(metadataAnnotations, metadataFiles); err != nil {
		return err
	}

	// Remove manifests annotations.
	manifestsAnnotations := metrics.MakeBundleObjectAnnotations("")
	manifestsFiles := []string{
		filepath.Join(sample.Dir(), "bundle", "manifests", sample.Name()+".clusterserviceversion.yaml"),
		filepath.Join(sample.Dir(), "config", "manifests", "bases", sample.Name()+".clusterserviceversion.yaml"),
	}
	if err = removeAllAnnotationLines(manifestsAnnotations, manifestsFiles); err != nil {
		return err
	}

	return nil
}

// removeAllAnnotationLines removes each line containing a key in annotations from all files at filePaths.
func removeAllAnnotationLines(annotations map[string]string, filePaths []string) error {
	var annotationREs []*regexp.Regexp
	for annotation := range annotations {
		re, err := regexp.Compile(".+" + regexp.QuoteMeta(annotation) + ".+\n")
		if err != nil {
			return fmt.Errorf("compiling annotation regexp: %v", err)
		}
		annotationREs = append(annotationREs, re)
	}

	for _, file := range filePaths {
		b, err := ioutil.ReadFile(file)
		if err != nil {
			return err
		}
		for _, re := range annotationREs {
			b = re.ReplaceAll(b, []byte{})
		}
		err = ioutil.WriteFile(file, b, 0644)
		if err != nil {
			return err
		}
	}
	return nil
}

// BuildBundleImage is a helper function to run `make bundle-build BUNDLE_IMAGE=<bundleImage>`
func BuildBundleImage(sample sample.Sample, bundleImage string) error {
	cmd := exec.Command("make", "bundle-build", "BUNDLE_IMG="+bundleImage)
	_, err := sample.CommandContext().Run(cmd, sample.Name())
	return err
}

// GeneratePackageManifests is a helper function to run `make packagemanifests IMG=<image>`
func GeneratePackageManifests(sample sample.Sample, image string) error {
	cmd := exec.Command("make", "packagemanifests", "IMG="+image)
	o, err := sample.CommandContext().Run(cmd, sample.Name())
	if err != nil {
		return fmt.Errorf("encountered an error when generating the packagemanifests: %w | OUTPUT: %s", err, o)
	}
	return nil
}
