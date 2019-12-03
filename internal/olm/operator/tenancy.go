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
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"

	registryutil "github.com/operator-framework/operator-sdk/internal/util/operator-registry"

	olmapiv1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1"
	olmapiv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// Annotation containing a CSV's member OperatorGroup's name.
	olmOperatorGroupAnnotation = "olm.operatorGroup"
	// Annotation containing a CSV's member OperatorGroup's namespace.
	olmOperatorGroupNamespaceAnnotation = "olm.operatorNamespace"
)

// operatorGroupDown handles logic to decide whether the SDK-managed
// OperatorGroup can be created.
//
// Check if an OperatorGroup needs to be created in m.namespace first. If
// there is and another OpreatorGroup is created, CSV installation will fail
// with reason TooManyOperatorGroups. If the CSV's installModes don't support
// the target namespace selection of the OperatorGroup, the CSV will fail
// with UnsupportedOperatorGroup.
//
// https://github.com/operator-framework/operator-lifecycle-manager/blob/master/doc/design/operatorgroups.md
func (m *operatorManager) operatorGroupUp(ctx context.Context) error {
	og, err := m.getOperatorGroupInNamespace(ctx, m.namespace)
	if err != nil {
		return err
	}
	sdkOG := newSDKOperatorGroup(m.namespace,
		withTargetNamespaces(m.installModeNamespaces...))
	if og == nil {
		m.olmObjects = append(m.olmObjects, sdkOG)
	} else {
		// An exactly matching set of namespaces is needed to ensure the operator
		// is deployed only in namespaces specified by the user.
		sort.Strings(og.Status.Namespaces)
		sort.Strings(m.installModeNamespaces)
		if reflect.DeepEqual(og.Status.Namespaces, m.installModeNamespaces) {
			log.Printf("  Using existing OperatorGroup %q in namespace %q", sdkOperatorGroupName, m.namespace)
		} else if og.GetName() == sdkOperatorGroupName {
			// operator-sdk manages this OperatorGroup, so we can modify it.
			ogCSVs, err := m.getCSVsInOperatorGroup(ctx, og.GetName(), m.namespace)
			if err != nil {
				return err
			}
			if len(ogCSVs) != 0 {
				fmt.Printf("OperatorGroup %q in namespace %q has existing member CSVs. "+
					"You may merge currently selected namespaces (1) with new namespaces (2):\n"+
					"(1) %+q\n(2) %+q\n"+
					"Doing so may transition installed CSVs into a Failed state with reason "+
					"UnsupportedOperatorGroup or have other unwanted side-effects.\n"+
					"Proceed? [y/N] ", og.GetName(), m.namespace, og.Status.Namespaces,
					m.installModeNamespaces)
				cli := bufio.NewReader(os.Stdin)
				resp, err := cli.ReadString('\n')
				if err != nil {
					return err
				}
				if resp = strings.TrimSpace(resp); resp != "y" && resp != "Y" {
					fmt.Printf("Not merging namespaces. Please modify the existing OperatorGroup "+
						"%q in namespace %q manually, or create this operator in a new namespace.\n",
						og.GetName(), m.namespace)
					os.Exit(0)
				}
				// All namespaces are used.
				sdkOG.Spec.TargetNamespaces = mergeNamespaces(
					sdkOG.Spec.TargetNamespaces, og.Status.Namespaces)
			}
			// Simple overwrite patch. Use merge patch type to avoid having to
			// construct a JSON patch.
			data, err := json.Marshal(sdkOG)
			if err != nil {
				return err
			}
			patch := client.ConstantPatch(types.MergePatchType, data)
			log.Printf("  Patching less permissive OperatorGroup %q in namespace %q", sdkOperatorGroupName, m.namespace)
			// Overwrite existing set of namespaces.
			err = m.client.KubeClient.Patch(ctx, sdkOG, patch)
			if err != nil {
				return err
			}
		} else {
			// operator-sdk does not own this OperatorGroup, cannot modify.
			return errors.Errorf("existing OperatorGroup %q in namespace %q does not"+
				" select all namespaces in %+q", og.GetName(), m.namespace, m.installModeNamespaces)
		}
	}
	return nil
}

func mergeNamespaces(set1, set2 []string) (result []string) {
	allNS := map[string]struct{}{}
	for _, ns := range append(set1, set2...) {
		if _, ok := allNS[ns]; !ok {
			result = append(result, ns)
			allNS[ns] = struct{}{}
		}
	}
	return result
}

// operatorGroupDown handles logic to decide whether the SDK-managed
// OperatorGroup can be deleted.
func (m *operatorManager) operatorGroupDown(ctx context.Context) error {
	// Check if OperatorGroup was created by operator-sdk before
	// deleting. We do not want to delete a pre-existing OperatorGroup, or
	// one that is used by existing CSVs.
	og, err := m.getOperatorGroupInNamespace(ctx, m.namespace)
	if err != nil {
		return err
	}
	if og != nil && og.GetName() == sdkOperatorGroupName {
		ogCSVs, err := m.getCSVsInOperatorGroup(ctx, sdkOperatorGroupName, m.namespace)
		if err != nil {
			return err
		}
		bundle, err := m.manifests.GetBundleForVersion(m.version)
		if err != nil {
			return err
		}
		csv, err := bundle.ClusterServiceVersion()
		if err != nil {
			return err
		}
		if len(ogCSVs) == 0 || (len(ogCSVs) == 1 && ogCSVs[0].GetName() == csv.GetName()) {
			m.olmObjects = append(m.olmObjects, newSDKOperatorGroup(m.namespace))
		} else {
			log.Infof("  Existing OperatorGroup %q in namespace %q is used by existing CSVs, skipping delete", og.GetName(), m.namespace)
		}
	}
	return nil
}

// getCSVsInOperatorGroup gets all CSVs that are members of the OperatorGroup
// in namespace with name ogName. If ogCSVs is empty, no CSVs are members.
func (m operatorManager) getCSVsInOperatorGroup(ctx context.Context, ogName, namespace string) (ogCSVs []*olmapiv1alpha1.ClusterServiceVersion, err error) {
	csvs := olmapiv1alpha1.ClusterServiceVersionList{}
	opt := client.InNamespace(namespace)
	err = m.client.KubeClient.List(ctx, &csvs, opt)
	if err != nil && !apierrors.IsNotFound(err) {
		return nil, err
	}
	for _, csv := range csvs.Items {
		annotations := csv.GetAnnotations()
		if annotations != nil {
			csvOGName, ogOK := annotations[olmOperatorGroupAnnotation]
			csvOGNamespace, nsOK := annotations[olmOperatorGroupNamespaceAnnotation]
			// TODO(estroz): ensure this works for "" (AllNamespaces).
			if ogOK && nsOK && csvOGName == ogName && csvOGNamespace == namespace {
				ogCSVs = append(ogCSVs, &csv)
			}
		}
	}
	return ogCSVs, nil
}

// installModeCompatible ensures installMode is compatible with the namespaces
// and CSV's installModes being used.
func (m operatorManager) installModeCompatible(installMode olmapiv1alpha1.InstallModeType) error {
	err := validateInstallModeWithNamespaces(installMode, m.installModeNamespaces)
	if err != nil {
		return err
	}
	if installMode == olmapiv1alpha1.InstallModeTypeOwnNamespace {
		if ns := m.installModeNamespaces[0]; ns != m.namespace {
			return errors.Errorf("installMode %s namespace %q must match namespace %q", installMode, ns, m.namespace)
		}
	}
	// Ensure CSV supports installMode.
	bundle, err := m.manifests.GetBundleForVersion(m.version)
	if err != nil {
		return err
	}
	csv, err := bundle.ClusterServiceVersion()
	if err != nil {
		return err
	}
	olmCSV := registryutil.MustBundleCSVToCSV(csv)
	for _, mode := range olmCSV.Spec.InstallModes {
		if mode.Type == installMode && !mode.Supported {
			return errors.Errorf("installMode %s not supported in CSV %q", installMode, olmCSV.GetName())
		}
	}
	return nil
}

// Mapping of installMode string values to types, for validation.
var installModeStrings = map[string]olmapiv1alpha1.InstallModeType{
	string(olmapiv1alpha1.InstallModeTypeOwnNamespace):    olmapiv1alpha1.InstallModeTypeOwnNamespace,
	string(olmapiv1alpha1.InstallModeTypeSingleNamespace): olmapiv1alpha1.InstallModeTypeSingleNamespace,
	string(olmapiv1alpha1.InstallModeTypeMultiNamespace):  olmapiv1alpha1.InstallModeTypeMultiNamespace,
	string(olmapiv1alpha1.InstallModeTypeAllNamespaces):   olmapiv1alpha1.InstallModeTypeAllNamespaces,
}

// parseInstallModeKV parses an installMode string of the format
// installModeFormat.
func parseInstallModeKV(raw string) (olmapiv1alpha1.InstallModeType, []string, error) {
	modeSplit := strings.Split(raw, "=")
	if len(modeSplit) != 2 {
		return "", nil, errors.Errorf("installMode string %q is malformatted, must be: %s", raw, installModeFormat)
	}
	modeStr, namespaceList := modeSplit[0], modeSplit[1]
	mode, ok := installModeStrings[modeStr]
	if !ok {
		return "", nil, errors.Errorf("installMode type string %q is not a valid installMode type", modeStr)
	}
	namespaces := []string{}
	for _, namespace := range strings.Split(strings.Trim(namespaceList, ","), ",") {
		namespaces = append(namespaces, namespace)
	}
	return mode, namespaces, nil
}

// validateInstallModeWithNamespaces ensures namespaces are valid given mode.
func validateInstallModeWithNamespaces(mode olmapiv1alpha1.InstallModeType, namespaces []string) error {
	switch mode {
	case olmapiv1alpha1.InstallModeTypeOwnNamespace, olmapiv1alpha1.InstallModeTypeSingleNamespace:
		if len(namespaces) != 1 || namespaces[0] == "" {
			return errors.Errorf("installMode %s must be passed with exactly one non-empty namespace, have: %+q", mode, namespaces)
		}
	case olmapiv1alpha1.InstallModeTypeMultiNamespace:
		if len(namespaces) < 2 {
			return errors.Errorf("installMode %s must be passed with more than one non-empty namespaces, have: %+q", mode, namespaces)
		}
	case olmapiv1alpha1.InstallModeTypeAllNamespaces:
		if len(namespaces) != 1 || namespaces[0] != "" {
			return errors.Errorf("installMode %s must be passed with exactly one empty namespace, have: %+q", mode, namespaces)
		}
	default:
		return errors.Errorf("installMode %q is not a valid installMode type", mode)
	}
	return nil
}

// getOperatorGroupInNamespace gets the OperatorGroup in namespace. Becuase
// there must only be one OperatorGroup per namespace, an error is returned
// if more than one is found. nil is returned if no OperatorGroup exists in
// namespace.
func (m operatorManager) getOperatorGroupInNamespace(ctx context.Context, namespace string) (*olmapiv1.OperatorGroup, error) {
	// There must only be one OperatorGroup per namespace, but we should use list.
	ogs := olmapiv1.OperatorGroupList{}
	err := m.client.KubeClient.List(ctx, &ogs, client.InNamespace(namespace))
	if err != nil && !apierrors.IsNotFound(err) {
		return nil, err
	}
	if apierrors.IsNotFound(err) || len(ogs.Items) == 0 {
		return nil, nil
	}
	// There should never be more than one, but if there is return an error.
	if len(ogs.Items) > 1 {
		return nil, errors.Errorf("more than one OperatorGroup exists in namespace %q", namespace)
	}
	currOG := &ogs.Items[0]
	return currOG, nil
}
