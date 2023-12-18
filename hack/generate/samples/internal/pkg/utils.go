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

package pkg

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	log "github.com/sirupsen/logrus"

	"github.com/operator-framework/operator-sdk/internal/annotations/metrics"
)

// CheckError will exit with exit code 1 when err is not nil.
func CheckError(msg string, err error) {
	if err != nil {
		log.Errorf("error %s: %s", msg, err)
		os.Exit(1)
	}
}

// StripBundleAnnotations removes all annotations applied to bundle manifests and metadata
// by operator-sdk/internal/annotations/metrics annotators. Doing so decouples samples
// from which operator-sdk version they were build with, as this information is already
// available in git history.
func (ctx SampleContext) StripBundleAnnotations() (err error) {
	// Remove metadata labels.
	metadataAnnotations := metrics.MakeBundleMetadataLabels("")
	metadataFiles := []string{
		filepath.Join(ctx.Dir, "bundle", "metadata", "annotations.yaml"),
		filepath.Join(ctx.Dir, "bundle.Dockerfile"),
	}
	if err = removeAllAnnotationLines(metadataAnnotations, metadataFiles); err != nil {
		return err
	}

	// Remove manifests annotations.
	manifestsAnnotations := metrics.MakeBundleObjectAnnotations("")
	manifestsFiles := []string{
		filepath.Join(ctx.Dir, "bundle", "manifests", ctx.ProjectName+".clusterserviceversion.yaml"),
		filepath.Join(ctx.Dir, "config", "manifests", "bases", ctx.ProjectName+".clusterserviceversion.yaml"),
	}

	return removeAllAnnotationLines(manifestsAnnotations, manifestsFiles)
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
		b, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		for _, re := range annotationREs {
			b = re.ReplaceAll(b, []byte{})
		}
		err = os.WriteFile(file, b, 0644)
		if err != nil {
			return err
		}
	}
	return nil
}
