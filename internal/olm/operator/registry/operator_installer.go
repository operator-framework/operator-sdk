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

package registry

import (
	"context"
	"fmt"
	"time"

	v1 "github.com/operator-framework/api/pkg/operators/v1"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"

	olmclient "github.com/operator-framework/operator-sdk/internal/olm/client"
	"github.com/operator-framework/operator-sdk/internal/olm/operator"
)

type OperatorInstaller struct {
	CatalogSourceName     string
	PackageName           string
	StartingCSV           string
	Channel               string
	InstallMode           operator.InstallMode
	CatalogCreator        CatalogCreator
	SupportedInstallModes sets.String

	cfg *operator.Configuration
}

func NewOperatorInstaller(cfg *operator.Configuration) *OperatorInstaller {
	return &OperatorInstaller{cfg: cfg}
}

func (o OperatorInstaller) InstallOperator(ctx context.Context) (*v1alpha1.ClusterServiceVersion, error) {
	cs, err := o.CatalogCreator.CreateCatalog(ctx, o.CatalogSourceName)
	if err != nil {
		return nil, fmt.Errorf("create catalog: %v", err)
	}
	log.Infof("Created CatalogSource: %s", cs.GetName())

	// TODO: OLM doesn't appear to propagate the "READY" connection status to the
	// catalogsource in a timely manner even though its catalog-operator reports
	// a connection almost immediately. This condition either needs to be
	// propagated more quickly by OLM or we need to find a different resource to
	// probe for readiness.
	//
	// if err := o.waitForCatalogSource(ctx, cs); err != nil {
	// 	return nil, err
	// }

	// Ensure Operator Group
	if err = o.ensureOperatorGroup(ctx); err != nil {
		return nil, err
	}

	var subscription *v1alpha1.Subscription
	// Create Subscription
	if subscription, err = o.createSubscription(ctx, cs.GetName()); err != nil {
		return nil, err
	}

	// Wait for the Install Plan to be generated
	if err = o.waitForInstallPlan(ctx, subscription); err != nil {
		return nil, err
	}

	// Approve Install Plan for the subscription
	if err = o.approveInstallPlan(ctx, subscription); err != nil {
		return nil, err
	}

	// Wait for successfully installed CSV
	csv, err := o.getInstalledCSV(ctx)
	if err != nil {
		return nil, err
	}

	log.Infof("OLM has successfully installed %q", o.StartingCSV)

	return csv, nil
}

//nolint:unused
func (o OperatorInstaller) waitForCatalogSource(ctx context.Context, cs *v1alpha1.CatalogSource) error {
	catSrcKey, err := client.ObjectKeyFromObject(cs)
	if err != nil {
		return fmt.Errorf("error getting catalog source key: %v", err)
	}

	// verify that catalog source connection status is READY
	catSrcCheck := wait.ConditionFunc(func() (done bool, err error) {
		if err := o.cfg.Client.Get(ctx, catSrcKey, cs); err != nil {
			return false, err
		}
		if cs.Status.GRPCConnectionState != nil {
			if cs.Status.GRPCConnectionState.LastObservedState == "READY" {
				return true, nil
			}
		}
		return false, nil
	})

	if err := wait.PollImmediateUntil(200*time.Millisecond, catSrcCheck, ctx.Done()); err != nil {
		return fmt.Errorf("catalog source connection is not ready: %v", err)
	}

	return nil
}

func (o OperatorInstaller) ensureOperatorGroup(ctx context.Context) error {
	// Check OperatorGroup existence, since we cannot create a second OperatorGroup in namespace.
	og, ogFound, err := o.getOperatorGroup(ctx)
	if err != nil {
		return err
	}

	supported := o.SupportedInstallModes

	// --install-mode was given
	if !o.InstallMode.IsEmpty() {
		if o.InstallMode.InstallModeType == v1alpha1.InstallModeTypeSingleNamespace &&
			o.InstallMode.TargetNamespaces[0] == o.cfg.Namespace {
			return fmt.Errorf("use install mode %q to watch operator's namespace %q", v1alpha1.InstallModeTypeOwnNamespace, o.cfg.Namespace)
		}

		supported = supported.Intersection(sets.NewString(string(o.InstallMode.InstallModeType)))
		if supported.Len() == 0 {
			return fmt.Errorf("operator %q does not support install mode %q", o.StartingCSV, o.InstallMode.InstallModeType)
		}
	}

	targetNamespaces, err := o.getTargetNamespaces(supported)
	if err != nil {
		return err
	}

	if !ogFound {
		if og, err = o.createOperatorGroup(ctx, targetNamespaces); err != nil {
			return fmt.Errorf("create operator group: %v", err)
		}
		log.Infof("OperatorGroup %q created", og.Name)
	} else if err := o.isOperatorGroupCompatible(*og, targetNamespaces); err != nil {
		return err
	}

	return nil
}

func (o *OperatorInstaller) createOperatorGroup(ctx context.Context, targetNamespaces []string) (*v1.OperatorGroup, error) {
	og := newSDKOperatorGroup(o.cfg.Namespace, withTargetNamespaces(targetNamespaces...))
	if err := o.cfg.Client.Create(ctx, og); err != nil {
		return nil, err
	}
	return og, nil
}

func (o *OperatorInstaller) isOperatorGroupCompatible(og v1.OperatorGroup, targetNamespaces []string) error {
	// no install mode use the existing operator group
	if o.InstallMode.IsEmpty() {
		return nil
	}

	// otherwise, check that the target namespaces match
	targets := sets.NewString(targetNamespaces...)
	ogtargets := sets.NewString(og.Spec.TargetNamespaces...)
	if !ogtargets.Equal(targets) {
		return fmt.Errorf("existing operatorgroup %q is not compatible with install mode %q", og.Name, o.InstallMode)
	}

	return nil
}

// getOperatorGroup returns true if an OperatorGroup in the desired namespace was found.
// If more than one operator group exists in namespace, this function will return an error
// since CSVs in namespace will have an error status in that case.
func (o OperatorInstaller) getOperatorGroup(ctx context.Context) (*v1.OperatorGroup, bool, error) {
	ogList := &v1.OperatorGroupList{}
	if err := o.cfg.Client.List(ctx, ogList, client.InNamespace(o.cfg.Namespace)); err != nil {
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
		return nil, true, fmt.Errorf("more than one operator group in namespace %s: %+q", o.cfg.Namespace, names)
	}
	return &ogList.Items[0], true, nil
}

func (o OperatorInstaller) createSubscription(ctx context.Context, csName string) (*v1alpha1.Subscription, error) {
	sub := newSubscription(o.StartingCSV, o.cfg.Namespace,
		withPackageChannel(o.PackageName, o.Channel, o.StartingCSV),
		withCatalogSource(csName, o.cfg.Namespace),
		withInstallPlanApproval(v1alpha1.ApprovalManual))

	if err := o.cfg.Client.Create(ctx, sub); err != nil {
		return nil, fmt.Errorf("error creating subscription: %w", err)
	}
	log.Infof("Created Subscription: %s", sub.Name)

	return sub, nil
}

func (o OperatorInstaller) getInstalledCSV(ctx context.Context) (*v1alpha1.ClusterServiceVersion, error) {
	c, err := olmclient.NewClientForConfig(o.cfg.RESTConfig)
	if err != nil {
		return nil, err
	}

	// BUG(estroz): if namespace is not contained in targetNamespaces,
	// DoCSVWait will fail because the CSV is not deployed in namespace.
	nn := types.NamespacedName{
		Name:      o.StartingCSV,
		Namespace: o.cfg.Namespace,
	}
	log.Infof("Waiting for ClusterServiceVersion %q to reach 'Succeeded' phase", nn)
	if err = c.DoCSVWait(ctx, nn); err != nil {
		return nil, fmt.Errorf("error waiting for CSV to install: %w", err)
	}

	// TODO: check status of all resources in the desired bundle/package.
	csv := &v1alpha1.ClusterServiceVersion{}
	if err = o.cfg.Client.Get(ctx, nn, csv); err != nil {
		return nil, fmt.Errorf("error getting installed CSV: %w", err)
	}
	return csv, nil
}

// approveInstallPlan approves the install plan for a subscription, which will
// generate a CSV
func (o OperatorInstaller) approveInstallPlan(ctx context.Context, sub *v1alpha1.Subscription) error {
	ip := v1alpha1.InstallPlan{}

	ipKey := types.NamespacedName{
		Name:      sub.Status.InstallPlanRef.Name,
		Namespace: sub.Status.InstallPlanRef.Namespace,
	}

	if err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		if err := o.cfg.Client.Get(ctx, ipKey, &ip); err != nil {
			return fmt.Errorf("error getting install plan: %v", err)
		}
		// approve the install plan by setting Approved to true
		ip.Spec.Approved = true
		if err := o.cfg.Client.Update(ctx, &ip); err != nil {
			return fmt.Errorf("error approving install plan: %v", err)
		}
		return nil
	}); err != nil {
		return err
	}

	log.Infof("Approved InstallPlan %s for the Subscription: %s", ipKey.Name, sub.Name)

	return nil
}

// waitForInstallPlan verifies if an Install Plan exists through subscription status
func (o OperatorInstaller) waitForInstallPlan(ctx context.Context, sub *v1alpha1.Subscription) error {
	subKey := types.NamespacedName{
		Namespace: sub.GetNamespace(),
		Name:      sub.GetName(),
	}

	ipCheck := wait.ConditionFunc(func() (done bool, err error) {
		if err := o.cfg.Client.Get(ctx, subKey, sub); err != nil {
			return false, err
		}
		if sub.Status.InstallPlanRef != nil {
			return true, nil
		}
		return false, nil
	})

	if err := wait.PollImmediateUntil(200*time.Millisecond, ipCheck, ctx.Done()); err != nil {
		return fmt.Errorf("install plan is not available for the subscription %s: %v", sub.Name, err)
	}
	return nil
}

func (o *OperatorInstaller) getTargetNamespaces(supported sets.String) ([]string, error) {
	switch {
	case supported.Has(string(v1alpha1.InstallModeTypeAllNamespaces)):
		return nil, nil
	case supported.Has(string(v1alpha1.InstallModeTypeOwnNamespace)):
		return []string{o.cfg.Namespace}, nil
	case supported.Has(string(v1alpha1.InstallModeTypeSingleNamespace)):
		return o.InstallMode.TargetNamespaces, nil
	default:
		return nil, fmt.Errorf("no supported install modes")
	}
}
