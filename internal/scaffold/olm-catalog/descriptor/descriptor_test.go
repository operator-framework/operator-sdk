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

	"github.com/ghodss/yaml"
	olmapiv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const testFrameworkPackage = "github.com/operator-framework/operator-sdk/test/test-framework"

func getTestFrameworkDir(t *testing.T) string {
	t.Helper()
	absPath, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	sdkPath := absPath[:strings.Index(absPath, "internal")]
	tfDir := filepath.Join(sdkPath, "test", "test-framework")
	// parser.AddDirRecursive doesn't like absolute paths.
	relPath, err := filepath.Rel(absPath, tfDir)
	if err != nil {
		t.Fatal(err)
	}
	return relPath
}

func TestGetKindTypeForAPI(t *testing.T) {
	cases := []struct {
		description string
		pkg, kind   string
		numPkgTypes int
		wantNil     bool
	}{
		{
			"Find types successfully",
			testFrameworkPackage, "Dummy", 21, false,
		},
		{
			"Find types with error from wrong kind",
			testFrameworkPackage, "NotFound", 21, true,
		},
	}
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	tfDir := getTestFrameworkDir(t)
	if err := os.Chdir(tfDir); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err = os.Chdir(wd); err != nil {
			t.Fatal(err)
		}
	}()
	tfAPIDir := filepath.Join("pkg", "apis", "cache", "v1alpha1")
	universe, err := getTypesFromDir(tfAPIDir)
	if err != nil {
		t.Fatal(err)
	}

	for _, c := range cases {
		pkgTypes, err := getTypesForPkg(c.pkg, universe)
		if err != nil {
			t.Fatal(err)
		}
		if n := len(pkgTypes); n != c.numPkgTypes {
			t.Errorf("%s: expected %d package types, got %d", c.description, c.numPkgTypes, n)
		}
		kindType := findKindType(c.kind, pkgTypes)
		if c.wantNil && kindType != nil {
			t.Errorf("%s: expected type %q to not be found", c.description, kindType.Name)
		}
		if !c.wantNil && kindType == nil {
			t.Errorf("%s: expected type %q to be found", c.description, c.kind)
		}
		if !c.wantNil && kindType != nil && kindType.Name.Name != c.kind {
			t.Errorf("%s: expected type %q to have type name %q", c.description, kindType.Name, c.kind)
		}
	}
}

func TestGetCRDDescriptionForGVK(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	tfDir := getTestFrameworkDir(t)
	if err := os.Chdir(tfDir); err != nil {
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
					{Path: "size", DisplayName: "dummy-pods", Description: "Should be in spec", XDescriptors: xdescsFor("size")},
				},
				StatusDescriptors: []olmapiv1alpha1.StatusDescriptor{
					{Path: "hog.engine", DisplayName: "boss-hog-engine", Description: "Should be in status but not spec, since Hog isn't in DummySpec"},
					{Path: "hog.foo", DisplayName: "Public"},
					{Path: "hog.seatMaterial", DisplayName: "Seat Material"},
					{Path: "hog.seatMaterial", DisplayName: "Seat Material"},
					{Path: "nodes", DisplayName: "Nodes", Description: "Should be in status but not spec, since DummyStatus isn't in DummySpec"},
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
					{Path: "engine", DisplayName: "Engine", Description: "Should be in status but not spec, since Hog isn't in DummySpec"},
					{Path: "foo", DisplayName: "Public"},
					{Path: "seatMaterial", DisplayName: "Seat Material"},
					{Path: "seatMaterial", DisplayName: "Seat Material"},
				},
				StatusDescriptors: []olmapiv1alpha1.StatusDescriptor{
					{Path: "nothing", DisplayName: "Nothing", Description: "Should be in status but not spec, since this isn't a spec type"},
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
