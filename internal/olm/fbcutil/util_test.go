// Copyright 2023 The Operator-SDK Authors
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

package fbcutil

import (
	"crypto/sha256"
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("test fbutil", func() {

	emptyListHash := fmt.Sprintf("%x", sha256.New().Sum(nil))

	Context("test dirNameFromRefs", func() {
		DescribeTable("should name the directory with sha256 of the concatenation of the bundle image names", func(refs []string, expectedHash string) {
			dirName := dirNameFromRefs(refs)

			Expect(dirName).Should(HaveLen(sha256.Size * 2)) // in hex representation, each byte is two hex digits
			Expect(dirName).Should(Equal(expectedHash))
		},
			Entry("long image names", []string{
				"BtU5KRr8IWafnGTvckShoj3xBb5duLZp/XKHZRpOqdVxhHQkCL0Dy0lSRw0a0M/y158JRQKk1S@6KAauQHujQ30my9sivVYGZahR7R7UUSoUBUmnuFdqGHiUTT0aV5Di2",
				"lKM63iKupQcUPKd6AAsmRRABbGYNmwFTmQEX6fpswndQdb/niJPLRG8WhzaH84Q3kfZC/7hc3nK7Oeq@L5KxmVqbAz6jXlv1yKna2cH4zbZ3be0pcYNHyCSVUG/ZVqRAo",
			}, "cc048355dce06491fd090ce0c3ce5a48db3528250ad13a4fbf4090a3de8c325a"),
			Entry("multiple refs", []string{"a/b/c", "d/e/f", "g/h/i/j", "k/l/m/n/o"}, "df22ac3ac9e59ed300f70b9c2fdd7128be064652839a813948ee9fd1a2f36581"),
			Entry("single ref", []string{"a/b/c"}, "cbd2be7b96f770a0326948ebd158cf539fab0627e8adbddc97f7a65c6a8ae59a"),
			Entry("no refs", []string{}, emptyListHash),
			Entry("no refs (nil)", nil, emptyListHash),
		)
	})
})

func TestRegistry(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "fbutil Suite")
}
