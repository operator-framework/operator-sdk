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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"sync"
	"text/tabwriter"
	"time"

	"github.com/operator-framework/operator-sdk/pkg/restmapper"

	olmapiv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	deploymentutil "k8s.io/kubernetes/pkg/controller/deployment/util"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	olmNamespace = "olm"
)

var (
	olmOperatorKey     = types.NamespacedName{Namespace: olmNamespace, Name: "olm-operator"}
	catalogOperatorKey = types.NamespacedName{Namespace: olmNamespace, Name: "catalog-operator"}
	packageServerKey   = types.NamespacedName{Namespace: olmNamespace, Name: "packageserver"}
)

type Client struct {
	KubeClient client.Client
	HTTPClient http.Client

	BaseDownloadURL    string
	BaseReleasesAPIURL string
}

func ClientForConfig(cfg *rest.Config) (*Client, error) {
	rm, err := restmapper.NewDynamicRESTMapper(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create dynamic rest mapper")
	}

	sch := scheme.Scheme
	if err := olmapiv1alpha1.AddToScheme(sch); err != nil {
		return nil, errors.Wrap(err, "failed to add OLM types to scheme")
	}

	cl, err := client.New(cfg, client.Options{
		Scheme: sch,
		Mapper: rm,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create client")
	}

	c := &Client{
		KubeClient:         cl,
		HTTPClient:         *http.DefaultClient,
		BaseDownloadURL:    "https://github.com/operator-framework/operator-lifecycle-manager/releases",
		BaseReleasesAPIURL: "https://api.github.com/repos/operator-framework/operator-lifecycle-manager/releases",
	}
	return c, nil
}

func (c Client) InstallVersion(ctx context.Context, version string) (*Status, error) {
	resources, err := c.getResources(ctx, version)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get resources")
	}

	status := c.getStatus(ctx, resources)
	if status.HasExistingResources() {
		return nil, errors.New("detected existing OLM resources: OLM must be completely uninstalled before installation")
	}

	log.Print("Creating CRDs and resources")
	if err := c.doCreate(ctx, resources); err != nil {
		return nil, errors.Wrap(err, "failed to create CRDs and resources")
	}

	log.Print("Waiting for deployment/olm-operator rollout to complete")
	if err := c.doRolloutWait(ctx, olmOperatorKey); err != nil {
		return nil, errors.Wrapf(err, "deployment %q failed to rollout", olmOperatorKey.Name)
	}

	log.Print("Waiting for deployment/catalog-operator rollout to complete")
	if err := c.doRolloutWait(ctx, catalogOperatorKey); err != nil {
		return nil, errors.Wrapf(err, "deployment %q failed to rollout", catalogOperatorKey.Name)
	}

	packageServerCSV := types.NamespacedName{Namespace: olmNamespace, Name: fmt.Sprintf("packageserver.v%s", version)}
	log.Printf("Waiting for clusterserviceversion %q to reach 'Succeeded' phase", packageServerCSV.Name)
	if err := c.doCSVWait(ctx, packageServerCSV); err != nil {
		return nil, errors.Wrapf(err, "clusterservice version %q failed to reach phase succeeded", packageServerCSV.Name)
	}

	log.Printf("Waiting for deployment %q rollout to complete", packageServerKey.Name)
	if err := c.doRolloutWait(ctx, packageServerKey); err != nil {
		return nil, errors.Wrapf(err, "deployment %q failed to rollout", packageServerKey.Name)
	}

	status = c.getStatus(ctx, resources)
	return &status, nil
}

func (c Client) UninstallVersion(ctx context.Context, version string) error {
	resources, err := c.getResources(ctx, version)
	if err != nil {
		return errors.Wrap(err, "failed to get resources")
	}

	status := c.getStatus(ctx, resources)
	if !status.HasExistingResources() {
		return fmt.Errorf("failed to delete OLM version %s: no existing installation found", version)
	}

	log.Infof("Uninstalling resources for version %s", version)
	if err := c.doDelete(ctx, resources); err != nil {
		return errors.Wrapf(err, "failed to delete OLM version %s", version)
	}
	return nil
}

func (c Client) GetVersion(ctx context.Context, version string) (string, error) {
	log.Infof("Resolving version %q", version)
	version, err := c.resolveVersion(ctx, version)
	if err != nil {
		return "", errors.Wrapf(err, "failed to resolve version %s", version)
	}
	log.Infof("  Found GitHub release version %s", version)
	return version, nil
}

func (c Client) GetStatus(ctx context.Context, version string) (*Status, error) {
	resources, err := c.getResources(ctx, version)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get resources")
	}

	status := c.getStatus(ctx, resources)
	if !status.HasExistingResources() {
		return nil, errors.New("no existing installation found")
	}
	return &status, nil
}

func (c Client) getResources(ctx context.Context, version string) ([]unstructured.Unstructured, error) {
	log.Infof("Fetching CRDs for version %s", version)
	crdResources, err := c.getCRDs(ctx, version)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch CRDs")
	}

	log.Infof("Fetching resources for version %s", version)
	olmResources, err := c.getOLM(ctx, version)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch resources")
	}

	resources := append(crdResources, olmResources...)
	return resources, nil
}

func (c Client) getStatus(ctx context.Context, objs []unstructured.Unstructured) Status {
	var rss []ResourceStatus
	for _, obj := range objs {
		nn := types.NamespacedName{
			Namespace: obj.GetNamespace(),
			Name:      obj.GetName(),
		}
		u := unstructured.Unstructured{}
		u.SetGroupVersionKind(obj.GroupVersionKind())
		err := c.KubeClient.Get(ctx, nn, &u)
		rs := ResourceStatus{
			NamespacedName: nn,
			GVK:            obj.GroupVersionKind(),
		}
		if err != nil {
			rs.Error = err
		} else {
			rs.Resource = &u
		}
		rss = append(rss, rs)
	}

	return Status{Resources: rss}
}

func (c Client) resolveVersion(ctx context.Context, version string) (string, error) {
	var url string
	if version == "latest" {
		url = fmt.Sprintf("%s/%s", c.BaseReleasesAPIURL, version)
	} else {
		url = fmt.Sprintf("%s/tags/%s", c.BaseReleasesAPIURL, version)
	}

	resp, err := c.doRequest(ctx, url)
	if err != nil {
		return version, errors.Wrap(err, "request failed")
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return version, errors.Wrap(err, "read response body")
	}

	var d struct {
		TagName string `json:"tag_name"`
	}
	if err := json.Unmarshal([]byte(data), &d); err != nil {
		return version, errors.Wrap(err, "unmarshal json")
	}
	return d.TagName, nil
}

func (c Client) getCRDs(ctx context.Context, version string) ([]unstructured.Unstructured, error) {
	resp, err := c.doRequest(ctx, c.crdsURL(version))
	if err != nil {
		return nil, errors.Wrap(err, "request failed")
	}
	defer resp.Body.Close()
	return decodeResources(resp.Body)
}

func (c Client) getOLM(ctx context.Context, version string) ([]unstructured.Unstructured, error) {
	resp, err := c.doRequest(ctx, c.olmURL(version))
	if err != nil {
		return nil, errors.Wrap(err, "request failed")
	}
	defer resp.Body.Close()
	return decodeResources(resp.Body)
}

func (c Client) doRequest(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "create request")
	}
	req = req.WithContext(ctx)
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, errors.Wrapf(err, "failed GET '%s'", url)
	}
	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		msg := fmt.Sprintf("failed GET '%s': unexpected status code %d, expected %d", url, resp.StatusCode, http.StatusOK)
		if err != nil {
			return nil, errors.Wrap(err, msg)
		}
		return nil, fmt.Errorf("%s: %s", msg, string(body))
	}
	return resp, nil
}

func (c Client) getBaseDownloadURL(version string) string {
	if version == "latest" {
		return fmt.Sprintf("%s/%s/download", c.BaseDownloadURL, version)
	}
	return fmt.Sprintf("%s/download/%s", c.BaseDownloadURL, version)
}

func (c Client) crdsURL(version string) string {
	return fmt.Sprintf("%s/crds.yaml", c.getBaseDownloadURL(version))
}

func (c Client) olmURL(version string) string {
	return fmt.Sprintf("%s/olm.yaml", c.getBaseDownloadURL(version))
}

func decodeResources(rds ...io.Reader) ([]unstructured.Unstructured, error) {
	var objs []unstructured.Unstructured
	for _, r := range rds {
		dec := yaml.NewYAMLOrJSONDecoder(r, 8)
		for {
			var u unstructured.Unstructured
			err := dec.Decode(&u)
			if err == io.EOF {
				break
			} else if err != nil {
				return nil, err
			}
			objs = append(objs, u)
		}
	}
	return objs, nil
}

func (c Client) doCreate(ctx context.Context, objs []unstructured.Unstructured) error {
	for _, obj := range objs {
		log.Infof("  Creating %s %q", obj.GroupVersionKind().Kind, getName(obj.GetNamespace(), obj.GetName()))
		err := c.KubeClient.Create(ctx, &obj)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c Client) doDelete(ctx context.Context, objs []unstructured.Unstructured) error {
	for _, obj := range objs {
		log.Infof("  Removing %s %q", obj.GroupVersionKind().Kind, getName(obj.GetNamespace(), obj.GetName()))
		err := c.KubeClient.Delete(ctx, &obj, client.PropagationPolicy(metav1.DeletePropagationForeground))
		if err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			return err
		}
	}

	log.Infof("  Waiting for deleted resources to disappear")
	wait.PollImmediateUntil(time.Second, func() (bool, error) {
		s := c.getStatus(ctx, objs)
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

func (c Client) doRolloutWait(ctx context.Context, key types.NamespacedName) error {
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

func (c Client) doCSVWait(ctx context.Context, key types.NamespacedName) error {
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

type Status struct {
	Resources []ResourceStatus
}

type ResourceStatus struct {
	NamespacedName types.NamespacedName
	Resource       *unstructured.Unstructured
	GVK            schema.GroupVersionKind
	Error          error
}

func (s Status) HasExistingResources() bool {
	for _, r := range s.Resources {
		if r.Resource != nil {
			return true
		}
	}
	return false
}

func (s Status) String() string {
	out := &bytes.Buffer{}
	tw := tabwriter.NewWriter(out, 8, 4, 4, ' ', 0)
	fmt.Fprintf(tw, "NAME\tNAMESPACE\tKIND\tSTATUS\n")
	for _, r := range s.Resources {
		nn := r.NamespacedName
		kind := r.GVK.Kind
		var status string
		if r.Resource != nil {
			status = "Installed"
		} else if r.Error != nil {
			status = r.Error.Error()
		} else {
			status = "Unknown"
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", nn.Name, nn.Namespace, kind, status)
	}
	tw.Flush()

	return out.String()
}
