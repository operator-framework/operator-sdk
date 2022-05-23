package scorecard

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/operator-framework/operator-sdk/testutils/sample"
	kbutil "sigs.k8s.io/kubebuilder/v3/pkg/plugin/util"
)

const scorecardImage = "quay.io/operator-framework/scorecard-test:.*"
const scorecardImageReplace = "quay.io/operator-framework/scorecard-test:dev"

const customScorecardPatch = `
- op: add
  path: /stages/0/tests/-
  value:
    entrypoint:
    - custom-scorecard-tests
    - customtest1
    image: quay.io/operator-framework/custom-scorecard-tests:dev
    labels:
      suite: custom
      test: customtest1
- op: add
  path: /stages/0/tests/-
  value:
    entrypoint:
    - custom-scorecard-tests
    - customtest2
    image: quay.io/operator-framework/custom-scorecard-tests:dev
    labels:
      suite: custom
      test: customtest2
`

const customScorecardKustomize = `
- path: patches/custom.config.yaml
  target:
    group: scorecard.operatorframework.io
    version: v1alpha3
    kind: Configuration
    name: config
`

// AddScorecardCustomPatchFile adds a scorecard custom patch file
func AddScorecardCustomPatchFile(sample sample.Sample) error {
	// drop in the patch file
	customScorecardPatchFile := filepath.Join(sample.Dir(), "config", "scorecard", "patches", "custom.config.yaml")
	patchBytes := []byte(customScorecardPatch)
	err := ioutil.WriteFile(customScorecardPatchFile, patchBytes, 0777)
	if err != nil {
		fmt.Printf("can not write %s %s\n", customScorecardPatchFile, err.Error())
		return err
	}

	// append to config/scorecard/kustomization.yaml
	kustomizeFile := filepath.Join(sample.Dir(), "config", "scorecard", "kustomization.yaml")
	f, err := os.OpenFile(kustomizeFile, os.O_APPEND|os.O_WRONLY, 0777)
	if err != nil {
		fmt.Printf("error in opening scorecard kustomization.yaml file %s\n", err.Error())
		return err
	}
	defer f.Close()
	if _, err := f.WriteString(customScorecardKustomize); err != nil {
		fmt.Printf("error in append to scorecard kustomization.yaml %s\n", err.Error())
		return err
	}
	return nil
}

// ReplaceScorecardImagesForDev will replaces the scorecard images in the manifests per dev tag which is built
// in the CI based on the code changes made.
func ReplaceScorecardImagesForDev(sample sample.Sample) error {
	err := kbutil.ReplaceRegexInFile(
		filepath.Join(sample.Dir(), "config", "scorecard", "patches", "basic.config.yaml"),
		scorecardImage, scorecardImageReplace,
	)
	if err != nil {
		return err
	}

	err = kbutil.ReplaceRegexInFile(
		filepath.Join(sample.Dir(), "config", "scorecard", "patches", "olm.config.yaml"),
		scorecardImage, scorecardImageReplace,
	)
	return err
}
