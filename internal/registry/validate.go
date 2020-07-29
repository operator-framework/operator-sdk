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

package registry

import (
	"fmt"
	"io/ioutil"
	"os"

	apimanifests "github.com/operator-framework/api/pkg/manifests"
	apivalidation "github.com/operator-framework/api/pkg/validation"
	apierrors "github.com/operator-framework/api/pkg/validation/errors"
	registrybundle "github.com/operator-framework/operator-registry/pkg/lib/bundle"
	log "github.com/sirupsen/logrus"
	k8svalidation "k8s.io/apimachinery/pkg/api/validation"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/yaml"
)

// ValidateBundleContent confirms that the CSV and CRD files inside the bundle
// directory are valid and can be installed in a cluster. Other GVK types are
// also validated to confirm if they are "kubectl-able" to a cluster meaning
// if they can be applied to a cluster using `kubectl` provided users have all
// necessary permissions and configurations.
func ValidateBundleContent(logger *log.Entry, bundle *apimanifests.Bundle, mediaType string) []apierrors.ManifestResult {

	// Use errs to collect bundle-level validation errors.
	errs := apierrors.ManifestResult{
		Name: bundle.Name,
	}

	logger.Debug("Validating bundle contents")

	// helm+vX media types are not supported by this validation function.
	switch mediaType {
	case registrybundle.HelmType:
		return []apierrors.ManifestResult{errs}
	}

	for _, u := range bundle.Objects {
		// CSVs and CRDs will be validated separately.
		gvk := u.GetObjectKind().GroupVersionKind()
		if gvk.Kind == "ClusterServiceVersion" || gvk.Kind == "CustomResourceDefinition" {
			continue
		}

		logger.Debugf("Validating %s %q", gvk, u.GetName())

		// Verify if the object kind is supported for registry+v1 format.
		supported, _ := registrybundle.IsSupported(gvk.Kind)
		if mediaType == registrybundle.RegistryV1Type && !supported {
			errs.Add(apierrors.ErrInvalidBundle(fmt.Sprintf("unsupported media type %s for bundle object", mediaType), gvk))
			continue
		}

		if err := validateObject(metav1.Object(u)); err != nil {
			errs.Add(apierrors.ErrFailedValidation(err.Error(), u.GetName()))
		}
	}

	// Validate bundle itself.
	results := apivalidation.BundleValidator.Validate(bundle)

	// All bundles must have a CSV currently.
	if bundle.CSV != nil {
		results = append(results, apivalidation.ClusterServiceVersionValidator.Validate(bundle.CSV)...)
	} else {
		errs.Add(apierrors.ErrInvalidBundle("no ClusterServiceVersion in bundle", bundle.Name))
	}

	// Validate all CRD versions in the bundle together.
	var crds []interface{}
	for _, crd := range bundle.V1beta1CRDs {
		crds = append(crds, crd)
	}
	for _, crd := range bundle.V1CRDs {
		crds = append(crds, crd)
	}
	if len(crds) != 0 {
		results = append(results, apivalidation.CustomResourceDefinitionValidator.Validate(crds...)...)
	}

	// Add all other results/errors to the bundle validation results.
	results = appendResult(results, errs)

	return results
}

// validateObject validates an arbitrary metav1.Object's metadata.
func validateObject(obj metav1.Object) error {
	f := func(string, bool) []string { return nil }
	errs := k8svalidation.ValidateObjectMetaAccessor(obj, false, f, field.NewPath("metadata"))
	if len(errs) > 0 {
		return fmt.Errorf("error validating object: %s. %v", errs.ToAggregate(), obj)
	}
	return nil
}

// appendResult attempts to find a result in results that matches r.Name, and
// if found appends errors and warnings to that result. Otherwise r is added
// to the end of results.
func appendResult(results []apierrors.ManifestResult, r apierrors.ManifestResult) []apierrors.ManifestResult {
	resultIdx := -1
	for i, result := range results {
		if result.Name == r.Name {
			resultIdx = i
			break
		}
	}
	if resultIdx < 0 {
		results = append(results, r)
	} else {
		results[resultIdx].Add(r.Errors...)
		results[resultIdx].Add(r.Warnings...)
	}

	return results
}

// RewriteAnnotationsYaml unmarshalls the specified yaml file, appends the content and
// converts it again to yaml.
func RewriteAnnotationsYaml(filename string, content map[string]string) error {

	metadata, err := getAnnotationFileContents(filename)
	if err != nil {
		return err
	}

	// Append the contents to annotationsYaml
	for key, val := range content {
		metadata.Annotations[key] = val
	}

	err = writeAnnotationFile(filename, metadata)
	if err != nil {
		return err
	}

	return nil
}

func getAnnotationFileContents(filename string) (*registrybundle.AnnotationMetadata, error) {
	f, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	annotationsYaml := &registrybundle.AnnotationMetadata{}
	if err := yaml.Unmarshal(f, annotationsYaml); err != nil {
		return nil, fmt.Errorf("error parsing annotations file: %v", err)
	}
	return annotationsYaml, nil
}

func writeAnnotationFile(filename string, annotation *registrybundle.AnnotationMetadata) error {
	file, err := yaml.Marshal(annotation)
	if err != nil {
		return err
	}

	mode := os.FileMode(0666)
	if info, err := os.Stat(filename); err == nil {
		mode = info.Mode()
	}

	err = ioutil.WriteFile(filename, []byte(file), mode)
	if err != nil {
		return fmt.Errorf("error writing modified contents to annotations file, %v", err)
	}
	return nil
}
