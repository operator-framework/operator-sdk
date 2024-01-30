// Copyright 2021 The Operator-SDK Authors
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

package flags_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/pflag"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/operator-framework/operator-sdk/internal/helm/flags"
)

var _ = Describe("Flags", func() {
	Describe("ToManagerOptions", func() {
		var (
			f       *flags.Flags
			flagSet *pflag.FlagSet
			options manager.Options
		)
		BeforeEach(func() {
			f = &flags.Flags{}
			flagSet = pflag.NewFlagSet("test", pflag.ExitOnError)
			f.AddTo(flagSet)
		})

		When("the flag is set", func() {
			It("uses the flag value when corresponding option value is empty", func() {
				expOptionValue := ":5678"
				options.Metrics.BindAddress = ""
				parseArgs(flagSet, "--metrics-bind-address", expOptionValue)
				Expect(f.ToManagerOptions(options).Metrics.BindAddress).To(Equal(expOptionValue))
			})
			It("uses the flag value when corresponding option value is not empty", func() {
				expOptionValue := ":5678"
				options.Metrics.BindAddress = ":1234"
				parseArgs(flagSet, "--metrics-bind-address", expOptionValue)
				Expect(f.ToManagerOptions(options).Metrics.BindAddress).To(Equal(expOptionValue))
			})
		})
		When("the flag is not set", func() {
			It("uses the default flag value when corresponding option value is empty", func() {
				expOptionValue := ":8080"
				options.Metrics.BindAddress = ""
				parseArgs(flagSet)
				Expect(f.ToManagerOptions(options).Metrics.BindAddress).To(Equal(expOptionValue))
			})
			It("uses the option value when corresponding option value is not empty", func() {
				expOptionValue := ":1234"
				options.Metrics.BindAddress = expOptionValue
				parseArgs(flagSet)
				Expect(f.ToManagerOptions(options).Metrics.BindAddress).To(Equal(expOptionValue))
			})
		})
	})
})

func parseArgs(fs *pflag.FlagSet, extraArgs ...string) {
	Expect(fs.Parse(extraArgs)).To(Succeed())
}
