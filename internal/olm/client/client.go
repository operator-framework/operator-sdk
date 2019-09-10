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
package olm

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/operator-framework/operator-sdk/pkg/restmapper"

	"github.com/blang/semver"
	olmapiv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	deploymentutil "k8s.io/kubernetes/pkg/controller/deployment/util"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const OLMNamespace = "olm"

var ErrOLMNotInstalled = errors.New("no existing installation found")

var Scheme = scheme.Scheme

func init() {
	if err := olmapiv1alpha1.AddToScheme(Scheme); err != nil {
		log.Fatalf("Failed to add OLM operator API v1alpha1 types to scheme: %v", err)
	}
}

type Client struct {
	KubeClient client.Client
}

func ClientForConfig(cfg *rest.Config) (*Client, error) {
	rm, err := restmapper.NewDynamicRESTMapper(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create dynamic rest mapper")
	}

	cl, err := client.New(cfg, client.Options{
		Scheme: Scheme,
		Mapper: rm,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create client")
	}

	c := &Client{
		KubeClient: cl,
	}
	return c, nil
}

func (c Client) DoCreate(ctx context.Context, objs ...runtime.Object) error {
	for _, obj := range objs {
		a, err := meta.Accessor(obj)
		if err != nil {
			return err
		}
		kind := obj.GetObjectKind().GroupVersionKind().Kind
		log.Infof("  Creating %s %q", kind, getName(a.GetNamespace(), a.GetName()))
		err = c.KubeClient.Create(ctx, obj)
		if err != nil {
			if apierrors.IsAlreadyExists(err) {
				log.Infof("    %s %q already exists", kind, getName(a.GetNamespace(), a.GetName()))
				return nil
			}
			return err
		}
	}
	return nil
}

func (c Client) DoDelete(ctx context.Context, objs ...runtime.Object) error {
	for _, obj := range objs {
		a, err := meta.Accessor(obj)
		if err != nil {
			return err
		}
		kind := obj.GetObjectKind().GroupVersionKind().Kind
		log.Infof("  Deleting %s %q", kind, getName(a.GetNamespace(), a.GetName()))
		err = c.KubeClient.Delete(ctx, obj, client.PropagationPolicy(metav1.DeletePropagationForeground))
		if err != nil {
			if apierrors.IsNotFound(err) {
				log.Infof("    %s %q does not exist", kind, getName(a.GetNamespace(), a.GetName()))
				continue
			}
			return err
		}
	}

	log.Infof("  Waiting for deleted resources to disappear")
	wait.PollImmediateUntil(time.Second, func() (bool, error) {
		s := c.GetObjectsStatus(ctx, objs...)
		return !s.HasExistingResources(), nil
	}, ctx.Done())

	return nil
}

func getName(namespace, name string) string {
	if namespace != "" {
		name = fmt.Sprintf("%s/%s", namespace, name)
	}
	return name
}

func (c Client) DoRolloutWait(ctx context.Context, key types.NamespacedName) error {
	onceReplicasUpdated := sync.Once{}
	oncePendingTermination := sync.Once{}
	onceNotAvailable := sync.Once{}
	onceSpecUpdate := sync.Once{}

	rolloutComplete := func() (bool, error) {
		deployment := appsv1.Deployment{}
		err := c.KubeClient.Get(ctx, key, &deployment)
		if err != nil {
			return false, err
		}
		if deployment.Generation <= deployment.Status.ObservedGeneration {
			cond := deploymentutil.GetDeploymentCondition(deployment.Status, appsv1.DeploymentProgressing)
			if cond != nil && cond.Reason == deploymentutil.TimedOutReason {
				return false, errors.New("progress deadline exceeded")
			}
			if deployment.Spec.Replicas != nil && deployment.Status.UpdatedReplicas < *deployment.Spec.Replicas {
				onceReplicasUpdated.Do(func() {
					log.Printf("  Waiting for deployment %q to rollout: %d out of %d new replicas have been updated", deployment.Name, deployment.Status.UpdatedReplicas, *deployment.Spec.Replicas)
				})
				return false, nil
			}
			if deployment.Status.Replicas > deployment.Status.UpdatedReplicas {
				oncePendingTermination.Do(func() {
					log.Printf("  Waiting for deployment %q to rollout: %d old replicas are pending termination", deployment.Name, deployment.Status.Replicas-deployment.Status.UpdatedReplicas)
				})
				return false, nil
			}
			if deployment.Status.AvailableReplicas < deployment.Status.UpdatedReplicas {
				onceNotAvailable.Do(func() {
					log.Printf("  Waiting for deployment %q to rollout: %d of %d updated replicas are available", deployment.Name, deployment.Status.AvailableReplicas, deployment.Status.UpdatedReplicas)
				})
				return false, nil
			}
			log.Printf("  Deployment %q successfully rolled out", deployment.Name)
			return true, nil
		}
		onceSpecUpdate.Do(func() {
			log.Printf("  Waiting for deployment %q to rollout: waiting for deployment spec update to be observed", deployment.Name)
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

	csvPhaseSucceeded := func() (bool, error) {
		csv := olmapiv1alpha1.ClusterServiceVersion{}
		err := c.KubeClient.Get(ctx, key, &csv)
		if err != nil {
			if apierrors.IsNotFound(err) {
				once.Do(func() {
					log.Printf("  Waiting for clusterserviceversion %q to appear", key.Name)
				})
				return false, nil
			}
			return false, err
		}
		newPhase = csv.Status.Phase
		if newPhase != curPhase {
			curPhase = newPhase
			log.Printf("  Found clusterserviceversion %q phase: %s", key.Name, curPhase)
		}
		return curPhase == olmapiv1alpha1.CSVPhaseSucceeded, nil
	}

	return wait.PollImmediateUntil(time.Second, csvPhaseSucceeded, ctx.Done())
}

func (c Client) GetInstalledVersion(ctx context.Context) (string, error) {
	opts := client.InNamespace(OLMNamespace)
	csvs := &olmapiv1alpha1.ClusterServiceVersionList{}
	if err := c.KubeClient.List(ctx, csvs, opts); err != nil {
		if apierrors.IsNotFound(err) || meta.IsNoMatchError(err) {
			return "", ErrOLMNotInstalled
		}
		return "", errors.Wrapf(err, "failed to list CSVs in namespace %q", OLMNamespace)
	}
	var pkgServerCSV *olmapiv1alpha1.ClusterServiceVersion
	for _, csv := range csvs.Items {
		name := csv.GetName()
		// Check old and new name possibilities.
		if name == pkgServerCSVNewName || strings.HasPrefix(name, pkgServerCSVOldNamePrefix) {
			// There is more than one version of OLM installed in the cluster,
			// so we can't resolve the version being used.
			if pkgServerCSV != nil {
				return "", errors.Errorf("more than one OLM (package server) version installed: %q and %q", pkgServerCSV.GetName(), name)
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
	return "", errors.Errorf("no OLM version found in CSV %q spec", csv.GetName())
}
