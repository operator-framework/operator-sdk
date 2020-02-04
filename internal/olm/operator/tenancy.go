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

package olm

import (
	"encoding/json"
	"fmt"
	"strings"

	olmapiv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"github.com/operator-framework/operator-registry/pkg/registry"
)

// Mapping of installMode string values to types, for validation.
var installModeStrings = map[string]olmapiv1alpha1.InstallModeType{
	string(olmapiv1alpha1.InstallModeTypeOwnNamespace):    olmapiv1alpha1.InstallModeTypeOwnNamespace,
	string(olmapiv1alpha1.InstallModeTypeSingleNamespace): olmapiv1alpha1.InstallModeTypeSingleNamespace,
	string(olmapiv1alpha1.InstallModeTypeMultiNamespace):  olmapiv1alpha1.InstallModeTypeMultiNamespace,
	string(olmapiv1alpha1.InstallModeTypeAllNamespaces):   olmapiv1alpha1.InstallModeTypeAllNamespaces,
}

// installModeCompatible ensures installMode is compatible with the namespaces
// and CSV's installModes being used.
func (m operatorManager) installModeCompatible(installMode olmapiv1alpha1.InstallModeType) error {
	err := validateInstallModeForNamespaces(installMode, m.installModeNamespaces)
	if err != nil {
		return err
	}
	if installMode == olmapiv1alpha1.InstallModeTypeOwnNamespace {
		if ns := m.installModeNamespaces[0]; ns != m.namespace {
			return fmt.Errorf("installMode %s namespace %q must match namespace %q",
				installMode, ns, m.namespace)
		}
	}
	// Ensure CSV supports installMode.
	bundle, err := getBundleForVersion(m.bundles, m.version)
	if err != nil {
		return err
	}
	bcsv, err := bundle.ClusterServiceVersion()
	if err != nil {
		return err
	}
	csv, err := bundleCSVToCSV(bcsv)
	if err != nil {
		return err
	}
	for _, mode := range csv.Spec.InstallModes {
		if mode.Type == installMode && !mode.Supported {
			return fmt.Errorf("installMode %s not supported in CSV %q", installMode, csv.GetName())
		}
	}
	return nil
}

// parseInstallModeKV parses an installMode string of the format
// installModeFormat.
func parseInstallModeKV(raw string) (olmapiv1alpha1.InstallModeType, []string, error) {
	modeSplit := strings.Split(raw, "=")
	if len(modeSplit) != 2 {
		return "", nil, fmt.Errorf("installMode string %q is malformatted, must be: %s", raw, installModeFormat)
	}
	modeStr, namespaceList := modeSplit[0], modeSplit[1]
	mode, ok := installModeStrings[modeStr]
	if !ok {
		return "", nil, fmt.Errorf("installMode type string %q is not a valid installMode type", modeStr)
	}
	namespaces := []string{}
	namespaces = append(namespaces, strings.Split(strings.Trim(namespaceList, ","), ",")...)
	return mode, namespaces, nil
}

// validateInstallModeForNamespaces ensures namespaces are valid given mode.
func validateInstallModeForNamespaces(mode olmapiv1alpha1.InstallModeType, namespaces []string) error {
	switch mode {
	case olmapiv1alpha1.InstallModeTypeOwnNamespace, olmapiv1alpha1.InstallModeTypeSingleNamespace:
		if len(namespaces) != 1 || namespaces[0] == "" {
			return fmt.Errorf("installMode %s must be passed with exactly one non-empty namespace, have: %+q",
				mode, namespaces)
		}
	case olmapiv1alpha1.InstallModeTypeMultiNamespace:
		if len(namespaces) < 2 {
			return fmt.Errorf("installMode %s must be passed with more than one non-empty namespaces, have: %+q",
				mode, namespaces)
		}
	case olmapiv1alpha1.InstallModeTypeAllNamespaces:
		if len(namespaces) != 1 || namespaces[0] != "" {
			return fmt.Errorf("installMode %s must be passed with exactly one empty namespace, have: %+q",
				mode, namespaces)
		}
	default:
		return fmt.Errorf("installMode %q is not a valid installMode type", mode)
	}
	return nil
}

// bundleCSVToCSV converts a registry.ClusterServiceVersion bcsv to a
// v1alpha1.ClusterServiceVersion. The returned type will not have a status.
func bundleCSVToCSV(bcsv *registry.ClusterServiceVersion) (*olmapiv1alpha1.ClusterServiceVersion, error) {
	spec := olmapiv1alpha1.ClusterServiceVersionSpec{}
	if err := json.Unmarshal(bcsv.Spec, &spec); err != nil {
		return nil, fmt.Errorf("error converting bundle CSV %q type: %w", bcsv.GetName(), err)
	}
	return &olmapiv1alpha1.ClusterServiceVersion{
		TypeMeta:   bcsv.TypeMeta,
		ObjectMeta: bcsv.ObjectMeta,
		Spec:       spec,
	}, nil
}
