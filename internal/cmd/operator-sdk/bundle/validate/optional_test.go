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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apimanifests "github.com/operator-framework/api/pkg/manifests"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	apierrors "github.com/operator-framework/api/pkg/validation/errors"
	"k8s.io/apimachinery/pkg/labels"
)

var _ = Describe("Running optional validators", func() {
	var (
		vals validators
	)

	BeforeEach(func() {
		vals = validators{}
	})

	Describe("run", func() {
		var (
			bundle  *apimanifests.Bundle
			results []apierrors.ManifestResult
			sel     labels.Selector
		)

		BeforeEach(func() {
			vals = optionalValidators[:1]
		})

		It("runs no validators for an empty selector", func() {
			bundle = &apimanifests.Bundle{}
			sel = labels.SelectorFromSet(map[string]string{})
			Expect(vals.run(bundle, sel, nil)).To(BeEmpty())
		})
		It("runs a validator for one selector on an empty bundle", func() {
			bundle = &apimanifests.Bundle{}
			sel = labels.SelectorFromSet(map[string]string{
				nameKey: "operatorhub",
			})
			results = vals.run(bundle, sel, map[string]string{"k8s-version": "1.22"})
			Expect(results).To(HaveLen(1))
			Expect(results[0].Errors).To(HaveLen(1))
		})
		It("runs a validator for one selector on a bundle", func() {
			bundle = &apimanifests.Bundle{}
			bundle.CSV = &v1alpha1.ClusterServiceVersion{}
			sel = labels.SelectorFromSet(map[string]string{
				nameKey: "operatorhub",
			})
			results = vals.run(bundle, sel, nil)
			Expect(results).To(HaveLen(1))
			// Only test that more than one error was returned than the empty bundle case, which
			// indicates validation happening.
			Expect(len(results[0].Errors)).To(BeNumerically(">", 1))
		})
	})

	Describe("checkMatches", func() {
		var (
			sel labels.Selector
			err error
		)

		It("returns an error for an empty selector with no validators", func() {
			sel = labels.SelectorFromSet(map[string]string{})
			err = vals.checkMatches(sel)
			Expect(err).To(HaveOccurred())
		})
		It("returns an error for an unmatched selector with no validators", func() {
			sel = labels.SelectorFromSet(map[string]string{
				nameKey: "operatorhub",
			})
			err = vals.checkMatches(sel)
			Expect(err).To(HaveOccurred())
		})
		It("returns no error for an unmatched selector with all optional validators", func() {
			sel = labels.SelectorFromSet(map[string]string{
				nameKey: "operatorhub",
			})
			vals = optionalValidators
			err = vals.checkMatches(sel)
			Expect(err).ToNot(HaveOccurred())
		})
	})

})
