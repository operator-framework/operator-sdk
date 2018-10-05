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
	"errors"
	"fmt"
	"strings"

	"github.com/markbates/inflect"
)

// Resource contains the information required to scaffold files for a resource.
type Resource struct {
	// APIVersion is the complete group-subdomain/version e.g app.example.com/v1alpha1
	APIVersion string

	// Kind is the API Kind e.g AppService
	Kind string

	// FullGroup is the complete group name with subdomain e.g app.example.com
	// Parsed from APIVersion
	FullGroup string

	// Group is the API Group.  Does not contain the sub-domain. e.g app
	// Parsed from APIVersion
	Group string

	// Version is the API version - e.g. v1alpha1
	// Parsed from APIVersion
	Version string

	// Resource is the API Resource i.e plural(lowercased(Kind)) e.g appservices
	Resource string

	// LowerKind is lowercased(Kind) e.g appservice
	LowerKind string

	// TODO: allow user to specify list of short names for Resource e.g app, myapp
}

func NewResource(apiVersion, kind string) (*Resource, error) {
	r := &Resource{
		APIVersion: apiVersion,
		Kind:       kind,
	}
	if err := r.Validate(); err != nil {
		return nil, err
	}
	return r, nil
}

// Validate defaults and checks the Resource values to make sure they are valid.
func (r *Resource) Validate() error {
	if len(r.APIVersion) == 0 {
		return errors.New("api-version cannot be empty")
	}

	r.FullGroup = strings.Split(r.APIVersion, "/")[0]
	r.Version = strings.Split(r.APIVersion, "/")[1]
	r.Group = strings.Split(r.FullGroup, ".")[0]

	if len(r.Group) == 0 {
		return errors.New("group cannot be empty")
	}
	if len(r.Version) == 0 {
		return errors.New("version cannot be empty")
	}
	if len(r.Kind) == 0 {
		return errors.New("kind cannot be empty")
	}

	r.LowerKind = strings.ToLower(r.Kind)

	// TODO: regex match kind to only be [aA-zZ]]
	if strings.Title(r.Kind) != r.Kind {
		return fmt.Errorf("kind must begin with uppercase (was %v)", r.Kind)
	}

	rs := inflect.NewDefaultRuleset()
	if len(r.Resource) == 0 {
		r.Resource = rs.Pluralize(strings.ToLower(r.Kind))
	}

	// TODO: regex match group (without subdomain) must be lowercased [a-z]

	// TODO: regex match version to be v1, v1alpha1, v1beta1 etc

	return nil
}
