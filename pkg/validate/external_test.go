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
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

var _ = Describe("External", func() {
	var (
		ctx context.Context

		oldEnv      = os.Getenv(ValidatorEntrypointsEnv)
		testdataDir = "testdata"
	)

	BeforeEach(func() {
		ctx = context.Background()
	})
	AfterEach(func() {
		Expect(os.Setenv(ValidatorEntrypointsEnv, oldEnv)).To(Succeed())
	})

	Context("passing validator", func() { //nolint:dupl
		It("runs successfully", func() {
			Expect(os.Setenv(ValidatorEntrypointsEnv, filepath.Join(testdataDir, "passes.sh"))).To(Succeed())
			entrypoints, hasExternal := GetExternalValidatorEntrypoints()
			Expect(hasExternal).To(BeTrue())
			results, err := RunExternalValidators(ctx, entrypoints, "foo/bar")
			Expect(err).NotTo(HaveOccurred())
			Expect(results).To(HaveLen(1))
			Expect(results[0].Passed).To(BeTrue())
			Expect(results[0].Outputs).To(HaveLen(2))
			Expect(results[0].Outputs[0].Type).To(Equal(logrus.InfoLevel.String()))
			Expect(results[0].Outputs[0].Message).To(Equal("found bundle: foo/bar"))
			Expect(results[0].Outputs[1].Type).To(Equal(logrus.WarnLevel.String()))
			Expect(results[0].Outputs[1].Message).To(Equal("foo"))
		})
	})

	Context("failing validator", func() { //nolint:dupl
		It("fails with one error", func() {
			Expect(os.Setenv(ValidatorEntrypointsEnv, filepath.Join(testdataDir, "fails.sh"))).To(Succeed())
			entrypoints, hasExternal := GetExternalValidatorEntrypoints()
			Expect(hasExternal).To(BeTrue())
			results, err := RunExternalValidators(ctx, entrypoints, "foo/bar")
			Expect(err).NotTo(HaveOccurred())
			Expect(results).To(HaveLen(1))
			Expect(results[0].Passed).To(BeFalse())
			Expect(results[0].Outputs).To(HaveLen(2))
			Expect(results[0].Outputs[0].Type).To(Equal(logrus.InfoLevel.String()))
			Expect(results[0].Outputs[0].Message).To(Equal("found bundle: foo/bar"))
			Expect(results[0].Outputs[1].Type).To(Equal(logrus.ErrorLevel.String()))
			Expect(results[0].Outputs[1].Message).To(Equal("got error"))
		})
	})

	Context("errored validator", func() {
		It("doesn not run", func() {
			Expect(os.Setenv(ValidatorEntrypointsEnv, filepath.Join(testdataDir, "errors.sh"))).To(Succeed())
			entrypoints, hasExternal := GetExternalValidatorEntrypoints()
			Expect(hasExternal).To(BeTrue())
			stderrBuf := &bytes.Buffer{}
			stderr = stderrBuf
			results, err := RunExternalValidators(ctx, entrypoints, "foo/bar")
			Expect(err).To(HaveOccurred())
			Expect(stderrBuf.String()).To(Equal("validator runtime error"))
			Expect(results).To(HaveLen(0))
		})
	})

})
