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

package tests

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/operator-framework/operator-registry/pkg/registry"
	scapiv1alpha2 "github.com/operator-framework/operator-sdk/pkg/apis/scorecard/v1alpha2"
	log "github.com/sirupsen/logrus"

	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// GetCRs parses a Bundle's CSV for CRs
func GetCRs(bundle registry.Bundle) (crList []unstructured.Unstructured, err error) {

	// get CRs from CSV's alm-examples annotation, assume single bundle
	csv, err := bundle.ClusterServiceVersion()
	if err != nil {
		return crList, fmt.Errorf("error in csv retrieval %s", err.Error())
	}

	if csv.GetAnnotations() == nil {
		return crList, nil
	}

	almExamples := csv.ObjectMeta.Annotations["alm-examples"]

	if almExamples == "" {
		return crList, nil
	}

	err = json.Unmarshal([]byte(almExamples), &crList)
	if err != nil {
		return nil, fmt.Errorf("failed to parse alm-examples annotation: %v", err)
	}
	return crList, nil
}

func validateCR(cr unstructured.Unstructured, crds []*v1beta1.CustomResourceDefinition,
	r scapiv1alpha2.ScorecardTestResult) scapiv1alpha2.ScorecardTestResult {

	// check if the CRD matches the testing CR
	for _, crd := range crds {
		gvk := cr.GroupVersionKind()
		// Only check the validation block if the CRD and CR have the same Kind and Version
		if !(matchVersion(gvk.Version, crd) && matchKind(gvk.Kind, crd.Spec.Names.Kind)) {
			continue
		}
		if crd.Spec.Validation == nil {
			r.Suggestions = append(r.Suggestions, fmt.Sprintf("Add CRD validation for %s/%s",
				crd.Spec.Names.Kind, crd.Spec.Version))
			continue
		}
		failed := false
		if cr.Object["spec"] != nil {
			spec := cr.Object["spec"].(map[string]interface{})
			for key := range spec {
				if _, ok := crd.Spec.Validation.OpenAPIV3Schema.Properties["spec"].Properties[key]; !ok {
					failed = true
					r.Suggestions = append(r.Suggestions,
						fmt.Sprintf("Add CRD validation for spec field `%s` in %s/%s",
							key, gvk.Kind, gvk.Version))
				}
			}
		}
		if cr.Object["status"] != nil {
			status := cr.Object["status"].(map[string]interface{})
			for key := range status {
				if _, ok := crd.Spec.Validation.OpenAPIV3Schema.Properties["status"].Properties[key]; !ok {
					failed = true
					r.Suggestions = append(r.Suggestions, fmt.Sprintf("Add CRD validation for status"+
						" field `%s` in %s/%s", key, gvk.Kind, gvk.Version))
				}
			}
		}
		if failed {
			r.State = scapiv1alpha2.FailState
		}
	}
	return r
}

// matchVersion checks if a CRD contains a specified version in a case insensitive manner
func matchVersion(version string, crd *v1beta1.CustomResourceDefinition) bool {
	if strings.EqualFold(version, crd.Spec.Version) {
		return true
	}
	// crd.Spec.Version is deprecated, so check in crd.Spec.Versions as well
	for _, currVer := range crd.Spec.Versions {
		if strings.EqualFold(version, currVer.Name) {
			return true
		}
	}
	return false
}

func matchKind(kind1, kind2 string) bool {

	var restMapper meta.DefaultRESTMapper
	singularKind1, err := restMapper.ResourceSingularizer(kind1)
	if err != nil {
		singularKind1 = kind1
		log.Warningf("could not find singular version of %s", kind1)
	}
	singularKind2, err := restMapper.ResourceSingularizer(kind2)
	if err != nil {
		singularKind2 = kind2
		log.Warningf("could not find singular version of %s", kind2)
	}
	return strings.EqualFold(singularKind1, singularKind2)
}
