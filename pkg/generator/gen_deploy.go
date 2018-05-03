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

package generator

import (
	"fmt"
	"io"
	"strings"
	"text/template"
)

const (
	crdTmplName      = "deploy/crd.yaml"
	operatorTmplName = "deploy/operator.yaml"
	rbacTmplName     = "deploy/rbac.yaml"
	crTmplName       = "deploy/cr.yaml"
)

// CRDYaml contains data needed to generate deploy/crd.yaml
type CRDYaml struct {
	Kind         string
	KindSingular string
	KindPlural   string
	GroupName    string
	Version      string
}

// renderCRDYaml generates deploy/crd.yaml
func renderCRDYaml(w io.Writer, kind, apiVersion string) error {
	t := template.New(crdTmplName)
	t, err := t.Parse(crdYamlTmpl)
	if err != nil {
		return fmt.Errorf("failed to parse crd yaml template: %v", err)
	}

	ks := strings.ToLower(kind)
	o := CRDYaml{
		Kind:         kind,
		KindSingular: ks,
		KindPlural:   toPlural(ks),
		GroupName:    groupName(apiVersion),
		Version:      version(apiVersion),
	}
	return t.Execute(w, o)
}

// OperatorYaml contains data needed to generate deploy/operator.yaml
type OperatorYaml struct {
	ProjectName string
	Image       string
}

// renderOperatorYaml generates deploy/operator.yaml.
func renderOperatorYaml(w io.Writer, projectName, image string) error {
	t := template.New(operatorTmplName)
	t, err := t.Parse(operatorYamlTmpl)
	if err != nil {
		return fmt.Errorf("failed to parse operator yaml template: %v", err)
	}

	o := OperatorYaml{
		ProjectName: projectName,
		Image:       image,
	}
	return t.Execute(w, o)
}

// RBACYaml contains all the customized data needed to generate deploy/rbac.yaml for a new operator
// when pairing with rbacYamlTmpl template.
type RBACYaml struct {
	ProjectName string
	GroupName   string
}

// renderRBACYaml generates deploy/rbac.yaml.
func renderRBACYaml(w io.Writer, projectName, groupName string) error {
	t := template.New(rbacTmplName)
	t, err := t.Parse(rbacYamlTmpl)
	if err != nil {
		return fmt.Errorf("failed to parse rbac yaml template: %v", err)
	}

	r := RBACYaml{
		ProjectName: projectName,
		GroupName:   groupName,
	}
	return t.Execute(w, r)
}

// CRYaml contains all the customized data needed to generate deploy/cr.yaml.
type CRYaml struct {
	APIVersion string
	Kind       string
	Name       string
}

func renderCustomResourceYaml(w io.Writer, apiVersion, kind string) error {
	t := template.New(crTmplName)
	t, err := t.Parse(crYamlTmpl)
	if err != nil {
		return fmt.Errorf("failed to parse cr yaml template: %v", err)
	}

	r := CRYaml{
		APIVersion: apiVersion,
		Kind:       kind,
	}
	return t.Execute(w, r)
}
