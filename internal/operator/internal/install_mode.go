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

package internal

import (
	"flag"
	"fmt"
	"sort"
	"strings"

	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	"k8s.io/apimachinery/pkg/util/validation"
)

type InstallMode struct {
	InstallModeType  v1alpha1.InstallModeType
	TargetNamespaces []string
}

var _ flag.Value = &InstallMode{}

func (i *InstallMode) Set(str string) error {
	split := strings.SplitN(str, "=", 2)
	i.InstallModeType = v1alpha1.InstallModeType(split[0])
	if len(split) == 2 {
		namespaces := strings.Split(split[1], ",")
		for _, ns := range namespaces {
			i.TargetNamespaces = append(i.TargetNamespaces, strings.TrimSpace(ns))
		}
		sort.Strings(i.TargetNamespaces)
	}
	return i.Validate()
}

func (i InstallMode) IsEmpty() bool {
	return i.InstallModeType == ""
}

func (i InstallMode) String() string {
	switch i.InstallModeType {
	case v1alpha1.InstallModeTypeSingleNamespace, v1alpha1.InstallModeTypeMultiNamespace:
		return fmt.Sprintf("%s=%s", i.InstallModeType, strings.Join(i.TargetNamespaces, ","))
	default:
		return string(i.InstallModeType)
	}
}

func (InstallMode) Type() string {
	return "InstallModeValue"
}

func (i InstallMode) Validate() error {
	switch i.InstallModeType {
	case v1alpha1.InstallModeTypeAllNamespaces, v1alpha1.InstallModeTypeOwnNamespace:
		if len(i.TargetNamespaces) != 0 {
			return fmt.Errorf("install mode %q must have zero target namespaces", i.InstallModeType)
		}
	case v1alpha1.InstallModeTypeSingleNamespace:
		if len(i.TargetNamespaces) != 1 {
			return fmt.Errorf("install mode %q must have exactly one target namespace", i.InstallModeType)
		}
	case v1alpha1.InstallModeTypeMultiNamespace:
		if len(i.TargetNamespaces) == 0 {
			return fmt.Errorf("install mode %q must have at least one target namespace", i.InstallModeType)
		}
	case "":
		if len(i.TargetNamespaces) != 0 {
			return fmt.Errorf("target namespaces defined without type")
		}
	default:
		return fmt.Errorf("unknown install mode type")
	}
	for _, ns := range i.TargetNamespaces {
		errs := validation.IsDNS1123Label(ns)
		if len(errs) > 0 {
			return fmt.Errorf("invalid target namespace %q: %v", ns, strings.Join(errs, ", "))
		}
	}
	return nil
}
