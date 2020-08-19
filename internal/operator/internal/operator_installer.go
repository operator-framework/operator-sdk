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

	v1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	internalolm "github.com/operator-framework/operator-sdk/internal/olm/operator"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/operator-framework/operator-sdk/internal/operator"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

// InstallPlanApproval - type of InstallPlan approval for the subscription
type InstallPlanApproval = string

const (
	// ManualApproval is the manual install plan approval
	ManualApproval InstallPlanApproval = "Manual"
	// AutomaticApproval is the automatic install plan approval
	AutomaticApproval InstallPlanApproval = "Automatic"
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

	// Verify that InstallPlan was successfully generated for the subscription
	// through subscription status
	err = o.verifyInstallPlanGeneration(ctx, sub)
	if err != nil {
		return nil, fmt.Errorf("error in verifying install plan: %v", err)
	}

	// Approve Install Plan (if necessary)

	// Wait for successfully installed CSV

	return todo, nil
}

func (o OperatorInstaller) createSubscription() *v1alpha1.Subscription {
	sub := internalolm.NewSubscription(o.CatalogSourceName, o.cfg.Namespace,
		withCatalogSource(o.CatalogSourceName, o.cfg.Namespace),
		withBundleChannel(o.PackageName, o.Channel, o.StartingCSV),
		withInstallPlanApproval(ManualApproval))

	fmt.Printf("Creating Subscription: %s", sub.Name)
	return sub
}

type subscriptionOption func(*v1alpha1.Subscription)

func withCatalogSource(catSrcName, catSrcNamespace string) subscriptionOption {
	return func(sub *v1alpha1.Subscription) {
		if sub.Spec == nil {
			sub.Spec = &v1alpha1.SubscriptionSpec{}
		}
		sub.Spec.CatalogSource = catSrcName
		sub.Spec.CatalogSourceNamespace = catSrcNamespace

	}
}

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

func withInstallPlanApproval(approval string) subscriptionOption {
	return func(sub *v1alpha1.Subscription) {
		if sub.Spec == nil {
			sub.Spec = &v1alpha1.SubscriptionSpec{}
		}
		// set the install plan approval to manual
		sub.Spec.InstallPlanApproval = v1alpha1.Approval(approval)
	}
}

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
