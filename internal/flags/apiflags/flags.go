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

package apiflags

import (
	"fmt"
	"strings"

	"github.com/spf13/pflag"

	gencrd "github.com/operator-framework/operator-sdk/internal/generate/crd"
)

type APIFlags struct {
	SkipGeneration bool
	APIVersion     string
	Kind           string
	CrdVersion     string
}

// AddTo - Add the reconcile period and watches file flags to the the flagset
// helpTextPrefix will allow you add a prefix to default help text. Joined by a space.
func (f *APIFlags) AddTo(flagSet *pflag.FlagSet) {
	flagSet.StringVar(&f.APIVersion, "api-version", "",
		"Kubernetes apiVersion and has a format of $GROUP_NAME/$VERSION (e.g app.example.com/v1alpha1)")

	flagSet.StringVar(&f.Kind, "kind", "",
		"Kubernetes resource Kind name. (e.g AppService)")

	flagSet.BoolVar(&f.SkipGeneration, "skip-generation", false,
		"Skip generation of deepcopy and OpenAPI code and OpenAPI CRD specs")

	flagSet.StringVar(&f.CrdVersion, "crd-version", gencrd.DefaultCRDVersion,
		"CRD version to generate")

}

// VerifyCommonFlags func is used to verify flags common to both "new" and "add api" commands.
func (f *APIFlags) VerifyCommonFlags(operatorType string) error {
	if len(f.APIVersion) == 0 {
		return fmt.Errorf("value of --api-version must not have empty value")
	}
	if len(f.Kind) == 0 {
		return fmt.Errorf("value of --kind must not have empty value")
	}
	kindFirstLetter := string(f.Kind[0])
	if kindFirstLetter != strings.ToUpper(kindFirstLetter) {
		return fmt.Errorf("value of --kind must start with an uppercase letter")
	}
	if strings.Count(f.APIVersion, "/") != 1 {
		return fmt.Errorf("value of --api-version has wrong format (%v);"+
			" format must be $GROUP_NAME/$VERSION (e.g app.example.com/v1alpha1)", f.APIVersion)
	}
	return nil
}
