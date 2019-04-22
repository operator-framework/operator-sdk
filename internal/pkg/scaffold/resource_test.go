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

// Modified from github.com/kubernetes-sigs/controller-tools/pkg/scaffold/resource/resource.go

package scaffold

import (
	"testing"

	"github.com/operator-framework/operator-sdk/internal/pkg/scaffold/input"
)

type fakeResource struct {
	Resource Resource
}

func (s *fakeResource) GetInput() (_ input.Input, _ error) { return }

type fakeResourceEmbed struct {
	Resource
}

func (s *fakeResourceEmbed) GetInput() (_ input.Input, _ error) { return }

type fakeResourcePtr struct {
	Resource *Resource
}

func (s *fakeResourcePtr) GetInput() (_ input.Input, _ error) { return }

func TestValidateFileResource(t *testing.T) {
	// Empty, should see errors.
	if err := ValidateFileResource(&fakeResource{}); err == nil {
		t.Error("expected error validating empty file resource, got none")
	}
	if err := ValidateFileResource(&fakeResourceEmbed{}); err == nil {
		t.Error("expected error validating empty file resource embed, got none")
	}
	if err := ValidateFileResource(&fakeResourcePtr{}); err == nil {
		t.Error("expected error validating empty file resource pointer, got none")
	}

	// Not empty, should not see errors.
	if err := ValidateFileResource(&fakeResource{Resource{GoImportGroup: "foo"}}); err != nil {
		t.Errorf("expected no error validating file resource, got %v", err)
	}
	if err := ValidateFileResource(&fakeResourceEmbed{Resource{GoImportGroup: "bar"}}); err != nil {
		t.Errorf("expected no error validating file resource embed, got %v", err)
	}
	if err := ValidateFileResource(&fakeResourcePtr{&Resource{GoImportGroup: "baz"}}); err != nil {
		t.Errorf("expected no error validating file resource pointer, got %v", err)
	}
}
