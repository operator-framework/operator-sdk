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

// Package olm provides an API to install, uninstall, and check the
// status of an Operator Lifecycle Manager installation.
// TODO: move to OLM repository?
package client

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/blang/semver/v4"
	olmapiv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	log "github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	deploymentutil "k8s.io/kubectl/pkg/util/deployment"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

var ErrOLMNotInstalled = errors.New("no existing installation found")

var Scheme = scheme.Scheme

// custom error struct to capture deployment errors
// while verifying CSV installs.
type resourceError struct {
	name  string
	issue string
}
type podError struct {
	resourceError
}
type deploymentError struct {
	resourceError
	podErrs podErrors
}
type deploymentErrors []deploymentError
type podErrors []podError

func (e deploymentErrors) Error() string {
	var sb strings.Builder
	for _, i := range e {
		sb.WriteString(fmt.Sprintf("deployment %s has error: %s\n%s", i.name, i.issue, i.podErrs.Error()))
	}
	return sb.String()
}

func (e podErrors) Error() string {
	var sb strings.Builder
	for _, i := range e {
		sb.WriteString(fmt.Sprintf("\tpod %s has error: %s\n", i.name, i.issue))
	}
	return sb.String()
}

func init() {
	if err := olmapiv1alpha1.AddToScheme(Scheme); err != nil {
		log.Fatalf("Failed to add OLM operator API v1alpha1 types to scheme: %v", err)
	}
}

type Client struct {
	KubeClient client.Client
}

func NewClientForConfig(cfg *rest.Config) (*Client, error) {
	rm, err := apiutil.NewDynamicRESTMapper(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic rest mapper: %v", err)
	}

	cl, err := client.New(cfg, client.Options{
		Scheme: Scheme,
		Mapper: rm,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %v", err)
	}

	c := &Client{
		KubeClient: cl,
	}
	return c, nil
}

func (c Client) DoCreate(ctx context.Context, objs ...client.Object) error {
	for _, obj := range objs {
		kind := obj.GetObjectKind().GroupVersionKind().Kind
		log.Infof("  Creating %s %q", kind, getName(obj.GetNamespace(), obj.GetName()))
		err := c.KubeClient.Create(ctx, obj)
		if err != nil {
			if !apierrors.IsAlreadyExists(err) {
				return err
			}
			log.Infof("    %s %q already exists", kind, getName(obj.GetNamespace(), obj.GetName()))
		}
	}
	return nil
}

func (c Client) DoDelete(ctx context.Context, objs ...client.Object) error {
	for _, obj := range objs {
		kind := obj.GetObjectKind().GroupVersionKind().Kind
		log.Infof("  Deleting %s %q", kind, getName(obj.GetNamespace(), obj.GetName()))
		err := c.KubeClient.Delete(ctx, obj, client.PropagationPolicy(metav1.DeletePropagationBackground))
		if err != nil {
			if !apierrors.IsNotFound(err) {
				return err
			}
			log.Infof("    %s %q does not exist", kind, getName(obj.GetNamespace(), obj.GetName()))
		}
		key := client.ObjectKeyFromObject(obj)
		if err := wait.PollImmediateUntil(time.Millisecond*100, func() (bool, error) {
			err := c.KubeClient.Get(ctx, key, obj)
			if apierrors.IsNotFound(err) {
				return true, nil
			} else if err != nil {
				return false, err
			}
			return false, nil
		}, ctx.Done()); err != nil {
			return err
		}
	}
	return nil
}

func getName(namespace, name string) string {
	if namespace != "" {
		name = fmt.Sprintf("%s/%s", namespace, name)
	}
	return name
}

func (c Client) DoRolloutWait(ctx context.Context, key types.NamespacedName) error {
	onceNotFound := sync.Once{}
	onceReplicasUpdated := sync.Once{}
	oncePendingTermination := sync.Once{}
	onceNotAvailable := sync.Once{}
	onceSpecUpdate := sync.Once{}

	rolloutComplete := func() (bool, error) {
		deployment := appsv1.Deployment{}
		err := c.KubeClient.Get(ctx, key, &deployment)
		if err != nil {
			if apierrors.IsNotFound(err) {
				onceNotFound.Do(func() {
					log.Printf("  Waiting for Deployment %q to appear", key)
				})
				return false, nil
			}
			return false, err
		}
		if deployment.Generation <= deployment.Status.ObservedGeneration {
			cond := deploymentutil.GetDeploymentCondition(deployment.Status, appsv1.DeploymentProgressing)
			if cond != nil && cond.Reason == deploymentutil.TimedOutReason {
				return false, errors.New("progress deadline exceeded")
			}
			if deployment.Spec.Replicas != nil && deployment.Status.UpdatedReplicas < *deployment.Spec.Replicas {
				onceReplicasUpdated.Do(func() {
					log.Printf(
						"  Waiting for Deployment %q to rollout: %d out of %d new replicas have been updated",
						key, deployment.Status.UpdatedReplicas, *deployment.Spec.Replicas)
				})
				return false, nil
			}
			if deployment.Status.Replicas > deployment.Status.UpdatedReplicas {
				oncePendingTermination.Do(func() {
					log.Printf("  Waiting for Deployment %q to rollout: %d old replicas are pending termination",
						key, deployment.Status.Replicas-deployment.Status.UpdatedReplicas)
				})
				return false, nil
			}
			if deployment.Status.AvailableReplicas < deployment.Status.UpdatedReplicas {
				onceNotAvailable.Do(func() {
					log.Printf("  Waiting for Deployment %q to rollout: %d of %d updated replicas are available",
						key, deployment.Status.AvailableReplicas, deployment.Status.UpdatedReplicas)
				})
				return false, nil
			}
			log.Printf("  Deployment %q successfully rolled out", key)
			return true, nil
		}
		onceSpecUpdate.Do(func() {
			log.Printf("Waiting for Deployment %q to rollout: waiting for deployment spec update to be observed",
				key)
		})
		return false, nil
	}
	return wait.PollImmediateUntil(time.Second, rolloutComplete, ctx.Done())
}

func (c Client) DoCSVWait(ctx context.Context, key types.NamespacedName) error {
	var (
		curPhase olmapiv1alpha1.ClusterServiceVersionPhase
		newPhase olmapiv1alpha1.ClusterServiceVersionPhase
	)
	once := sync.Once{}

	csv := olmapiv1alpha1.ClusterServiceVersion{}
	csvPhaseSucceeded := func() (bool, error) {
		err := c.KubeClient.Get(ctx, key, &csv)
		if err != nil {
			if apierrors.IsNotFound(err) {
				once.Do(func() {
					log.Printf("  Waiting for ClusterServiceVersion %q to appear", key)
				})
				return false, nil
			}
			return false, err
		}
		newPhase = csv.Status.Phase
		if newPhase != curPhase {
			curPhase = newPhase
			log.Printf("  Found ClusterServiceVersion %q phase: %s", key, curPhase)
		}

		switch curPhase {
		case olmapiv1alpha1.CSVPhaseFailed:
			return false, fmt.Errorf("csv failed: reason: %q, message: %q", csv.Status.Reason, csv.Status.Message)
		case olmapiv1alpha1.CSVPhaseSucceeded:
			return true, nil
		default:
			return false, nil
		}
	}

	err := wait.PollImmediateUntil(time.Second, csvPhaseSucceeded, ctx.Done())
	if err != nil && errors.Is(err, context.DeadlineExceeded) {
		depCheckErr := c.checkDeploymentErrors(ctx, key, csv)
		if depCheckErr != nil {
			return depCheckErr
		}
	}
	return err
}

// checkDeploymentErrors function loops through deployment specs of a given CSV, and prints reason
// in case of failures, based on deployment condition.
func (c Client) checkDeploymentErrors(ctx context.Context, key types.NamespacedName, csv olmapiv1alpha1.ClusterServiceVersion) error {
	depErrs := deploymentErrors{}
	if key.Namespace == "" {
		return fmt.Errorf("no namespace provided to get deployment failures")
	}
	dep := &appsv1.Deployment{}
	for _, ds := range csv.Spec.InstallStrategy.StrategySpec.DeploymentSpecs {
		depKey := types.NamespacedName{
			Namespace: key.Namespace,
			Name:      ds.Name,
		}
		depSelectors := ds.Spec.Selector
		if err := c.KubeClient.Get(ctx, depKey, dep); err != nil {
			depErrs = append(depErrs, deploymentError{
				resourceError: resourceError{
					name:  ds.Name,
					issue: err.Error(),
				},
			})
			continue
		}
		for _, s := range dep.Status.Conditions {
			if s.Type == appsv1.DeploymentAvailable && s.Status != corev1.ConditionTrue {
				depErr := deploymentError{
					resourceError: resourceError{
						name:  ds.Name,
						issue: s.Reason,
					},
				}
				podErr := c.checkPodErrors(ctx, depSelectors, key)
				podErrs := podErrors{}
				if errors.As(podErr, &podErrs) {
					depErr.podErrs = append(depErr.podErrs, podErrs...)
				} else {
					return podErr
				}
				depErrs = append(depErrs, depErr)
			}
		}
	}
	return depErrs
}

// checkPodErrors loops through pods, and returns pod errors if any.
func (c Client) checkPodErrors(ctx context.Context, depSelectors *metav1.LabelSelector, key types.NamespacedName) error {
	// loop through pods and return specific error message.
	podErr := podErrors{}
	podList := &corev1.PodList{}
	podLabelSelectors, err := metav1.LabelSelectorAsSelector(depSelectors)
	if err != nil {
		return err
	}
	options := client.ListOptions{
		LabelSelector: podLabelSelectors,
		Namespace:     key.Namespace,
	}
	if err := c.KubeClient.List(ctx, podList, &options); err != nil {
		return fmt.Errorf("error getting Pods: %v", err)
	}
	for _, p := range podList.Items {
		for _, cs := range p.Status.ContainerStatuses {
			if !cs.Ready {
				if cs.State.Waiting != nil {
					containerName := p.Name + ":" + cs.Name
					podErr = append(podErr, podError{
						resourceError{
							name:  containerName,
							issue: cs.State.Waiting.Message,
						},
					})
				}
			}
		}
	}
	return podErr
}

// GetInstalledVersion returns the OLM version installed in the namespace informed.
func (c Client) GetInstalledVersion(ctx context.Context, namespace string) (string, error) {
	opts := client.InNamespace(namespace)
	csvs := &olmapiv1alpha1.ClusterServiceVersionList{}
	if err := c.KubeClient.List(ctx, csvs, opts); err != nil {
		if apierrors.IsNotFound(err) || meta.IsNoMatchError(err) {
			return "", ErrOLMNotInstalled
		}
		return "", fmt.Errorf("failed to list CSVs in namespace %q: %v", namespace, err)
	}
	var pkgServerCSV *olmapiv1alpha1.ClusterServiceVersion
	for i := range csvs.Items {
		csv := csvs.Items[i]
		name := csv.GetName()
		// Check old and new name possibilities.
		if name == pkgServerCSVNewName || strings.HasPrefix(name, pkgServerCSVOldNamePrefix) {
			// There is more than one version of OLM installed in the cluster,
			// so we can't resolve the version being used.
			if pkgServerCSV != nil {
				return "", fmt.Errorf("more than one OLM (package server) version installed: %q and %q",
					pkgServerCSV.GetName(), name)
			}
			pkgServerCSV = &csv
		}
	}
	if pkgServerCSV == nil {
		return "", ErrOLMNotInstalled
	}
	return getOLMVersionFromPackageServerCSV(pkgServerCSV)
}

const (
	// Versions pre-0.11 have a versioned name.
	pkgServerCSVOldNamePrefix = "packageserver."
	// Versions 0.11+ have a fixed name.
	pkgServerCSVNewName      = "packageserver"
	pkgServerOLMVersionLabel = "olm.version"
)

func getOLMVersionFromPackageServerCSV(csv *olmapiv1alpha1.ClusterServiceVersion) (string, error) {
	// Package server CSV's from OLM versions > 0.10.1 have a label containing
	// the OLM version.
	if labels := csv.GetLabels(); labels != nil {
		if ver, ok := labels[pkgServerOLMVersionLabel]; ok {
			return ver, nil
		}
	}
	// Fall back to getting OLM version from package server CSV name. Versions
	// of OLM <= 0.10.1 are not labelled with pkgServerOLMVersionLabel.
	ver := strings.TrimPrefix(csv.GetName(), pkgServerCSVOldNamePrefix)
	// OLM releases do not have a "v" prefix but CSV versions do.
	ver = strings.TrimPrefix(ver, "v")
	// Check if a valid semver. Ignore non-nil errors as they are not related
	// to the reason OLM version can't be found.
	if _, err := semver.Parse(ver); err == nil {
		return ver, nil
	}
	return "", fmt.Errorf("no OLM version found in CSV %q spec", csv.GetName())
}
