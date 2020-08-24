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
	"context"
	"fmt"
	"reflect"
	"sort"
	"strings"

	operatorsv1 "github.com/operator-framework/api/pkg/operators/v1"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	log "github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/client"

	olmclient "github.com/operator-framework/operator-sdk/internal/olm/client"
	"github.com/operator-framework/operator-sdk/internal/operator"
)

// Mapping of installMode string values to types, for validation.
var installModeStrings = map[string]operatorsv1alpha1.InstallModeType{
	string(operatorsv1alpha1.InstallModeTypeOwnNamespace):    operatorsv1alpha1.InstallModeTypeOwnNamespace,
	string(operatorsv1alpha1.InstallModeTypeSingleNamespace): operatorsv1alpha1.InstallModeTypeSingleNamespace,
	string(operatorsv1alpha1.InstallModeTypeMultiNamespace):  operatorsv1alpha1.InstallModeTypeMultiNamespace,
	string(operatorsv1alpha1.InstallModeTypeAllNamespaces):   operatorsv1alpha1.InstallModeTypeAllNamespaces,
}

// installModeCompatible ensures installMode is compatible with the namespaces
// and CSV's installModes being used.
func installModeCompatible(csv *operatorsv1alpha1.ClusterServiceVersion, installMode operatorsv1alpha1.InstallModeType,
	operatorNamespace string, targetNamespaces []string) error {

	err := validateInstallModeForNamespaces(installMode, targetNamespaces)
	if err != nil {
		return err
	}
	if installMode == operatorsv1alpha1.InstallModeTypeOwnNamespace {
		if ns := targetNamespaces[0]; ns != operatorNamespace {
			return fmt.Errorf("installMode %s namespace %q must match namespace %q",
				installMode, ns, operatorNamespace)
		}
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
func parseInstallModeKV(raw, operatorNs string) (operatorsv1alpha1.InstallModeType, []string, error) {
	modeSplit := strings.Split(raw, "=")
	if allNs := string(operatorsv1alpha1.InstallModeTypeAllNamespaces); raw == allNs || modeSplit[0] == allNs {
		return operatorsv1alpha1.InstallModeTypeAllNamespaces, nil, nil
	}
	if ownNs := string(operatorsv1alpha1.InstallModeTypeOwnNamespace); raw == ownNs || modeSplit[0] == ownNs {
		return operatorsv1alpha1.InstallModeTypeOwnNamespace, []string{operatorNs}, nil
	}
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
func validateInstallModeForNamespaces(mode operatorsv1alpha1.InstallModeType, namespaces []string) error {
	switch mode {
	case operatorsv1alpha1.InstallModeTypeOwnNamespace, operatorsv1alpha1.InstallModeTypeSingleNamespace:
		if len(namespaces) != 1 || namespaces[0] == "" {
			return fmt.Errorf("installMode %s must be passed with exactly one non-empty namespace, have: %+q",
				mode, namespaces)
		}
	case operatorsv1alpha1.InstallModeTypeMultiNamespace:
		if len(namespaces) < 2 {
			return fmt.Errorf("installMode %s must be passed with more than one non-empty namespaces, have: %+q",
				mode, namespaces)
		}
	case operatorsv1alpha1.InstallModeTypeAllNamespaces:
		if len(namespaces) != 0 && namespaces[0] != "" {
			return fmt.Errorf("installMode %s must be passed with no namespaces, have: %+q",
				mode, namespaces)
		}
	default:
		return fmt.Errorf("installMode %q is not a valid installMode type", mode)
	}
	return nil
}

// createOperatorGroup creates an OperatorGroup using pkgName if an OperatorGroup does not exist.
// If one exists in the desired namespace and it's target namespaces do not match the desired set,
// createOperatorGroup will return an error.
func (m *packageManifestsManager) createOperatorGroup(ctx context.Context, pkgName string) error {
	// Check OperatorGroup existence, since we cannot create a second OperatorGroup in namespace.
	og, ogFound, err := getOperatorGroup(ctx, m.client, m.namespace)
	if err != nil {
		return err
	}
	if ogFound {
		// Simple check for OperatorGroup compatibility: if namespaces are not an exact match,
		// the user must manage the resource themselves.
		sort.Strings(og.Status.Namespaces)
		sort.Strings(m.targetNamespaces)
		if !reflect.DeepEqual(og.Status.Namespaces, m.targetNamespaces) {
			msg := fmt.Sprintf("namespaces %+q do not match desired namespaces %+q", og.Status.Namespaces, m.targetNamespaces)
			if og.GetName() == operator.SDKOperatorGroupName {
				return fmt.Errorf("existing SDK-managed operator group's %s, "+
					"please clean up existing operators `operator-sdk cleanup` before running package %q", msg, pkgName)
			}
			return fmt.Errorf("existing operator group %q's %s, "+
				"please ensure it has the exact namespace set before running package %q", og.GetName(), msg, pkgName)
		}
		log.Infof("  Using existing operator group %q", og.GetName())
	} else {
		// New SDK-managed OperatorGroup.
		og = newSDKOperatorGroup(m.namespace, withTargetNamespaces(m.targetNamespaces...))
		if err = m.client.DoCreate(ctx, og); err != nil {
			return fmt.Errorf("error creating operator resources: %w", err)
		}
	}
	return nil
}

// getOperatorGroup returns true if an operator group in namespace was found and that operator group.
// If more than one operator group exists in namespace, this function will return an error
// since CSVs in namespace will have an error status in that case.
func getOperatorGroup(ctx context.Context, c *olmclient.Client, namespace string) (*operatorsv1.OperatorGroup, bool, error) {
	ogList := &operatorsv1.OperatorGroupList{}
	if err := c.KubeClient.List(ctx, ogList, client.InNamespace(namespace)); err != nil {
		return nil, false, err
	}
	if len(ogList.Items) == 0 {
		return nil, false, nil
	}
	if len(ogList.Items) != 1 {
		var names []string
		for _, og := range ogList.Items {
			names = append(names, og.GetName())
		}
		return nil, true, fmt.Errorf("more than one operator group in namespace %s: %+q", namespace, names)
	}
	return &ogList.Items[0], true, nil
}
