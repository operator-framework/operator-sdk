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

package registry

import (
	"context"
	"fmt"
	"time"

	v1 "github.com/operator-framework/api/pkg/operators/v1"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"

	"github.com/operator-framework/operator-sdk/internal/olm/operator"
	registrypod "github.com/operator-framework/operator-sdk/internal/olm/operator/registry/pod"
	"github.com/operator-framework/operator-sdk/internal/util/k8sutil"
)

func getSubscriptionName(csvName string) string {
	name := k8sutil.FormatOperatorNameDNS1123(csvName)
	return fmt.Sprintf("%s-sub", name)
}

// Update catalog source with source type as grpc, new registry pod address as the pod IP,
// and annotations from items and the pod.
func updateCatalogSource(ctx context.Context, cfg *operator.Configuration, cs *v1alpha1.CatalogSource, updateFunc func(*v1alpha1.CatalogSource)) error {
	key := types.NamespacedName{Namespace: cs.GetNamespace(), Name: cs.GetName()}
	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		if err := cfg.Client.Get(ctx, key, cs); err != nil {
			return fmt.Errorf("error getting catalog source: %w", err)
		}
		updateFunc(cs)
		if err := cfg.Client.Update(ctx, cs); err != nil {
			return fmt.Errorf("error updating catalog source: %w", err)
		}
		return nil
	})
}

// Use if no extra updates need to be made to an annotated CatalogSource.
func updateFieldsNoOp(*v1alpha1.CatalogSource) {}

// updateGRPCFieldsFunc updates cs' address and source type to a gRPC setting defined by targetPod.
func updateGRPCFieldsFunc(targetPod *corev1.Pod) func(*v1alpha1.CatalogSource) {
	return func(cs *v1alpha1.CatalogSource) {
		// set `spec.Address` and `spec.SourceType` as grpc
		cs.Spec.Address = registrypod.GetHostName(targetPod.Status.PodIP, defaultGRPCPort)
		cs.Spec.SourceType = v1alpha1.SourceTypeGrpc
	}
}

func deleteRegistryPod(ctx context.Context, cfg *operator.Configuration, podName string) error {
	// get registry pod key
	podKey := types.NamespacedName{
		Namespace: cfg.Namespace,
		Name:      podName,
	}

	pod := corev1.Pod{}
	podCheck := wait.ConditionFunc(func() (done bool, err error) {
		if err := cfg.Client.Get(ctx, podKey, &pod); err != nil {
			return false, fmt.Errorf("error getting previous registry pod %s: %w", podName, err)
		}
		return true, nil
	})

	if err := wait.PollImmediateUntil(200*time.Millisecond, podCheck, ctx.Done()); err != nil {
		return fmt.Errorf("error getting previous registry pod: %v", err)
	}

	if err := cfg.Client.Delete(ctx, &pod); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("delete %q: %v", pod.GetName(), err)
	} else if err == nil {
		log.Infof("Deleted previous registry pod with name %q", pod.GetName())
	}

	// Failure of the old pod to clean up should block and cause the caller to error out if it fails,
	// since the old pod may still be connected to OLM.
	if err := wait.PollImmediateUntil(200*time.Millisecond, func() (bool, error) {
		if err := cfg.Client.Get(ctx, podKey, &pod); apierrors.IsNotFound(err) {
			return true, nil
		} else if err != nil {
			return false, err
		}
		return false, nil
	}, ctx.Done()); err != nil {
		return fmt.Errorf("old registry pod %q failed to delete (%v), requires manual cleanup", pod.GetName(), err)
	}

	return nil
}

// withCatalogSource returns a function that sets the Subscription argument's
// target CatalogSource's name and namespace.
func withCatalogSource(csName, csNamespace string) func(*v1alpha1.Subscription) {
	return func(sub *v1alpha1.Subscription) {
		sub.Spec.CatalogSource = csName
		sub.Spec.CatalogSourceNamespace = csNamespace
	}
}

// withPackageChannel returns a function that sets the Subscription argument's
// target package, channel, and starting CSV to those in channel.
func withPackageChannel(pkgName, channelName, startingCSV string) func(*v1alpha1.Subscription) {
	return func(sub *v1alpha1.Subscription) {
		sub.Spec.Package = pkgName
		sub.Spec.Channel = channelName
		sub.Spec.StartingCSV = startingCSV
	}
}

// withInstallPlanApproval sets the Subscription's install plan approval field
// to manual
func withInstallPlanApproval(approval v1alpha1.Approval) func(*v1alpha1.Subscription) {
	return func(sub *v1alpha1.Subscription) {
		if sub.Spec == nil {
			sub.Spec = &v1alpha1.SubscriptionSpec{}
		}
		sub.Spec.InstallPlanApproval = approval
	}
}

// newSubscription creates a new Subscription for a CSV with a name derived
// from csvName, the CSV's objectmeta.name, in namespace. opts will be applied
// to the Subscription object.
func newSubscription(csvName, namespace string, opts ...func(*v1alpha1.Subscription)) *v1alpha1.Subscription {
	sub := &v1alpha1.Subscription{}
	sub.SetGroupVersionKind(v1alpha1.SchemeGroupVersion.WithKind(v1alpha1.SubscriptionKind))
	sub.SetName(getSubscriptionName(csvName))
	sub.SetNamespace(namespace)
	sub.Spec = &v1alpha1.SubscriptionSpec{}
	for _, opt := range opts {
		opt(sub)
	}
	return sub
}

func withSDKPublisher(pkgName string) func(*v1alpha1.CatalogSource) {
	return func(cs *v1alpha1.CatalogSource) {
		cs.Spec.DisplayName = pkgName
		cs.Spec.Publisher = "operator-sdk"
	}
}

// newCatalogSource creates a new CatalogSource with a name derived from
// pkgName, the package manifest's packageName, in namespace. opts will
// be applied to the CatalogSource object.
func newCatalogSource(name, namespace string, opts ...func(*v1alpha1.CatalogSource)) *v1alpha1.CatalogSource {
	cs := &v1alpha1.CatalogSource{}
	cs.SetGroupVersionKind(v1alpha1.SchemeGroupVersion.WithKind(v1alpha1.CatalogSourceKind))
	cs.SetName(name)
	cs.SetNamespace(namespace)
	for _, opt := range opts {
		opt(cs)
	}
	return cs
}

// withTargetNamespaces returns a function that sets the OperatorGroup argument's targetNamespaces to namespaces.
// namespaces can be length 0..N; if namespaces length is 0, targetNamespaces is unset, indicating a global scope.
func withTargetNamespaces(namespaces ...string) func(*v1.OperatorGroup) {
	return func(og *v1.OperatorGroup) {
		if len(namespaces) != 0 && namespaces[0] != "" {
			og.Spec.TargetNamespaces = namespaces
		}
	}
}

// newSDKOperatorGroup creates a new OperatorGroup with name
// sdkOperatorGroupName in namespace. opts will be applied to the
// OperatorGroup object. Note that the default OperatorGroup has a global
// scope.
func newSDKOperatorGroup(namespace string, opts ...func(*v1.OperatorGroup)) *v1.OperatorGroup {
	og := &v1.OperatorGroup{}
	og.SetGroupVersionKind(v1.SchemeGroupVersion.WithKind(v1.OperatorGroupKind))
	og.SetName(operator.SDKOperatorGroupName)
	og.SetNamespace(namespace)
	for _, opt := range opts {
		opt(og)
	}
	return og
}
