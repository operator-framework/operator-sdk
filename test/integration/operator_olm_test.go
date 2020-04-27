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

	"github.com/operator-framework/operator-registry/pkg/registry"
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
		defaultTestImageTag = image
	}
	t.Run("AnnotationsBasic", OperatorAnnotationsBasic)
	t.Run("AnnotationsAllNamespaces", OperatorAnnotationsAllNamespaces)
	t.Run("PackageManifestBasic", OperatorPackageManifestBasic)
}

func OperatorAnnotationsBasic(t *testing.T) {

	csvConfig := CSVTemplateConfig{
		OperatorName:    defaultOperatorName,
		OperatorVersion: defaultOperatorVersion,
		TestImageTag:    defaultTestImageTag,
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
		IsManifests: true,
	}
	tmp, cleanup := mkTempDirWithCleanup(t, "sdk-integration.")
	defer cleanup()

	channels := []string{"alpha"}
	manifestsDir := filepath.Join(tmp, defaultOperatorName)
	err := writeOperatorManifests(manifestsDir, csvConfig)
	if err != nil {
		os.RemoveAll(tmp)
		t.Fatal(err)
	}
	err = writeAnnotations(manifestsDir, defaultOperatorName, channels)
	if err != nil {
		os.RemoveAll(tmp)
		t.Fatal(err)
	}
	opcmd := operator.OLMCmd{
		ManifestsDir:    manifestsDir,
		OperatorVersion: defaultOperatorVersion,
		KubeconfigPath:  kubeconfigPath,
		Timeout:         defaultTimeout,
		OLMNamespace:    olm.DefaultOLMNamespace,
	}
	// Cleanup.
	defer func() {
		if err := opcmd.Cleanup(); err != nil {
			t.Fatal(err)
		}
	}()

	// Remove operator before deploy.
	assert.NoError(t, opcmd.Cleanup())

	// Deploy operator.
	assert.NoError(t, opcmd.Run())
	// Fail to deploy operator after deploy.
	assert.Error(t, opcmd.Run())

	// Remove operator after deploy.
	assert.NoError(t, opcmd.Cleanup())
	// Remove operator after removal.
	assert.NoError(t, opcmd.Cleanup())
}

func OperatorAnnotationsAllNamespaces(t *testing.T) {

	csvConfig := CSVTemplateConfig{
		OperatorName:    defaultOperatorName,
		OperatorVersion: defaultOperatorVersion,
		TestImageTag:    defaultTestImageTag,
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
		IsManifests: true,
	}
	tmp, cleanup := mkTempDirWithCleanup(t, "sdk-integration.")
	defer cleanup()

	channels := []string{"alpha"}
	manifestsDir := filepath.Join(tmp, defaultOperatorName)
	err := writeOperatorManifests(manifestsDir, csvConfig)
	if err != nil {
		os.RemoveAll(tmp)
		t.Fatal(err)
	}
	err = writeAnnotations(manifestsDir, defaultOperatorName, channels)
	if err != nil {
		os.RemoveAll(tmp)
		t.Fatal(err)
	}
	opcmd := operator.OLMCmd{
		ManifestsDir:    manifestsDir,
		OperatorVersion: defaultOperatorVersion,
		KubeconfigPath:  kubeconfigPath,
		Timeout:         defaultTimeout,
		OLMNamespace:    olm.DefaultOLMNamespace,
		InstallMode:     string(operatorsv1alpha1.InstallModeTypeAllNamespaces),
	}
	// Cleanup.
	defer func() {
		if err := opcmd.Cleanup(); err != nil {
			t.Fatal(err)
		}
	}()

	// Deploy operator.
	assert.NoError(t, opcmd.Run())
}

func OperatorPackageManifestBasic(t *testing.T) {

	csvConfig := CSVTemplateConfig{
		OperatorName:    defaultOperatorName,
		OperatorVersion: defaultOperatorVersion,
		TestImageTag:    defaultTestImageTag,
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
	tmp, cleanup := mkTempDirWithCleanup(t, "sdk-integration.")
	defer cleanup()

	channels := []registry.PackageChannel{
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
	opcmd := operator.OLMCmd{
		ManifestsDir:    manifestsDir,
		OperatorVersion: defaultOperatorVersion,
		KubeconfigPath:  kubeconfigPath,
		Timeout:         defaultTimeout,
		OLMNamespace:    olm.DefaultOLMNamespace,
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
