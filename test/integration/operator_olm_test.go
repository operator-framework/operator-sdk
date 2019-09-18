// Copyright 2018 The Operator-SDK Authors
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

package e2e

import (
	"io/ioutil"
	"os"
	"testing"

	operator "github.com/operator-framework/operator-sdk/internal/olm/operator"

	"github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
)

func TestOLMIntegration(t *testing.T) {
	t.Run("Operator", func(t *testing.T) {
		t.Run("Single", SingleOperator)
	})
}

func SingleOperator(t *testing.T) {
	csvConfig := CSVTemplateConfig{
		OperatorName:    "memcached-operator",
		OperatorVersion: "0.0.2",
		TestImageTag:    defaultTestImageTag,
		Maturity:        "alpha",
		ReplacesCSVName: "",
		CRDKeys: []DefinitionKey{
			{
				Kind:  "Memcached",
				Name:  "memcacheds.cache.example.com",
				Group: "cache.example.com",
				Versions: []v1beta1.CustomResourceDefinitionVersion{
					{Name: "v1alpha1", Storage: true, Served: true},
				},
			},
		},
		InstallModes: []v1alpha1.InstallMode{
			{Type: v1alpha1.InstallModeTypeOwnNamespace, Supported: true},
			{Type: v1alpha1.InstallModeTypeSingleNamespace, Supported: true},
			{Type: v1alpha1.InstallModeTypeMultiNamespace, Supported: false},
			{Type: v1alpha1.InstallModeTypeAllNamespaces, Supported: true},
		},
	}
	tmp, err := ioutil.TempDir("", "sdk-integration.")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmp)
	defaultChannel := "alpha"
	operatorName := "memcached-operator"
	operatorVersion := "0.0.2"
	manifestsDir, err := writeOperatorManifests(tmp, operatorName, defaultChannel, csvConfig)
	if err != nil {
		t.Fatal(err)
	}
	opcmd := operator.OLMCmd{
		ManifestsDir:    manifestsDir,
		OperatorVersion: operatorVersion,
		KubeconfigPath:  KubeconfigPath,
	}
	cases := []struct {
		description string
		op          func() error
		force       bool
		wantErr     bool
	}{
		{"Remove operator before deploy", opcmd.Down, false, true},
		{"Deploy operator", opcmd.Up, false, false},
		{"Deploy operator after deploy", opcmd.Up, false, true},
		{"Deploy operator after deploy with force", opcmd.Up, true, false},
		{"Remove operator after deploy", opcmd.Down, false, false},
		{"Remove operator after removal", opcmd.Down, false, true},
		{"Remove operator after removal with force", opcmd.Down, true, false},
	}

	for _, c := range cases {
		opcmd.Force = c.force
		err := c.op()
		if c.wantErr && err == nil {
			t.Fatalf("%s (%s): wanted error, got: nil", c.description, operatorName)
		} else if !c.wantErr && err != nil {
			t.Fatalf("%s (%s): wanted no error, got: %v", c.description, operatorName, err)
		}
	}
}
