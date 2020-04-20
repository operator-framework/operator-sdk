// Copyright 2019 The Operator-SDK Authors
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

package descriptor

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/operator-framework/operator-sdk/internal/util/diffutil"

	olmapiv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/gengo/types"
	"sigs.k8s.io/yaml"
)

var (
	testDataDir = filepath.Join("..", "..", "testdata", "go")
)

func TestGetKindTypeForAPI(t *testing.T) {
	multiAPIRootDir := filepath.Join("pkg", "apis")
	singleAPIRootDir := "api"
	group := "cache"
	version := "v1alpha1"

	subTests := []struct {
		description string
		// path to apis types root dir e.g pkg/apis
		apisDir string
		// path to kind api pkg e.g pkg/apis/cache/v1alpha1
		expectedPkgPath string
		group           string
		version         string
		kind            string
		numPkgTypes     int
		wantNil         bool
		// True if apis dir in the expected single or multi group layout
		isExpectedLayout bool
	}{
		{
			"Must Succeed: Find types for Kind from multi APIs root directory",
			multiAPIRootDir,
			filepath.Join(multiAPIRootDir, group, version),
			group,
			version,
			"Dummy",
			22,
			false,
			true,
		},
		{
			"Must Fail: Find types for non-existing Kind from multi APIs root directory",
			multiAPIRootDir,
			filepath.Join(multiAPIRootDir, group, version),
			group,
			version,
			"NotFound",
			22,
			true,
			true,
		},
		{
			"Must Succeed: Find types for Kind from single APIs root directory",
			singleAPIRootDir,
			filepath.Join(singleAPIRootDir, version),
			group,
			version,
			"Memcached",
			4,
			false,
			true,
		},
		{
			"Must Fail: Find types for non-existing Kind from single APIs root directory",
			singleAPIRootDir,
			filepath.Join(singleAPIRootDir, version),
			group,
			version,
			"NotFound",
			4,
			true,
			true,
		},
		// TODO: Add cases for non-standard api dir layouts: pkg/apis/<foo>/<bar>/version
	}

	// Change directory to test data dir so the test cases can form the correct pkg imports
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

	for _, st := range subTests {
		t.Run(st.description, func(t *testing.T) {
			expectedPkgPath, err := getExpectedPkgLayout(st.apisDir, st.group, st.version)
			if err != nil {
				t.Fatalf("Failed to getExpectedPkgLayout(%s, %s, %s): %v", st.apisDir, st.group, st.version, err)
			}
			if st.isExpectedLayout {
				if expectedPkgPath == "" || !strings.HasSuffix(expectedPkgPath, st.expectedPkgPath) {
					t.Fatalf("Expected (%s) as suffix to expected pkg path (%s)", st.expectedPkgPath, expectedPkgPath)
				}
			}

			var pkgTypes []*types.Type
			if st.isExpectedLayout {
				universe, err := getPkgsFromDirRecursive(expectedPkgPath)
				if err != nil {
					t.Fatalf("Failed to get universe of types from API root directory (%s): %v)", st.apisDir, err)
				}
				pkgTypes, err = getTypesForPkgPath(expectedPkgPath, universe)
				if err != nil {
					t.Fatalf("Failed to get types of pkg path (%s) from API root directory(%s): %v)",
						expectedPkgPath, st.apisDir, err)
				}
			} else {
				universe, err := getPkgsFromDirRecursive(st.apisDir)
				if err != nil {
					t.Fatalf("Failed to get universe of types from API root directory (%s): %v)", st.apisDir, err)
				}
				pkgTypes, err = getTypesForPkgName(st.version, universe)
				if err != nil {
					t.Fatalf("Failed to get types of pkg name (%s) from API root directory(%s): %v)", st.version, st.apisDir, err)
				}
			}

			if n := len(pkgTypes); n != st.numPkgTypes {
				t.Errorf("Expected %d package types, got %d", st.numPkgTypes, n)
			}
			kindType := findKindType(st.kind, pkgTypes)
			if st.wantNil && kindType != nil {
				t.Errorf("Expected type %q to not be found", kindType.Name)
			}
			if !st.wantNil && kindType == nil {
				t.Errorf("Expected type %q to be found", st.kind)
			}
			if !st.wantNil && kindType != nil && kindType.Name.Name != st.kind {
				t.Errorf("Expected type %q to have type name %q", kindType.Name, st.kind)
			}
		})
	}
}

func TestGetCRDDescriptionForGVK(t *testing.T) {

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
	xdescsFor := func(paths ...string) (xdescs []string) {
		for _, p := range paths {
			xdescs = append(xdescs, getSpecXDescriptorsByPath(nil, p)...)
		}
		return xdescs
	}

	// TODO(hasbro17): Change to run as subtests
	cases := []struct {
		description string
		apisDir     string
		gvk         schema.GroupVersionKind
		expected    olmapiv1alpha1.CRDDescription
		wantErr     bool
		expErr      error
	}{
		{
			"Populate CRDDescription successfully",
			filepath.Join("pkg", "apis"),
			schema.GroupVersionKind{Group: "cache.example.com", Version: "v1alpha1", Kind: "Dummy"},
			olmapiv1alpha1.CRDDescription{
				Kind:        "Dummy",
				Version:     "v1alpha1",
				DisplayName: "Dummy App",
				Description: "Dummy is the Schema for the dummy API",
				Resources: []olmapiv1alpha1.APIResourceReference{
					{Name: "dummy-deployment", Kind: "Deployment", Version: "v1"},
					{Name: "dummy-pod", Kind: "Pod", Version: "v1"},
					{Name: "dummy-replicaset", Kind: "ReplicaSet", Version: "v1beta2"},
				},
				SpecDescriptors: []olmapiv1alpha1.SpecDescriptor{
					{Path: "size", DisplayName: "dummy-pods", Description: "Should be in spec",
						XDescriptors: xdescsFor("size")},
					{Path: "wheels", DisplayName: "Wheels", Description: "Should be in spec, but should not have array index in path",
						XDescriptors: []string{"urn:alm:descriptor:com.tectonic.ui:text"}},
					{Path: "wheels[0].type", DisplayName: "Wheel Type",
						Description: "Type should be in spec with path equal to wheels[0].type",
						XDescriptors: []string{
							"urn:alm:descriptor:com.tectonic.ui:arrayFieldGroup:wheels",
							"urn:alm:descriptor:com.tectonic.ui:text",
						}},
				},
				StatusDescriptors: []olmapiv1alpha1.StatusDescriptor{
					{Path: "hog.engine", DisplayName: "boss-hog-engine",
						Description: "Should be in status but not spec, since Hog isn't in DummySpec"},
					{Path: "hog.foo", DisplayName: "Public"},
					{Path: "hog.seatMaterial", DisplayName: "Seat Material"},
					{Path: "hog.seatMaterial", DisplayName: "Seat Material"},
					{Path: "nodes", DisplayName: "Nodes",
						Description: "Should be in status but not spec, since DummyStatus isn't in DummySpec"},
				},
			},
			false,
			nil,
		},
		{
			"Populate CRDDescription with non-standard spec type successfully",
			filepath.Join("pkg", "apis"),
			schema.GroupVersionKind{Group: "cache.example.com", Version: "v1alpha1", Kind: "OtherDummy"},
			olmapiv1alpha1.CRDDescription{
				Kind:        "OtherDummy",
				Version:     "v1alpha1",
				DisplayName: "Other Dummy App",
				Description: "OtherDummy is the Schema for the other dummy API",
				Resources: []olmapiv1alpha1.APIResourceReference{
					{Name: "other-dummy-pod", Kind: "Pod", Version: "v1"},
					{Name: "other-dummy-service", Kind: "Service", Version: "v1"},
				},
				SpecDescriptors: []olmapiv1alpha1.SpecDescriptor{
					{Path: "engine", DisplayName: "Engine",
						Description: "Should be in status but not spec, since Hog isn't in DummySpec"},
					{Path: "foo", DisplayName: "Public"},
					{Path: "seatMaterial", DisplayName: "Seat Material"},
					{Path: "seatMaterial", DisplayName: "Seat Material"},
				},
				StatusDescriptors: []olmapiv1alpha1.StatusDescriptor{
					{Path: "nothing", DisplayName: "Nothing",
						Description: "Should be in status but not spec, since this isn't a spec type"},
				},
			},
			false,
			nil,
		},
		{
			"Fail to populate CRDDescription with skip on dir not exist",
			filepath.Join("pkg", "notexist"),
			schema.GroupVersionKind{Group: "cache.example.com", Version: "v1alpha1", Kind: "Dummy"},
			olmapiv1alpha1.CRDDescription{},
			true,
			ErrAPIDirNotExist,
		},
		{
			"Fail to populate CRDDescription with skip on type",
			filepath.Join("pkg", "apis"),
			schema.GroupVersionKind{Group: "cache.example.com", Version: "v1alpha1", Kind: "NoKind"},
			olmapiv1alpha1.CRDDescription{},
			true,
			ErrAPITypeNotFound,
		},
	}

	for _, c := range cases {
		description, err := GetCRDDescriptionForGVK(c.apisDir, c.gvk)
		if !c.wantErr && err != nil {
			t.Errorf("%s: expected nil error, got %q", c.description, err)
		} else if c.wantErr && err == nil {
			t.Errorf("%s: expected non-nil error, got nil error", c.description)
		} else if !c.wantErr && err == nil {
			if !reflect.DeepEqual(c.expected, description) {
				be, _ := yaml.Marshal(c.expected)
				bg, _ := yaml.Marshal(description)
				t.Errorf("%s: populated CRDDescription not equal to expected:\n%s",
					c.description, diffutil.Diff(string(be), string(bg)))
			}
		}
	}
}
