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
	"regexp"
	"strings"

	"github.com/markbates/inflect"
	"k8s.io/apimachinery/pkg/util/validation"
	"sigs.k8s.io/kubebuilder/pkg/model/config"
)

var (
	// ResourceVersionRegexp matches Kubernetes API versions.
	// See https://kubernetes.io/docs/concepts/overview/kubernetes-api/#api-versioning
	ResourceVersionRegexp = regexp.MustCompile("^v[1-9][0-9]*((alpha|beta)[1-9][0-9]*)?$")
	// ResourceKindRegexp matches Kubernetes API Kind's.
	ResourceKindRegexp = regexp.MustCompile("^[A-Z]{1}[a-zA-Z0-9]+$")
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

	// GoImportGroup is the non-hyphenated go import group for this resource
	GoImportGroup string

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

	if err := r.checkAndSetKinds(); err != nil {
		return err
	}
	if err := r.checkAndSetGroups(); err != nil {
		return err
	}
	if err := r.checkAndSetVersion(); err != nil {
		return err
	}

	rs := inflect.NewDefaultRuleset()
	if len(r.Resource) == 0 {
		r.Resource = rs.Pluralize(strings.ToLower(r.Kind))
	}

	return nil
}

// ResourceData holds data used to construct a Resource.
type ResourceData struct {
	APIVersion string
	Domain     string
	Group      string
	Version    string
	Kind       string
}

// ToResource transforms ResourceData into a full Resource.
func (d ResourceData) ToResource() (Resource, error) {
	if err := d.validate(); err != nil {
		return Resource{}, fmt.Errorf("invalid resource: %v", err)
	}

	apiVersion := d.APIVersion
	if apiVersion == "" {
		apiVersion = fmt.Sprintf("%s.%s/%s", d.Group, d.Domain, d.Version)
	}
	r, err := NewResource(apiVersion, d.Kind)
	return *r, err
}

// Validate ensures fields in ResourceData conform to their corresponding Kubernetes specs.
func (d ResourceData) validate() error {
	// Group and Domain can be empty if ResourceData is used to create a native k8s resource
	// that does not require either, ex. a ConfigMap.
	if d.Group != "" && d.Domain != "" {
		if strings.HasSuffix(d.Group, d.Domain) {
			return fmt.Errorf("group %q cannot contain a domain suffix", d.Group)
		}
		if errs := validation.IsQualifiedName(fmt.Sprintf("%s.%s", d.Group, d.Domain)); len(errs) != 0 {
			return fmt.Errorf("%+q", errs)
		}
	}
	if d.Version == "" {
		return errors.New("version cannot be empty")
	}
	if d.Kind == "" {
		return errors.New("kind cannot be empty")
	}

	return nil
}

// ToGVK transforms ResourceData into a GVK. Useful when working with a Config.
func (d ResourceData) ToGVK() config.GVK {
	return config.GVK{Group: d.Group, Version: d.Version, Kind: d.Kind}
}

func (r *Resource) checkAndSetKinds() error {
	if len(r.Kind) == 0 {
		return errors.New("kind cannot be empty")
	}

	r.LowerKind = strings.ToLower(r.Kind)

	if strings.Title(r.Kind) != r.Kind {
		return fmt.Errorf("kind must begin with uppercase (was %v)", r.Kind)
	}
	if !ResourceKindRegexp.MatchString(r.Kind) {
		return errors.New("kind should consist of lower and uppercase alphabetical characters")
	}
	return nil
}

func (r *Resource) checkAndSetGroups() error {
	fg := strings.Split(r.APIVersion, "/")
	if len(fg) < 2 || len(fg[0]) == 0 {
		return errors.New("full group cannot be empty")
	}
	g := strings.Split(fg[0], ".")
	if len(g) == 0 || len(g[0]) == 0 {
		return errors.New("group cannot be empty")
	}
	r.FullGroup = fg[0]
	r.Group = g[0]

	s := strings.ToLower(r.Group)
	r.GoImportGroup = strings.Replace(s, "-", "", -1)

	if err := validation.IsDNS1123Subdomain(r.Group); err != nil {
		return fmt.Errorf("group name is invalid: %v", err)
	}
	return nil
}

func (r *Resource) checkAndSetVersion() error {
	api := strings.Split(r.APIVersion, "/")
	if len(api) < 2 || len(api[1]) == 0 {
		return errors.New("version cannot be empty")
	}
	r.Version = api[1]

	if !ResourceVersionRegexp.MatchString(r.Version) {
		return errors.New("version is not in the correct Kubernetes version format, ex. v1alpha1")
	}
	return nil
}
