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

package validate

import (
	"bytes"
	"fmt"
	"io"
	"text/tabwriter"

	apimanifests "github.com/operator-framework/api/pkg/manifests"
	apivalidation "github.com/operator-framework/api/pkg/validation"
	apierrors "github.com/operator-framework/api/pkg/validation/errors"
	interfaces "github.com/operator-framework/api/pkg/validation/interfaces"
	"k8s.io/apimachinery/pkg/labels"
)

// Keys for label selectors to be used by all validators.
const (
	nameKey  = "name"
	suiteKey = "suite"
)

// optionalValidators is a list of validators with 0their name, labels for CLI usage, and a light description.
var optionalValidators = validators{
	{
		Validator: apivalidation.OperatorHubValidator,
		name:      "operatorhub",
		labels: map[string]string{
			nameKey:  "operatorhub",
			suiteKey: "operatorframework",
		},
		desc: "OperatorHub.io metadata validation. ",
	},
	{
		Validator: apivalidation.CommunityOperatorValidator,
		name:      "community",
		labels: map[string]string{
			nameKey: "community",
		},
		desc: "(stage: alpha) Community Operator bundle validation. See https://github.com/operator-framework/community-operators/blob/master/docs/packaging-required-fields.md",
	},
	{
		Validator: apivalidation.AlphaDeprecatedAPIsValidator,
		name:      "alpha-deprecated-apis",
		labels: map[string]string{
			nameKey: "alpha-deprecated-apis",
		},
		desc: "(stage: alpha) Deprecated APIs bundle validation. This valiator can help you out verify if your bundle contains manifests which uses deprecated APIs. More info: https://kubernetes.io/docs/reference/using-api/deprecation-guide/",
	},
}

// runOptionalValidators runs optional validators selected by sel on bundle.
func runOptionalValidators(bundle *apimanifests.Bundle, sel labels.Selector, optionalValues map[string]string) []apierrors.ManifestResult {
	return optionalValidators.run(bundle, sel, optionalValues)
}

// listOptionalValidators lists all optional validators.
func listOptionalValidators(w io.Writer) error {
	_, err := fmt.Fprint(w, optionalValidators.String())
	return err
}

// validator can validate a set of bundle objects and report information about those objects.
type validator struct {
	interfaces.Validator
	name   string
	labels map[string]string
	desc   string
}

type validators []validator

func (vals validators) String() string {
	out := &bytes.Buffer{}
	tw := tabwriter.NewWriter(out, 8, 4, 4, ' ', 0)
	fmt.Fprintf(tw, "NAME\tLABELS\tDESCRIPTION\n")
	for _, val := range vals {
		var labelStrs []string
		for k, v := range val.labels {
			labelStrs = append(labelStrs, fmt.Sprintf("%s=%s", k, v))
		}
		if len(labelStrs) != 0 {
			fmt.Fprintf(tw, "%s\t%s\t%s\n", val.name, labelStrs[0], val.desc)
		}
		if len(labelStrs) > 1 {
			for _, labelStr := range labelStrs[1:] {
				fmt.Fprintf(tw, "\t%s\t\n", labelStr)
			}
		}
	}
	tw.Flush()
	return out.String()
}

// checkMatches returns an error if sel does not match any validators. This method helps the CLI
// to fail early in case of erroneous input.
func (vals validators) checkMatches(sel labels.Selector) error {
	for _, v := range vals {
		if sel.Matches(labels.Set(v.labels)) {
			return nil
		}
	}
	return fmt.Errorf("selector %q does not match any validator labels", sel.String())
}

// run runs optional validators selected by sel on bundle.
func (vals validators) run(bundle *apimanifests.Bundle, sel labels.Selector, optionalValues map[string]string) (results []apierrors.ManifestResult) {
	// No selector set, do not run any optional validators.
	if sel == nil || sel.String() == "" {
		return results
	}

	// Pass all exposed bundle objects to the validator, since the underlying validator could filter by type
	// or arbitrary unstructured object keys.
	// NB(estroz): we may also want to pass metadata to these validators, however the set of metadata in a bundle
	// object is not complete (only dependencies, no annotations).
	objs := bundle.ObjectsToValidate()
	for _, obj := range bundle.Objects {
		objs = append(objs, obj)
	}

	// Pass the --optional-values. e.g. --optional-values="k8s-version=1.22"
	// or --optional-values="image-path=bundle.Dockerfile"
	objs = append(objs, optionalValues)

	for _, v := range vals {
		if sel.Matches(labels.Set(v.labels)) {
			results = append(results, v.Validate(objs...)...)
		}
	}

	return results
}
