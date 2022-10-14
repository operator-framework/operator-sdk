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
	"sort"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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
							{Path: "sideCar", DisplayName: "Side Car"},
							{Path: "size", DisplayName: "dummy-size", Description: "Should be in spec",
								XDescriptors: []string{"urn:alm:descriptor:com.tectonic.ui:podCount"}},
							{Path: "useful.containers", DisplayName: "Containers"},
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
							{Path: "hog.engine", DisplayName: "boss-hog-engine", Description: "Should be in status but not spec, since Hog isn't in DummySpec"},
							{Path: "hog.foo", DisplayName: "Public"},
							{Path: "hog.seatMaterial", DisplayName: "Seat Material"},
							{Path: "hog.seatMaterial", DisplayName: "Seat Material"},
							{Path: "nodes", DisplayName: "Nodes", Description: "Should be in status but not spec, since DummyStatus isn't in DummySpec"},
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
							{Path: "engine", DisplayName: "Engine", Description: "Should be in status but not spec, since Hog isn't in DummySpec"},
							{Path: "foo", DisplayName: "Public"},
							{Path: "seatMaterial", DisplayName: "Seat Material"},
							{Path: "seatMaterial", DisplayName: "Seat Material"},
						},
						StatusDescriptors: []v1alpha1.StatusDescriptor{
							{Path: "nothing", DisplayName: "Nothing", Description: "Should be in status but not spec, since this isn't a spec type"},
						},
					},
				},
			},
		},
		{
			description: "Populate CRDDescription with GVKs with same GK and different versions",
			apisDir:     "api",
			csv:         &v1alpha1.ClusterServiceVersion{},
			gvks: []schema.GroupVersionKind{
				{Group: "cache.example.com", Version: "v1alpha1", Kind: "Memcached"},
				{Group: "cache.example.com", Version: "v1alpha2", Kind: "Memcached"},
			},
			expectedCRDs: v1alpha1.CustomResourceDefinitions{
				Owned: []v1alpha1.CRDDescription{
					{
						Name: "memcacheds.cache.example.com", Version: "v1alpha2", Kind: "Memcached",
						DisplayName: "Memcached App",
						Description: "Memcached is the Schema for the memcacheds API",
						SpecDescriptors: []v1alpha1.SpecDescriptor{
							{Path: "size", DisplayName: "Size", Description: "Size is the size of the memcached deployment"},
						},
						StatusDescriptors: []v1alpha1.StatusDescriptor{
							{Path: "nodes", DisplayName: "Nodes", Description: "Nodes are the names of the memcached pods"},
						},
					},
					{
						Name: "memcacheds.cache.example.com", Version: "v1alpha1", Kind: "Memcached",
						DisplayName: "Memcached App Display Name",
						Description: "Memcached is the Schema for the memcacheds API",
						StatusDescriptors: []v1alpha1.StatusDescriptor{
							{Path: "nodes", DisplayName: "Nodes", Description: "Nodes are the names of the memcached pods"},
						},
						SpecDescriptors: []v1alpha1.SpecDescriptor{
							{Path: "containers", DisplayName: "Containers"},
							{Path: "providers", DisplayName: "Providers", Description: "List of Providers"},
							{Path: "providers[0].foo", DisplayName: "Foo Provider", Description: "Foo represents the Foo provider"},
							{Path: "providers[0].foo.credentialsSecret", DisplayName: "Secret Containing the Credentials",
								Description:  "CredentialsSecret is a reference to a secret containing authentication details for the Foo server",
								XDescriptors: []string{"urn:alm:descriptor:io.kubernetes:Secret"},
							},
							{
								Path: "providers[0].foo.credentialsSecret.key", DisplayName: "Key within the secret",
								Description:  "Key represents the specific key to reference from the secret",
								XDescriptors: []string{"urn:alm:descriptor:com.tectonic.ui:advanced", "urn:alm:descriptor:com.tectonic.ui:text"},
							},
							{
								Path: "providers[0].foo.credentialsSecret.name", DisplayName: "Name of the secret",
								Description:  "Name represents the name of the secret",
								XDescriptors: []string{"urn:alm:descriptor:com.tectonic.ui:advanced", "urn:alm:descriptor:com.tectonic.ui:text"},
							},
							{
								Path: "providers[0].foo.credentialsSecret.namespace", DisplayName: "Namespace containing the secret",
								Description:  "Namespace represents the namespace containing the secret",
								XDescriptors: []string{"urn:alm:descriptor:com.tectonic.ui:advanced", "urn:alm:descriptor:com.tectonic.ui:text"},
							},
							{
								Path: "size", DisplayName: "Size", Description: "Size is the size of the memcached deployment"},
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
									{Path: "foo", DisplayName: "Foo", Description: "Should not be removed"},
								},
								StatusDescriptors: []v1alpha1.StatusDescriptor{
									{Path: "bar", DisplayName: "Bar", Description: "Should not be removed"},
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
							{Path: "foo", DisplayName: "Foo", Description: "Should not be removed"},
						},
						StatusDescriptors: []v1alpha1.StatusDescriptor{
							{Path: "bar", DisplayName: "Bar", Description: "Should not be removed"},
						},
					},
				},
			},
		},
		{
			description: "Return the CSV unchanged for non-existent APIs dir",
			apisDir:     filepath.Join("pkg", "notexist"),
			csv: &v1alpha1.ClusterServiceVersion{
				Spec: v1alpha1.ClusterServiceVersionSpec{
					CustomResourceDefinitions: v1alpha1.CustomResourceDefinitions{
						Owned: []v1alpha1.CRDDescription{
							{
								Name: "nokinds.cache.example.com", Version: "v1alpha2", Kind: "NoKind",
								DisplayName: "NoKind App",
								Description: "NoKind is the Schema for the other nokind API",
							},
						},
					},
				},
			},
			expectedCRDs: v1alpha1.CustomResourceDefinitions{
				Owned: []v1alpha1.CRDDescription{
					{
						Name: "nokinds.cache.example.com", Version: "v1alpha2", Kind: "NoKind",
						DisplayName: "NoKind App",
						Description: "NoKind is the Schema for the other nokind API",
					},
				},
			},
			wantErr: false,
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

var _ = Describe("updateDefinitionsByKey", func() {
	var (
		startingCSV *v1alpha1.ClusterServiceVersion
		defsByGVK   map[schema.GroupVersionKind]*descriptionValues
	)

	BeforeEach(func() {
		startingCSV = &v1alpha1.ClusterServiceVersion{}
		defsByGVK = make(map[schema.GroupVersionKind]*descriptionValues)
	})

	It("handles an empty CSV and descriptions without error", func() {
		updateDefinitionsByKey(startingCSV, defsByGVK)
		Expect(startingCSV.Spec.CustomResourceDefinitions.Owned).To(Equal([]v1alpha1.CRDDescription{}))
	})
	It("preserves ordering of two existing descriptions with no new descriptions", func() {
		owned := []v1alpha1.CRDDescription{
			{Name: "memcacheds.cache.example.com", Version: "v1alpha1", Kind: "Memcached"},
			{Name: "memcached2s.cache.example.com", Version: "v1alpha1", Kind: "Memcached2"},
		}
		startingCSV.Spec.CustomResourceDefinitions.Owned = owned
		csv := startingCSV.DeepCopy()
		updateDefinitionsByKey(csv, defsByGVK)
		Expect(csv.Spec.CustomResourceDefinitions.Owned).To(Equal(owned))
	})
	It("orders two unordered existing descriptions with no new descriptions", func() {
		owned := []v1alpha1.CRDDescription{
			{Name: "memcached2s.cache.example.com", Version: "v1alpha1", Kind: "Memcached2"},
			{Name: "memcacheds.cache.example.com", Version: "v1alpha1", Kind: "Memcached"},
		}
		startingCSV.Spec.CustomResourceDefinitions.Owned = owned
		csv := startingCSV.DeepCopy()
		updateDefinitionsByKey(csv, defsByGVK)
		sort.Slice(owned, func(i, j int) bool { return owned[i].Name < owned[j].Name })
		Expect(csv.Spec.CustomResourceDefinitions.Owned).To(Equal(owned))
	})
	It("orders two new descriptions with increasing orders with no existing descriptions", func() {
		desc1 := v1alpha1.CRDDescription{Name: "memcached3s.cache.example.com", Version: "v1alpha1", Kind: "Memcached3"}
		desc2 := v1alpha1.CRDDescription{Name: "memcached4s.cache.example.com", Version: "v1alpha1", Kind: "Memcached4"}
		defsByGVK[descToGVK(desc1)] = &descriptionValues{crdOrder: 0, crd: desc1}
		defsByGVK[descToGVK(desc2)] = &descriptionValues{crdOrder: 1, crd: desc2}

		expected := []v1alpha1.CRDDescription{desc1, desc2}
		updateDefinitionsByKey(startingCSV, defsByGVK)
		Expect(startingCSV.Spec.CustomResourceDefinitions.Owned).To(Equal(expected))
	})
	It("orders two new descriptions with decreasing orders with no existing descriptions", func() {
		desc1 := v1alpha1.CRDDescription{Name: "memcached3s.cache.example.com", Version: "v1alpha1", Kind: "Memcached3"}
		desc2 := v1alpha1.CRDDescription{Name: "memcached4s.cache.example.com", Version: "v1alpha1", Kind: "Memcached4"}
		defsByGVK[descToGVK(desc1)] = &descriptionValues{crdOrder: 1, crd: desc1}
		defsByGVK[descToGVK(desc2)] = &descriptionValues{crdOrder: 0, crd: desc2}

		expected := []v1alpha1.CRDDescription{desc2, desc1}
		updateDefinitionsByKey(startingCSV, defsByGVK)
		Expect(startingCSV.Spec.CustomResourceDefinitions.Owned).To(Equal(expected))
	})
	It("orders two new descriptions with the same order with no existing descriptions", func() {
		desc1 := v1alpha1.CRDDescription{Name: "memcached3s.cache.example.com", Version: "v1alpha1", Kind: "Memcached3"}
		desc2 := v1alpha1.CRDDescription{Name: "memcached4s.cache.example.com", Version: "v1alpha1", Kind: "Memcached4"}
		defsByGVK[descToGVK(desc1)] = &descriptionValues{crdOrder: 0, crd: desc1}
		defsByGVK[descToGVK(desc2)] = &descriptionValues{crdOrder: 0, crd: desc2}

		expected := []v1alpha1.CRDDescription{desc1, desc2}
		updateDefinitionsByKey(startingCSV, defsByGVK)
		Expect(startingCSV.Spec.CustomResourceDefinitions.Owned).To(Equal(expected))
	})

	It("orders one new description after two existing descriptions", func() {
		desc1 := v1alpha1.CRDDescription{Name: "memcached3s.cache.example.com", Version: "v1alpha1", Kind: "Memcached3"}
		defsByGVK[descToGVK(desc1)] = &descriptionValues{crdOrder: 2, crd: desc1}

		owned := []v1alpha1.CRDDescription{
			{Name: "memcached2s.cache.example.com", Version: "v1alpha1", Kind: "Memcached2"},
			{Name: "memcacheds.cache.example.com", Version: "v1alpha1", Kind: "Memcached"},
		}
		expected := []v1alpha1.CRDDescription{owned[0], owned[1], desc1}
		startingCSV.Spec.CustomResourceDefinitions.Owned = owned
		csv := startingCSV.DeepCopy()
		updateDefinitionsByKey(csv, defsByGVK)
		Expect(csv.Spec.CustomResourceDefinitions.Owned).To(Equal(expected))
	})
	It("orders one new description between two existing descriptions", func() {
		desc1 := v1alpha1.CRDDescription{Name: "memcached3s.cache.example.com", Version: "v1alpha1", Kind: "Memcached3"}
		defsByGVK[descToGVK(desc1)] = &descriptionValues{crdOrder: 0, crd: desc1}

		owned := []v1alpha1.CRDDescription{
			{Name: "memcached2s.cache.example.com", Version: "v1alpha1", Kind: "Memcached2"},
			{Name: "memcacheds.cache.example.com", Version: "v1alpha1", Kind: "Memcached"},
		}
		expected := []v1alpha1.CRDDescription{desc1, owned[0], owned[1]}
		startingCSV.Spec.CustomResourceDefinitions.Owned = owned
		csv := startingCSV.DeepCopy()
		updateDefinitionsByKey(csv, defsByGVK)
		Expect(csv.Spec.CustomResourceDefinitions.Owned).To(Equal(expected))
	})
	It("orders two new descriptions with the same order with overlapping orders of two existing descriptions", func() {
		desc1 := v1alpha1.CRDDescription{Name: "memcached3s.cache.example.com", Version: "v1alpha1", Kind: "Memcached3"}
		desc2 := v1alpha1.CRDDescription{Name: "memcached4s.cache.example.com", Version: "v1alpha1", Kind: "Memcached4"}
		defsByGVK[descToGVK(desc1)] = &descriptionValues{crdOrder: 0, crd: desc1}
		defsByGVK[descToGVK(desc2)] = &descriptionValues{crdOrder: 0, crd: desc2}

		owned := []v1alpha1.CRDDescription{
			{Name: "memcached2s.cache.example.com", Version: "v1alpha1", Kind: "Memcached2"},
			{Name: "memcacheds.cache.example.com", Version: "v1alpha1", Kind: "Memcached"},
		}
		expected := []v1alpha1.CRDDescription{desc1, desc2, owned[0], owned[1]}
		startingCSV.Spec.CustomResourceDefinitions.Owned = owned
		csv := startingCSV.DeepCopy()
		updateDefinitionsByKey(csv, defsByGVK)
		Expect(csv.Spec.CustomResourceDefinitions.Owned).To(Equal(expected))
	})
	It("orders one new description with two existing descriptions, one of which is in defsByGVK", func() {
		desc1 := v1alpha1.CRDDescription{Name: "memcached1s.cache.example.com", Version: "v1alpha1", Kind: "Memcached1"}
		desc2 := v1alpha1.CRDDescription{Name: "memcached2s.cache.example.com", Version: "v1alpha1", Kind: "Memcached2"}
		defsByGVK[descToGVK(desc1)] = &descriptionValues{crdOrder: 1, crd: desc1}
		defsByGVK[descToGVK(desc2)] = &descriptionValues{crdOrder: 0, crd: desc2}

		owned := []v1alpha1.CRDDescription{
			desc1,
			{Name: "memcacheds.cache.example.com", Version: "v1alpha1", Kind: "Memcached"},
		}
		expected := []v1alpha1.CRDDescription{desc2, desc1, owned[1]}
		startingCSV.Spec.CustomResourceDefinitions.Owned = owned
		csv := startingCSV.DeepCopy()
		updateDefinitionsByKey(csv, defsByGVK)
		Expect(csv.Spec.CustomResourceDefinitions.Owned).To(Equal(expected))
	})
	It("orders multiple new descriptions and existing descriptions", func() {
		desc1 := v1alpha1.CRDDescription{Name: "memcached3s.cache.example.com", Version: "v1alpha1", Kind: "Memcached3"}
		desc2 := v1alpha1.CRDDescription{Name: "memcached4s.cache.example.com", Version: "v1alpha1", Kind: "Memcached4"}
		desc3 := v1alpha1.CRDDescription{Name: "foo1s.bar.example.com", Version: "v1alpha1", Kind: "Foo1"}
		desc4 := v1alpha1.CRDDescription{Name: "foo2s.bar.example.com", Version: "v1alpha1", Kind: "Foo2"}
		desc5 := v1alpha1.CRDDescription{Name: "bazs.bar.example.com", Version: "v1alpha1", Kind: "Baz"}
		defsByGVK[descToGVK(desc1)] = &descriptionValues{crdOrder: 0, crd: desc1}
		defsByGVK[descToGVK(desc2)] = &descriptionValues{crdOrder: 3, crd: desc2}
		defsByGVK[descToGVK(desc3)] = &descriptionValues{crdOrder: 1, crd: desc3}
		defsByGVK[descToGVK(desc4)] = &descriptionValues{crdOrder: 0, crd: desc4}
		defsByGVK[descToGVK(desc5)] = &descriptionValues{crdOrder: 10000, crd: desc5}

		owned := []v1alpha1.CRDDescription{
			desc2, // Has order 3 above.
			{Name: "memcached2s.cache.example.com", Version: "v1alpha1", Kind: "Memcached2"},
			{Name: "memcacheds.cache.example.com", Version: "v1alpha1", Kind: "Memcached"},
			desc1, // Has order 0 above
			{Name: "bar2.cache.example.com", Version: "v1alpha1", Kind: "Bar2"},
			{Name: "bar1.cache.example.com", Version: "v1alpha1", Kind: "Bar1"},
		}
		expected := []v1alpha1.CRDDescription{desc4, desc1, desc3, owned[1], owned[2], desc2, owned[4], owned[5], desc5}
		startingCSV.Spec.CustomResourceDefinitions.Owned = owned
		csv := startingCSV.DeepCopy()
		updateDefinitionsByKey(csv, defsByGVK)
		Expect(csv.Spec.CustomResourceDefinitions.Owned).To(Equal(expected))
	})
})
