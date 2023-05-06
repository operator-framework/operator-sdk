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
	"context"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apierrors "github.com/operator-framework/api/pkg/validation/errors"
)

var _ = Describe("External", func() {
	var (
		ctx context.Context

		testdataDir = "testdata"
	)

	BeforeEach(func() {
		ctx = context.Background()
	})

	Describe("getexternalvalidatorentrypoints", func() {
		It("should return entrypoints", func() {
			paths := "/path/to/validate1.sh:/path/to/validator2"
			entrypoints, hasExternal := GetExternalValidatorEntrypoints(paths)
			Expect(hasExternal).To(BeTrue())
			Expect(entrypoints).To(HaveLen(2))
			Expect(entrypoints[0]).To(Equal("/path/to/validate1.sh"))
		})
		It("should return false", func() {
			entrypoints, hasExternal := GetExternalValidatorEntrypoints("")
			Expect(hasExternal).To(BeFalse())
			Expect(entrypoints).To(BeEmpty())
		})
	})

	Context("passing validator", func() { //nolint:dupl
		It("runs successfully", func() {
			entrypoints, hasExternal := GetExternalValidatorEntrypoints(filepath.Join(testdataDir, "passes.sh"))
			Expect(hasExternal).To(BeTrue())
			results, err := RunExternalValidators(ctx, entrypoints, "foo/bar")
			Expect(err).NotTo(HaveOccurred())
			Expect(results).To(HaveLen(1))
			Expect(results[0].Name).To(Equal("passes-bundle"))
			Expect(results[0].Errors).To(BeEmpty())
		})
	})

	Context("failing validator", func() { //nolint:dupl
		It("fails with one error", func() {
			entrypoints, hasExternal := GetExternalValidatorEntrypoints(filepath.Join(testdataDir, "fails.sh"))
			Expect(hasExternal).To(BeTrue())
			results, err := RunExternalValidators(ctx, entrypoints, "foo/bar")
			Expect(err).NotTo(HaveOccurred())
			Expect(results).To(HaveLen(1))
			Expect(results[0].Name).To(Equal("fails-bundle"))
			Expect(results[0].Errors).To(HaveLen(1))
			Expect(results[0].Errors[0].Type).To(Equal(apierrors.ErrorInvalidCSV))
			Expect(results[0].Errors[0].Detail).To(Equal("invalid field Pesce"))
		})
	})

	Context("errored validator", func() {
		It("does not run", func() {
			entrypoints, hasExternal := GetExternalValidatorEntrypoints(filepath.Join(testdataDir, "errors.sh"))
			Expect(hasExternal).To(BeTrue())
			stderrBuf := &bytes.Buffer{}
			stderr = stderrBuf
			results, err := RunExternalValidators(ctx, entrypoints, "foo/bar")
			Expect(err).To(HaveOccurred())
			Expect(stderrBuf.String()).To(Equal("validator runtime error"))
			Expect(results).To(BeEmpty())
		})
	})

})
