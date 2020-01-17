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
	"time"

	operator "github.com/operator-framework/operator-sdk/internal/olm/operator"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"

	opv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"github.com/stretchr/testify/assert"
	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
)

const (
	defaultTimeout = 2 * time.Minute
)

var (
	kubeconfigPath = os.Getenv(k8sutil.KubeConfigEnvVar)
)

func TestOLMIntegration(t *testing.T) {
	if image, ok := os.LookupEnv(imageEnvVar); ok && image != "" {
		defaultTestImageTag = image
	}
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
				Versions: []apiextv1beta1.CustomResourceDefinitionVersion{
					{Name: "v1alpha1", Storage: true, Served: true},
				},
			},
		},
		InstallModes: []opv1alpha1.InstallMode{
			{Type: opv1alpha1.InstallModeTypeOwnNamespace, Supported: true},
			{Type: opv1alpha1.InstallModeTypeSingleNamespace, Supported: true},
			{Type: opv1alpha1.InstallModeTypeMultiNamespace, Supported: false},
			{Type: opv1alpha1.InstallModeTypeAllNamespaces, Supported: true},
		},
	}
	tmp, err := ioutil.TempDir("", "sdk-integration.")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.RemoveAll(tmp); err != nil {
			t.Fatal(err)
		}
	}()
	defaultChannel := "alpha"
	operatorName := "memcached-operator"
	operatorVersion := "0.0.2"
	manifestsDir, err := writeOperatorManifests(tmp, operatorName, defaultChannel, csvConfig)
	if err != nil {
		os.RemoveAll(tmp)
		t.Fatal(err)
	}
	opcmd := operator.OLMCmd{
		ManifestsDir:    manifestsDir,
		OperatorVersion: operatorVersion,
		KubeconfigPath:  kubeconfigPath,
		Timeout:         defaultTimeout,
	}
	// Cleanup.
	defer func() {
		opcmd.ForceRegistry = true
		if err := opcmd.Cleanup(); err != nil {
			t.Fatal(err)
		}
	}()

	// "Remove operator before deploy"
	assert.NoError(t, opcmd.Cleanup())
	// "Remove operator before deploy (force delete registry)"
	opcmd.ForceRegistry = true
	assert.NoError(t, opcmd.Cleanup())

	// "Deploy operator"
	assert.NoError(t, opcmd.Run())
	// "Fail to deploy operator after deploy"
	assert.Error(t, opcmd.Run())

	// "Remove operator after deploy"
	assert.NoError(t, opcmd.Cleanup())
	// "Remove operator after removal"
	assert.NoError(t, opcmd.Cleanup())
	// "Remove operator after removal (force delete registry)"
	opcmd.ForceRegistry = true
	assert.NoError(t, opcmd.Cleanup())
}
