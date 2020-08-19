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
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	internalolm "github.com/operator-framework/operator-sdk/internal/olm/operator"
	"github.com/operator-framework/operator-sdk/internal/operator"
)

type OperatorInstaller struct {
	CatalogSourceName string
	PackageName       string
	StartingCSV       string
	Channel           string
	InstallMode       InstallMode
	CatalogCreator    CatalogCreator

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

	_ = cs

	fmt.Printf("OperatorInstaller.CatalogSourceName: %q\n", o.CatalogSourceName)
	fmt.Printf("OperatorInstaller.PackageName:       %q\n", o.PackageName)
	fmt.Printf("OperatorInstaller.StartingCSV:       %q\n", o.StartingCSV)
	fmt.Printf("OperatorInstaller.Channel:           %q\n", o.Channel)
	fmt.Printf("OperatorInstaller.InstallMode:       %q\n", o.InstallMode)
	todo := &v1alpha1.ClusterServiceVersion{
		ObjectMeta: metav1.ObjectMeta{Name: o.StartingCSV},
	}

	// Ensure Operator Group

	// Create Subscription
	sub := o.createSubscription()

	// Verify that InstallPlan was successfully generated for the
	// subscription through status
	err = o.verifyInstallPlanGeneration(ctx, sub)
	if err != nil {
		return nil, fmt.Errorf("error in verifying install plan: %v", err)
	}

	ipKey := types.NamespacedName{
		Name:      sub.Status.InstallPlanRef.Name,
		Namespace: sub.Status.InstallPlanRef.Namespace,
	}

	// Get Install Plan for the subscription
	ip, err := o.getInstallPlan(ctx, ipKey)
	if err != nil {
		return nil, fmt.Errorf("error in fetching the install plan: %v", err)
	}

	// Update the Install Plan for CSV generation
	err = o.updateInstallPlan(ctx, ip, withApproval(true))
	if err != nil {
		return nil, fmt.Errorf("error in setting install plan approval: %v", err)
	}

	// Wait for successfully installed CSV

	return todo, nil
}

// createSubscription creates a new subscription for a catalog
func (o OperatorInstaller) createSubscription() *v1alpha1.Subscription {
	sub := internalolm.NewSubscription(o.CatalogSourceName, o.cfg.Namespace,
		withCatalogSource(o.CatalogSourceName, o.cfg.Namespace),
		withBundleChannel(o.PackageName, o.Channel, o.StartingCSV),
		withInstallPlanApproval(v1alpha1.ApprovalManual))

	fmt.Printf("Creating Subscription: %s", sub.Name)
	return sub
}

type subscriptionOption func(*v1alpha1.Subscription)

// withCatalogSource sets the catalog source name and namespace for a subscription
func withCatalogSource(catSrcName, catSrcNamespace string) subscriptionOption {
	return func(sub *v1alpha1.Subscription) {
		if sub.Spec == nil {
			sub.Spec = &v1alpha1.SubscriptionSpec{}
		}
		sub.Spec.CatalogSource = catSrcName
		sub.Spec.CatalogSourceNamespace = catSrcNamespace

	}
}

// withBundleChannel sets the package name, channel name and starting CSV for a subscription
func withBundleChannel(packageName, channelName, startingCSV string) subscriptionOption {
	return func(sub *v1alpha1.Subscription) {
		if sub.Spec == nil {
			sub.Spec = &v1alpha1.SubscriptionSpec{}
		}
		sub.Spec.Package = packageName
		sub.Spec.Channel = channelName
		sub.Spec.StartingCSV = startingCSV
	}
}

// withInstallPlanApproval sets the subscription's install plan approval to manual
func withInstallPlanApproval(approval v1alpha1.Approval) subscriptionOption {
	return func(sub *v1alpha1.Subscription) {
		if sub.Spec == nil {
			sub.Spec = &v1alpha1.SubscriptionSpec{}
		}
		// set the install plan approval to manual
		sub.Spec.InstallPlanApproval = approval
	}
}

// verifyInstallPlanGeneration verifies if an Install Plan exists through subscription status
func (o OperatorInstaller) verifyInstallPlanGeneration(ctx context.Context, sub *v1alpha1.Subscription) error {
	subKey, err := client.ObjectKeyFromObject(sub)
	if err != nil {
		return fmt.Errorf("error in getting subscription key: %v", err)
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

// getInstallPlan returns the install plan
func (o OperatorInstaller) getInstallPlan(ctx context.Context, ipKey types.NamespacedName) (*v1alpha1.InstallPlan, error) {
	var ip *v1alpha1.InstallPlan

	err := o.cfg.Client.Get(ctx, ipKey, ip)
	if err != nil {
		return nil, fmt.Errorf("error in getting install plan: %v", err)
	}
	return ip, nil
}

type installPlanOption func(*v1alpha1.InstallPlan)

// withApproval sets the approved field to true
func withApproval(approval bool) installPlanOption {
	return func(ip *v1alpha1.InstallPlan) {
		// approve the install plan by setting Approved to true
		ip.Spec.Approved = true
	}
}

// updateInstallPlan updates the install plan by setting the approval to true
func (o OperatorInstaller) updateInstallPlan(ctx context.Context, ip *v1alpha1.InstallPlan, opts ...installPlanOption) error {
	for _, opt := range opts {
		opt(ip)
	}

	err := o.cfg.Client.Update(ctx, ip)
	if err != nil {
		return fmt.Errorf("error in approving install plan: %v", err)
	}

	return nil
}
