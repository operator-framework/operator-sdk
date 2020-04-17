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
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
)

func TestAddTo(t *testing.T) {
	testCases := []struct {
		name     string
		apiFlags APIFlags
		validate func(APIFlags, *pflag.FlagSet) error
	}{
		{
			// Populate FlagSet
			name: "Populate FlagSet",
			apiFlags: APIFlags{
				APIVersion:       "app.example.com/v1alpha1",
				Kind:             "AppService",
				SkipGeneration:   false,
				CrdVersion:       "v1",
				HelmChartRef:     "stable/app",
				HelmChartVersion: "1.0.0",
				HelmChartRepo:    "https://charts.mycompany.com/",
			},
			validate: func(apiFlags APIFlags, flagSet *pflag.FlagSet) error {
				val, err := flagSet.GetString("api-version")
				if err != nil {
					return err
				}
				if apiFlags.APIVersion != val {
					return fmt.Errorf("apiVersion does not match")
				}

				val, err = flagSet.GetString("kind")
				if err != nil {
					return err
				}
				if apiFlags.Kind != val {
					return fmt.Errorf("kind does not match")
				}

				boolVal, err := flagSet.GetBool("skip-generation")
				if err != nil {
					return err
				}
				if apiFlags.SkipGeneration != boolVal {
					return fmt.Errorf("skipGeneration does not match")
				}

				val, err = flagSet.GetString("crd-version")
				if err != nil {
					return err
				}
				if apiFlags.CrdVersion != val {
					return fmt.Errorf("crdVersion does not match")
				}

				val, err = flagSet.GetString("helm-chart-repo")
				if err != nil {
					return err
				}
				if apiFlags.HelmChartRepo != val {
					return fmt.Errorf("helmChartRepo does not match")
				}

				val, err = flagSet.GetString("helm-chart-version")
				if err != nil {
					return err
				}
				if apiFlags.HelmChartVersion != val {
					return fmt.Errorf("helmChartVersion does not match")
				}

				return nil
			},
		},
	}

	for _, tc := range testCases {
		expFlags := &pflag.FlagSet{}
		t.Run(tc.name, func(t *testing.T) {
			tc.apiFlags.AddTo(expFlags)
			if tc.validate != nil {
				if err := tc.validate(tc.apiFlags, expFlags); err != nil {
					t.Fatal("Unexpected error validating AddTo", err)
				}
			}
		})
	}
}

func TestVerifyCommonFlags(t *testing.T) {
	testCases := []struct {
		name         string
		apiFlags     APIFlags
		operatorType string
		expError     string
	}{
		{
			// Valid Go API Flags
			name: "Valid Go API Flags",
			apiFlags: APIFlags{
				APIVersion:     "app.example.com/v1alpha1",
				Kind:           "AppService",
				SkipGeneration: false,
			},
			operatorType: "go",
			expError:     "",
		},
		{
			// Invalid Go API Flags
			name: "Invalid Go API Flags-HelmChartRef",
			apiFlags: APIFlags{
				APIVersion:   "app.example.com/v1alpha1",
				HelmChartRef: "stable/repo",
			},
			operatorType: "go",
			expError:     "value of --helm-chart can only be used with --type=helm",
		},
		{
			// Invalid Go API Flags
			name: "Invalid Go API Flags-HelmChartRepo",
			apiFlags: APIFlags{
				APIVersion:    "app.example.com/v1alpha1",
				HelmChartRepo: "https://charts.mycompany.com/",
			},
			operatorType: "go",
			expError:     "value of --helm-chart-repo can only be used with --type=helm and --helm-chart",
		},
		{
			// Invalid Go API Flags
			name: "Invalid Go API Flags-HelmChartVersion",
			apiFlags: APIFlags{
				APIVersion:       "app.example.com/v1alpha1",
				HelmChartVersion: "1.2.0",
			},
			operatorType: "go",
			expError:     "value of --helm-chart-version can only be used with --type=helm and --helm-chart",
		},
		{
			// Valid Ansible API Flags
			name: "Valid Ansible API Flags",
			apiFlags: APIFlags{
				APIVersion: "app.example.com/v1alpha1",
				Kind:       "App",
				CrdVersion: "v1",
			},
			operatorType: "ansible",
			expError:     "",
		},
		{
			// Valid Ansible API Flags
			name: "Valid Ansible API Flags-check dup",
			apiFlags: APIFlags{
				APIVersion: "app.example.com/v1alpha1",
				Kind:       "App",
			},
			operatorType: "ansible",
			expError:     "",
		},
		{
			// Invalid Ansible API Flags
			name: "Invalid Ansible API Flags-Kind not present",
			apiFlags: APIFlags{
				APIVersion: "app.example.com/v1alpha1",
			},
			operatorType: "ansible",
			expError:     "value of --kind must not have empty value",
		},
		{
			// Invalid Ansible API Flags
			name: "Invalid Ansible API Flags-apiVersion not present",
			apiFlags: APIFlags{
				Kind: "App",
			},
			operatorType: "ansible",
			expError:     "value of --api-version must not have empty value",
		},
		{
			// Invalid Ansible API Flags
			name: "Invalid Ansible API Flags-HelmChartVersion is used",
			apiFlags: APIFlags{
				APIVersion:       "app.example.com/v1alpha1",
				Kind:             "App",
				HelmChartVersion: "1.2.0",
			},
			operatorType: "ansible",
			expError:     "value of --helm-chart-version can only be used with --type=helm and --helm-chart",
		},
		{
			// Invalid Ansible API Flags
			name: "Invalid Ansible API Flags-HelmChartRepo is used",
			apiFlags: APIFlags{
				APIVersion:    "app.example.com/v1alpha1",
				Kind:          "App",
				HelmChartRepo: "https://charts.mycompany.com/",
			},
			operatorType: "ansible",
			expError:     "value of --helm-chart-repo can only be used with --type=helm and --helm-chart",
		},
		{
			// Valid HELM API Flags
			name: "Valid HELM API Flags",
			apiFlags: APIFlags{
				APIVersion:     "app.example.com/v1alpha1",
				Kind:           "App",
				SkipGeneration: true,
			},
			operatorType: "helm",
			expError:     "",
		},
		{
			// Valid HELM API Flags
			name: "Valid HELM API Flags-Helmchart used",
			apiFlags: APIFlags{
				HelmChartRef: "stable/repo",
			},
			operatorType: "helm",
			expError:     "",
		},
		{
			// Valid HELM API Flags
			name: "Valid HELM API Flags-Helm specific flags",
			apiFlags: APIFlags{
				HelmChartRef:     "stable/repo",
				HelmChartRepo:    "https://charts.mycompany.com/",
				HelmChartVersion: "1.2.0",
			},
			operatorType: "helm",
			expError:     "",
		},
		{
			// Invalid HELM API Flags
			name: "Invalid HELM API Flags-no HelmChartRef provided",
			apiFlags: APIFlags{
				HelmChartRepo:    "https://charts.mycompany.com/",
				HelmChartVersion: "1.2.0",
			},
			operatorType: "helm",
			expError:     "value of --helm-chart-repo can only be used with --type=helm and --helm-chart",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var result string
			if err := tc.apiFlags.VerifyCommonFlags(tc.operatorType); err != nil {
				result = err.Error()
			}
			assert.Equal(t, tc.expError, result)
		})
	}
}
