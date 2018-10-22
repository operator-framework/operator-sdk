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

package cmdutil

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/operator-framework/operator-sdk/pkg/scaffold"

	yaml "gopkg.in/yaml.v2"
	rbacv1 "k8s.io/api/rbac/v1"
	cgoscheme "k8s.io/client-go/kubernetes/scheme"
)

const (
	gopkgToml       = "./Gopkg.toml"
	buildDockerfile = "./build/Dockerfile"
)

// OperatorType - the type of operator
type OperatorType = string

const (
	// OperatorTypeGo - golang type of operator.
	OperatorTypeGo OperatorType = "go"
	// OperatorTypeAnsible - ansible type of operator.
	OperatorTypeAnsible OperatorType = "ansible"
)

const (
	GopathEnv = "GOPATH"
	SrcDir    = "src"

	DefaultDirFileMode  = 0750
	DefaultFileMode     = 0644
	DefaultExecFileMode = 0744
)

// MustInProjectRoot checks if the current dir is the project root and returns the current repo's import path
// e.g github.com/example-inc/app-operator
func MustInProjectRoot() {
	// if the current directory has the "./build/dockerfile" file, then it is safe to say
	// we are at the project root.
	_, err := os.Stat(buildDockerfile)
	if err != nil && os.IsNotExist(err) {
		log.Fatalf("must run command in project root dir: %v", err)
	}
}

func MustGetwd() string {
	wd, err := os.Getwd()
	if err != nil {
		log.Fatalf("failed to get working directory: (%v)", err)
	}
	return wd
}

// CheckAndGetCurrPkg checks if this project's repository path is rooted under $GOPATH and returns the current directory's import path
// e.g: "github.com/example-inc/app-operator"
func CheckAndGetCurrPkg() string {
	gopath := os.Getenv(GopathEnv)
	if len(gopath) == 0 {
		log.Fatalf("get current pkg failed: GOPATH env not set")
	}
	goSrc := filepath.Join(gopath, SrcDir)

	wd := MustGetwd()
	if !strings.HasPrefix(filepath.Dir(wd), goSrc) {
		log.Fatalf("check current pkg failed: must run from gopath")
	}
	currPkg := strings.Replace(wd, goSrc+string(filepath.Separator), "", 1)
	// strip any "/" prefix from the repo path.
	return strings.TrimPrefix(currPkg, string(filepath.Separator))
}

// GetOperatorType returns type of operator is in cwd
// This function should be called after verifying the user is in project root
// e.g: "go", "ansible"
func GetOperatorType() OperatorType {
	// Assuming that if Gopkg.toml exists then this is a Go operator
	_, err := os.Stat(gopkgToml)
	if err != nil && os.IsNotExist(err) {
		return OperatorTypeAnsible
	}
	return OperatorTypeGo
}

func UpdateRoleForResource(r *scaffold.Resource, absProjectPath string) error {
	// append rbac rule to deploy/role.yaml
	roleFilePath := filepath.Join(absProjectPath, "deploy", "role.yaml")
	roleYAML, err := ioutil.ReadFile(roleFilePath)
	if err != nil {
		return fmt.Errorf("failed to read role manifest %v: %v", roleFilePath, err)
	}
	obj, _, err := cgoscheme.Codecs.UniversalDeserializer().Decode(roleYAML, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to decode role manifest %v: %v", roleFilePath, err)
	}
	switch role := obj.(type) {
	// TODO: use rbac/v1.
	case *rbacv1.Role:
		pr := &rbacv1.PolicyRule{}
		apiGroupFound := false
		for i := range role.Rules {
			if role.Rules[i].APIGroups[0] == r.FullGroup {
				apiGroupFound = true
				pr = &role.Rules[i]
				break
			}
		}
		// check if the resource already exists
		for _, resource := range pr.Resources {
			if resource == r.Resource {
				log.Printf("deploy/role.yaml RBAC rules already up to date for the resource (%v, %v)", r.APIVersion, r.Kind)
				return nil
			}
		}

		pr.Resources = append(pr.Resources, r.Resource)
		// create a new apiGroup if not found.
		if !apiGroupFound {
			pr.APIGroups = []string{r.FullGroup}
			// Using "*" to allow access to the resource and all its subresources e.g "memcacheds" and "memcacheds/finalizers"
			// https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/#ownerreferencespermissionenforcement
			pr.Resources = []string{"*"}
			pr.Verbs = []string{"*"}
			role.Rules = append(role.Rules, *pr)
		}
		// update role.yaml
		d, err := json.Marshal(&role)
		if err != nil {
			return fmt.Errorf("failed to marshal role(%+v): %v", role, err)
		}
		m := &map[string]interface{}{}
		err = yaml.Unmarshal(d, m)
		data, err := yaml.Marshal(m)
		if err != nil {
			return fmt.Errorf("failed to marshal role(%+v): %v", role, err)
		}
		if err := ioutil.WriteFile(roleFilePath, data, DefaultFileMode); err != nil {
			return fmt.Errorf("failed to update %v: %v", roleFilePath, err)
		}
	case *rbacv1.ClusterRole:
		pr := &rbacv1.PolicyRule{}
		apiGroupFound := false
		for i := range role.Rules {
			if role.Rules[i].APIGroups[0] == r.FullGroup {
				apiGroupFound = true
				pr = &role.Rules[i]
				break
			}
		}
		// check if the resource already exists
		for _, resource := range pr.Resources {
			if resource == r.Resource {
				log.Printf("deploy/role.yaml RBAC rules already up to date for the resource (%v, %v)", r.APIVersion, r.Kind)
				return nil
			}
		}

		pr.Resources = append(pr.Resources, r.Resource)
		// create a new apiGroup if not found.
		if !apiGroupFound {
			pr.APIGroups = []string{r.FullGroup}
			// Using "*" to allow access to the resource and all its subresources e.g "memcacheds" and "memcacheds/finalizers"
			// https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/#ownerreferencespermissionenforcement
			pr.Resources = []string{"*"}
			pr.Verbs = []string{"*"}
			role.Rules = append(role.Rules, *pr)
		}
		// update role.yaml
		d, err := json.Marshal(&role)
		if err != nil {
			return fmt.Errorf("failed to marshal role(%+v): %v", role, err)
		}
		m := &map[string]interface{}{}
		err = yaml.Unmarshal(d, m)
		data, err := yaml.Marshal(m)
		if err != nil {
			return fmt.Errorf("failed to marshal role(%+v): %v", role, err)
		}
		if err := ioutil.WriteFile(roleFilePath, data, DefaultFileMode); err != nil {
			return fmt.Errorf("failed to update %v: %v", roleFilePath, err)
		}
	default:
		return errors.New("failed to parse role.yaml as a role")
	}
	// not reachable
	return nil
}
