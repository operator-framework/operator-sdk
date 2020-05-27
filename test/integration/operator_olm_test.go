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
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	apimanifests "github.com/operator-framework/api/pkg/manifests"
	"github.com/operator-framework/operator-sdk/internal/olm"
	operator "github.com/operator-framework/operator-sdk/internal/olm/operator"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"

	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/stretchr/testify/assert"
	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
)

const (
	defaultTimeout = 2 * time.Minute

	defaultOperatorName    = "memcached-operator"
	defaultOperatorVersion = "0.0.2"
)

var (
	kubeconfigPath = os.Getenv(k8sutil.KubeConfigEnvVar)
)

func TestOLMIntegration(t *testing.T) {
	if image, ok := os.LookupEnv(imageEnvVar); ok && image != "" {
		testImageTag = image
	}

	t.Run("PackageManifestsBasic", PackageManifestsBasic)
	t.Run("PackageManifestsAllNamespaces", PackageManifestsAllNamespaces)
	t.Run("PackageManifestsMultiplePackages", PackageManifestsMultiplePackages)
}

func PackageManifestsAllNamespaces(t *testing.T) {

	csvConfig := CSVTemplateConfig{
		OperatorName:    defaultOperatorName,
		OperatorVersion: defaultOperatorVersion,
		TestImageTag:    testImageTag,
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
		InstallModes: []operatorsv1alpha1.InstallMode{
			{Type: operatorsv1alpha1.InstallModeTypeOwnNamespace, Supported: false},
			{Type: operatorsv1alpha1.InstallModeTypeSingleNamespace, Supported: false},
			{Type: operatorsv1alpha1.InstallModeTypeMultiNamespace, Supported: false},
			{Type: operatorsv1alpha1.InstallModeTypeAllNamespaces, Supported: true},
		},
	}
	tmp, cleanup := mkTempDirWithCleanup(t, "")
	defer cleanup()

	channels := []apimanifests.PackageChannel{
		{Name: "alpha", CurrentCSVName: fmt.Sprintf("%s.v%s", defaultOperatorName, defaultOperatorVersion)},
	}
	manifestsDir := filepath.Join(tmp, defaultOperatorName)
	err := writeOperatorManifests(manifestsDir, csvConfig)
	if err != nil {
		os.RemoveAll(tmp)
		t.Fatal(err)
	}
	err = writePackageManifest(manifestsDir, defaultOperatorName, channels)
	if err != nil {
		os.RemoveAll(tmp)
		t.Fatal(err)
	}
	opcmd := operator.PackageManifestsCmd{
		OperatorCmd: operator.OperatorCmd{
			KubeconfigPath: kubeconfigPath,
			Timeout:        defaultTimeout,
			OLMNamespace:   olm.DefaultOLMNamespace,
			InstallMode:    string(operatorsv1alpha1.InstallModeTypeAllNamespaces),
		},
		ManifestsDir:    manifestsDir,
		OperatorVersion: defaultOperatorVersion,
	}
	// Cleanup.
	defer func() {
		opcmd.ForceRegistry = true
		if err := opcmd.Cleanup(); err != nil {
			t.Fatal(err)
		}
	}()

	// Deploy operator.
	assert.NoError(t, opcmd.Run())
}

func PackageManifestsBasic(t *testing.T) {

	csvConfig := CSVTemplateConfig{
		OperatorName:    defaultOperatorName,
		OperatorVersion: defaultOperatorVersion,
		TestImageTag:    testImageTag,
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
		InstallModes: []operatorsv1alpha1.InstallMode{
			{Type: operatorsv1alpha1.InstallModeTypeOwnNamespace, Supported: true},
			{Type: operatorsv1alpha1.InstallModeTypeSingleNamespace, Supported: false},
			{Type: operatorsv1alpha1.InstallModeTypeMultiNamespace, Supported: false},
			{Type: operatorsv1alpha1.InstallModeTypeAllNamespaces, Supported: false},
		},
	}
	tmp, cleanup := mkTempDirWithCleanup(t, "")
	defer cleanup()

	channels := []apimanifests.PackageChannel{
		{Name: "alpha", CurrentCSVName: fmt.Sprintf("%s.v%s", defaultOperatorName, defaultOperatorVersion)},
	}
	manifestsDir := filepath.Join(tmp, defaultOperatorName)
	err := writeOperatorManifests(manifestsDir, csvConfig)
	if err != nil {
		os.RemoveAll(tmp)
		t.Fatal(err)
	}
	err = writePackageManifest(manifestsDir, defaultOperatorName, channels)
	if err != nil {
		os.RemoveAll(tmp)
		t.Fatal(err)
	}
	opcmd := operator.PackageManifestsCmd{
		OperatorCmd: operator.OperatorCmd{
			KubeconfigPath: kubeconfigPath,
			Timeout:        defaultTimeout,
			OLMNamespace:   olm.DefaultOLMNamespace,
		},
		ManifestsDir:    manifestsDir,
		OperatorVersion: defaultOperatorVersion,
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

	// "Deploy operator"
	assert.NoError(t, opcmd.Run())
	// "Fail to deploy operator after deploy"
	assert.Error(t, opcmd.Run())

	// "Remove operator after deploy"
	assert.NoError(t, opcmd.Cleanup())
	// "Remove operator after removal"
	assert.NoError(t, opcmd.Cleanup())
}

func PackageManifestsMultiplePackages(t *testing.T) {

	operatorVersion1 := defaultOperatorVersion
	operatorVersion2 := "0.0.3"
	csvConfigs := []CSVTemplateConfig{
		{
			OperatorName:    defaultOperatorName,
			OperatorVersion: operatorVersion1,
			TestImageTag:    testImageTag,
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
			InstallModes: []operatorsv1alpha1.InstallMode{
				{Type: operatorsv1alpha1.InstallModeTypeOwnNamespace, Supported: true},
				{Type: operatorsv1alpha1.InstallModeTypeSingleNamespace, Supported: false},
				{Type: operatorsv1alpha1.InstallModeTypeMultiNamespace, Supported: false},
				{Type: operatorsv1alpha1.InstallModeTypeAllNamespaces, Supported: false},
			},
		},
		{
			OperatorName:    defaultOperatorName,
			OperatorVersion: operatorVersion2,
			TestImageTag:    testImageTag,
			ReplacesCSVName: fmt.Sprintf("%s.v%s", defaultOperatorName, operatorVersion1),
			CRDKeys: []DefinitionKey{
				{
					Kind:  "Memcached",
					Name:  "memcacheds.cache.example.com",
					Group: "cache.example.com",
					Versions: []apiextv1beta1.CustomResourceDefinitionVersion{
						{Name: "v1alpha1", Storage: false, Served: true},
						{Name: "v1alpha2", Storage: true, Served: true},
					},
				},
			},
			InstallModes: []operatorsv1alpha1.InstallMode{
				{Type: operatorsv1alpha1.InstallModeTypeOwnNamespace, Supported: true},
				{Type: operatorsv1alpha1.InstallModeTypeSingleNamespace, Supported: false},
				{Type: operatorsv1alpha1.InstallModeTypeMultiNamespace, Supported: false},
				{Type: operatorsv1alpha1.InstallModeTypeAllNamespaces, Supported: false},
			},
		},
	}

	tmp, cleanup := mkTempDirWithCleanup(t, "")
	defer cleanup()

	channels := []apimanifests.PackageChannel{
		{Name: "stable", CurrentCSVName: fmt.Sprintf("%s.v%s", defaultOperatorName, operatorVersion2)},
		{Name: "alpha", CurrentCSVName: fmt.Sprintf("%s.v%s", defaultOperatorName, operatorVersion1)},
	}
	manifestsDir := filepath.Join(tmp, defaultOperatorName)
	for _, config := range csvConfigs {
		err := writeOperatorManifests(manifestsDir, config)
		if err != nil {
			os.RemoveAll(tmp)
			t.Fatal(err)
		}
	}
	err := writePackageManifest(manifestsDir, defaultOperatorName, channels)
	if err != nil {
		os.RemoveAll(tmp)
		t.Fatal(err)
	}
	opcmd := operator.PackageManifestsCmd{
		OperatorCmd: operator.OperatorCmd{
			KubeconfigPath: kubeconfigPath,
			Timeout:        defaultTimeout,
			OLMNamespace:   olm.DefaultOLMNamespace,
		},
		ManifestsDir:    manifestsDir,
		OperatorVersion: operatorVersion2,
	}
	// Cleanup.
	defer func() {
		opcmd.ForceRegistry = true
		if err := opcmd.Cleanup(); err != nil {
			t.Fatal(err)
		}
	}()

	// "Deploy operator"
	assert.NoError(t, opcmd.Run())
	// "Remove operator after deploy"
	assert.NoError(t, opcmd.Cleanup())
}
