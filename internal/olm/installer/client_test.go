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

package installer

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("helpers", func() {
	Describe("formatVersion", func() {
		const (
			latest            = "latest"
			zeroSixteen       = "0.16.0"
			vZeroSixteen      = "v" + zeroSixteen
			zeroSixteenOne    = "0.16.1"
			vZeroSixteenOne   = "v" + zeroSixteenOne
			zeroSeventeen     = "0.17.0"
			vZeroSeventeen    = "v" + zeroSeventeen
			zeroSeventeenOne  = "0.17.1"
			vZeroSeventeenOne = "v" + zeroSeventeenOne
			oneTwoThree       = "1.2.3"
			vOneTwoThree      = "v" + oneTwoThree
		)

		It("returns a non semantic version as-is", func() {
			By(fmt.Sprintf("receiving %s", latest))
			Expect(formatVersion(latest)).To(Equal(latest))
		})

		It("returns a v-prepended semantic version", func() {
			By(fmt.Sprintf("receiving %s", zeroSeventeen))
			Expect(formatVersion(zeroSeventeen)).To(Equal(vZeroSeventeen))

			By(fmt.Sprintf("receiving %s", vZeroSeventeen))
			Expect(formatVersion(vZeroSeventeen)).To(Equal(vZeroSeventeen))

			By(fmt.Sprintf("receiving %s", zeroSeventeenOne))
			Expect(formatVersion(zeroSeventeenOne)).To(Equal(vZeroSeventeenOne))

			By(fmt.Sprintf("receiving %s", vZeroSeventeenOne))
			Expect(formatVersion(vZeroSeventeenOne)).To(Equal(vZeroSeventeenOne))

			By(fmt.Sprintf("receiving %s", oneTwoThree))
			Expect(formatVersion(oneTwoThree)).To(Equal(vOneTwoThree))

			By(fmt.Sprintf("receiving %s", vOneTwoThree))
			Expect(formatVersion(vOneTwoThree)).To(Equal(vOneTwoThree))
		})

		It("returns a format semantic version", func() {
			By(fmt.Sprintf("receiving %s", zeroSixteen))
			Expect(formatVersion(zeroSixteen)).To(Equal(zeroSixteen))

			By(fmt.Sprintf("receiving %s", vZeroSixteen))
			Expect(formatVersion(vZeroSixteen)).To(Equal(zeroSixteen))

			By(fmt.Sprintf("receiving %s", zeroSixteenOne))
			Expect(formatVersion(zeroSixteenOne)).To(Equal(zeroSixteenOne))

			By(fmt.Sprintf("receiving %s", vZeroSixteenOne))
			Expect(formatVersion(vZeroSixteenOne)).To(Equal(zeroSixteenOne))
		})

	})
})
