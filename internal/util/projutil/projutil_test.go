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

package projutil

import (
	"errors"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Testing projutil helpers", func() {
	Describe("Testing RewriteFileContents", func() {
		var (
			fileContents   string
			instruction    string
			content        string
			expectedOutput string
		)
		It("Should pass when file has instruction", func() {
			fileContents = "LABEL operators.operatorframework.io.bundle.mediatype.v1=registry+v1 \n" +
				"LABEL operators.operatorframework.io.bundle.manifests.v1=manifests/ \n" +
				"LABEL operators.operatorframework.io.bundle.metadata.v1=metadata/ \n" +
				"COPY deploy/olm-catalog/memcached-operator/manifests /manifests/ \n"

			instruction = "LABEL"

			content = "LABEL operators.operatorframework.io.bundle.tests.v1=tests/ \n"

			expectedOutput = "LABEL operators.operatorframework.io.bundle.mediatype.v1=registry+v1 \n" +
				"LABEL operators.operatorframework.io.bundle.manifests.v1=manifests/ \n" +
				"LABEL operators.operatorframework.io.bundle.metadata.v1=metadata/ \n" +
				"LABEL operators.operatorframework.io.bundle.tests.v1=tests/ \n" +
				"COPY deploy/olm-catalog/memcached-operator/manifests /manifests/ \n"

			Expect(appendContent(fileContents, instruction, content)).To(Equal(expectedOutput))
		})

		It("Should result in error when file does not have instruction", func() {
			fileContents = "LABEL operators.operatorframework.io.bundle.mediatype.v1=registry+v1 \n" +
				"LABEL operators.operatorframework.io.bundle.manifests.v1=manifests/ \n" +
				"LABEL operators.operatorframework.io.bundle.metadata.v1=metadata/ \n" +
				"COPY deploy/olm-catalog/memcached-operator/manifests /manifests/ \n"

			instruction = "ADD"

			content = "ADD operators.operatorframework.io.bundle.tests.v1=tests/ \n"

			expectedOutput = "LABEL operators.operatorframework.io.bundle.mediatype.v1=registry+v1 \n" +
				"LABEL operators.operatorframework.io.bundle.manifests.v1=manifests/ \n" +
				"LABEL operators.operatorframework.io.bundle.metadata.v1=metadata/ \n" +
				"LABEL operators.operatorframework.io.bundle.tests.v1=tests/ \n" +
				"COPY deploy/olm-catalog/memcached-operator/manifests /manifests/ \n"

			_, err := appendContent(fileContents, instruction, content)

			Expect(err).Should(MatchError(errors.New("no prior string ADD in newContent")))
		})

		It("Should result in error as no new line at the end of dockerfile command", func() {
			fileContents = "LABEL operators.operatorframework.io.bundle.mediatype.v1=registry+v1 \n" +
				"LABEL operators.operatorframework.io.bundle.manifests.v1=manifests/ \n" +
				"LABEL operators.operatorframework.io.bundle.metadata.v1=metadata/"

			instruction = "LABEL"

			content = "LABEL operators.operatorframework.io.bundle.tests.v1=tests/ \n"

			expectedOutput = "LABEL operators.operatorframework.io.bundle.mediatype.v1=registry+v1 \n" +
				"LABEL operators.operatorframework.io.bundle.manifests.v1=manifests/ \n" +
				"LABEL operators.operatorframework.io.bundle.metadata.v1=metadata/ \n" +
				"LABEL operators.operatorframework.io.bundle.tests.v1=tests/ \n"

			_, err := appendContent(fileContents, instruction, content)

			Expect(err).ShouldNot((BeNil()))
		})

	})
})

func TestMetadata(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Projutil Helpers suite")
}
