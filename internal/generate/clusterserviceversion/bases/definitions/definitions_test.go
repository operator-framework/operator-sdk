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

package definitions

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// TODO(estroz): migrate to ginkgo/gomega

var (
	testDataDir = filepath.Join("..", "..", "..", "testdata", "go")
)

func TestApplyDefinitionsForKeysGo(t *testing.T) {

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(testDataDir); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err = os.Chdir(wd); err != nil {
			t.Fatal(err)
		}
	}()

	//nolint:dupl
	cases := []struct {
		description  string
		apisDir      string
		csv          *v1alpha1.ClusterServiceVersion
		gvks         []schema.GroupVersionKind
		expectedCRDs v1alpha1.CustomResourceDefinitions
		wantErr      bool
	}{
		{
			description: "Populate CRDDescription successfully",
			apisDir:     "api",
			csv:         &v1alpha1.ClusterServiceVersion{},
			gvks: []schema.GroupVersionKind{
				{Group: "cache.example.com", Version: "v1alpha2", Kind: "Dummy"},
			},
			expectedCRDs: v1alpha1.CustomResourceDefinitions{
				Owned: []v1alpha1.CRDDescription{
					{
						Name:        "dummys.cache.example.com",
						Kind:        "Dummy",
						Version:     "v1alpha2",
						DisplayName: "Dummy App",
						Description: "Dummy is the Schema for the dummy API",
						Resources: []v1alpha1.APIResourceReference{
							{Name: "dummy-deployment", Kind: "Deployment", Version: "v1"},
							{Name: "dummy-pod", Kind: "Pod", Version: "v1"},
							{Name: "dummy-replicaset", Kind: "ReplicaSet", Version: "v1beta2"},
						},
						SpecDescriptors: []v1alpha1.SpecDescriptor{
							{Path: "size", DisplayName: "dummy-size", Description: "Should be in spec",
								XDescriptors: []string{"urn:alm:descriptor:com.tectonic.ui:podCount"}},
							{Path: "wheels", DisplayName: "Wheels",
								Description:  "Should be in spec, but should not have array index in path",
								XDescriptors: []string{"urn:alm:descriptor:com.tectonic.ui:text"}},
							{Path: "wheels[0].type", DisplayName: "Wheel Type",
								Description: "Type should be in spec with path equal to wheels[0].type",
								XDescriptors: []string{
									"urn:alm:descriptor:com.tectonic.ui:arrayFieldGroup:wheels",
									"urn:alm:descriptor:com.tectonic.ui:text",
								}},
						},
						StatusDescriptors: []v1alpha1.StatusDescriptor{
							{Path: "hog.engine", DisplayName: "boss-hog-engine", XDescriptors: []string{},
								Description: "Should be in status but not spec, since Hog isn't in DummySpec"},
							{Path: "hog.foo", DisplayName: "Public", XDescriptors: []string{}},
							{Path: "hog.seatMaterial", DisplayName: "Seat Material", XDescriptors: []string{}},
							{Path: "hog.seatMaterial", DisplayName: "Seat Material", XDescriptors: []string{}},
							{Path: "nodes", DisplayName: "Nodes", XDescriptors: []string{},
								Description: "Should be in status but not spec, since DummyStatus isn't in DummySpec"},
						},
					},
				},
			},
		},
		{
			description: "Populate CRDDescription with non-standard spec type successfully",
			apisDir:     "api",
			csv:         &v1alpha1.ClusterServiceVersion{},
			gvks: []schema.GroupVersionKind{
				{Group: "cache.example.com", Version: "v1alpha2", Kind: "OtherDummy"},
			},
			expectedCRDs: v1alpha1.CustomResourceDefinitions{
				Owned: []v1alpha1.CRDDescription{
					{
						Name:        "otherdummies.cache.example.com",
						Kind:        "OtherDummy",
						Version:     "v1alpha2",
						DisplayName: "Other Dummy App",
						Description: "OtherDummy is the Schema for the other dummy API",
						Resources: []v1alpha1.APIResourceReference{
							{Name: "other-dummy-pod", Kind: "Pod", Version: "v1"},
							{Name: "other-dummy-service", Kind: "Service", Version: "v1"},
						},
						SpecDescriptors: []v1alpha1.SpecDescriptor{
							{Path: "engine", DisplayName: "Engine", XDescriptors: []string{},
								Description: "Should be in status but not spec, since Hog isn't in DummySpec"},
							{Path: "foo", DisplayName: "Public", XDescriptors: []string{}},
							{Path: "seatMaterial", DisplayName: "Seat Material", XDescriptors: []string{}},
							{Path: "seatMaterial", DisplayName: "Seat Material", XDescriptors: []string{}},
						},
						StatusDescriptors: []v1alpha1.StatusDescriptor{
							{Path: "nothing", DisplayName: "Nothing", XDescriptors: []string{},
								Description: "Should be in status but not spec, since this isn't a spec type"},
						},
					},
				},
			},
		},
		{
			description: "Do not change definitions with non-existent package dir",
			apisDir:     filepath.Join("pkg", "notexist"),
			csv: &v1alpha1.ClusterServiceVersion{
				Spec: v1alpha1.ClusterServiceVersionSpec{
					CustomResourceDefinitions: v1alpha1.CustomResourceDefinitions{
						Owned: []v1alpha1.CRDDescription{
							{
								Name: "dummys.cache.example.com", Version: "v1alpha2", Kind: "Dummy",
								DisplayName: "Dummy App",
								Description: "Dummy is the Schema for the other dummy API",
								Resources: []v1alpha1.APIResourceReference{
									{Name: "dummy-pod", Kind: "Pod", Version: "v1"},
								},
								SpecDescriptors: []v1alpha1.SpecDescriptor{
									{Path: "foo", DisplayName: "Foo", XDescriptors: []string{},
										Description: "Should not be removed"},
								},
								StatusDescriptors: []v1alpha1.StatusDescriptor{
									{Path: "bar", DisplayName: "Bar", XDescriptors: []string{},
										Description: "Should not be removed"},
								},
							},
						},
					},
				},
			},
			gvks: []schema.GroupVersionKind{
				{Group: "cache.example.com", Version: "v1alpha2", Kind: "Dummy"},
			},
			expectedCRDs: v1alpha1.CustomResourceDefinitions{
				Owned: []v1alpha1.CRDDescription{
					{
						Name: "dummys.cache.example.com", Version: "v1alpha2", Kind: "Dummy",
						DisplayName: "Dummy App",
						Description: "Dummy is the Schema for the other dummy API",
						Resources: []v1alpha1.APIResourceReference{
							{Name: "dummy-pod", Kind: "Pod", Version: "v1"},
						},
						SpecDescriptors: []v1alpha1.SpecDescriptor{
							{Path: "foo", DisplayName: "Foo", XDescriptors: []string{},
								Description: "Should not be removed"},
						},
						StatusDescriptors: []v1alpha1.StatusDescriptor{
							{Path: "bar", DisplayName: "Bar", XDescriptors: []string{},
								Description: "Should not be removed"},
						},
					},
				},
			},
		},
		{
			description: "Do not change definitions with non-existent type",
			apisDir:     "api",
			csv: &v1alpha1.ClusterServiceVersion{
				Spec: v1alpha1.ClusterServiceVersionSpec{
					CustomResourceDefinitions: v1alpha1.CustomResourceDefinitions{
						Owned: []v1alpha1.CRDDescription{
							{
								Name: "nokinds.cache.example.com", Version: "v1alpha2", Kind: "NoKind",
								DisplayName: "NoKind App",
								Description: "NoKind is the Schema for the other nokind API",
								Resources: []v1alpha1.APIResourceReference{
									{Name: "no-kind-pod", Kind: "Pod", Version: "v1"},
								},
								SpecDescriptors: []v1alpha1.SpecDescriptor{
									{Path: "foo", DisplayName: "Foo", XDescriptors: []string{},
										Description: "Should not be removed"},
								},
								StatusDescriptors: []v1alpha1.StatusDescriptor{
									{Path: "bar", DisplayName: "Bar", XDescriptors: []string{},
										Description: "Should not be removed"},
								},
							},
						},
					},
				},
			},
			gvks: []schema.GroupVersionKind{
				{Group: "cache.example.com", Version: "v1alpha2", Kind: "NoKind"},
			},
			expectedCRDs: v1alpha1.CustomResourceDefinitions{
				Owned: []v1alpha1.CRDDescription{
					{
						Name: "nokinds.cache.example.com", Version: "v1alpha2", Kind: "NoKind",
						DisplayName: "NoKind App",
						Description: "NoKind is the Schema for the other nokind API",
						Resources: []v1alpha1.APIResourceReference{
							{Name: "no-kind-pod", Kind: "Pod", Version: "v1"},
						},
						SpecDescriptors: []v1alpha1.SpecDescriptor{
							{Path: "foo", DisplayName: "Foo", XDescriptors: []string{},
								Description: "Should not be removed"},
						},
						StatusDescriptors: []v1alpha1.StatusDescriptor{
							{Path: "bar", DisplayName: "Bar", XDescriptors: []string{},
								Description: "Should not be removed"},
						},
					},
				},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.description, func(t *testing.T) {
			err := ApplyDefinitionsForKeysGo(c.csv, c.apisDir, c.gvks)
			if !c.wantErr && err != nil {
				t.Errorf("Expected nil error, got %q", err)
			} else if c.wantErr && err == nil {
				t.Errorf("Expected non-nil error, got nil error")
			} else if !c.wantErr && err == nil {
				assert.Equal(t, c.expectedCRDs, c.csv.Spec.CustomResourceDefinitions)
			}
		})
	}
}
