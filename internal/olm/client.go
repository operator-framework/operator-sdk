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
	"io"
	"io/ioutil"
	"net/http"
	"time"

	olmresourceclient "github.com/operator-framework/operator-sdk/internal/olm/client"

	olmapiv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/rest"
)

const (
	olmNamespace = "olm"
)

var (
	olmOperatorKey     = types.NamespacedName{Namespace: olmNamespace, Name: "olm-operator"}
	catalogOperatorKey = types.NamespacedName{Namespace: olmNamespace, Name: "catalog-operator"}
	packageServerKey   = types.NamespacedName{Namespace: olmNamespace, Name: "packageserver"}
)

var ErrOLMNotInstalled = errors.New("no existing installation found")

type Client struct {
	*olmresourceclient.Client
	HTTPClient      http.Client
	BaseDownloadURL string
}

func ClientForConfig(cfg *rest.Config) (*Client, error) {
	cl, err := olmresourceclient.ClientForConfig(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get OLM resource client")
	}
	c := &Client{
		Client:          cl,
		HTTPClient:      *http.DefaultClient,
		BaseDownloadURL: "https://github.com/operator-framework/operator-lifecycle-manager/releases",
	}
	return c, nil
}

func (c Client) InstallVersion(ctx context.Context, version string) (*olmresourceclient.Status, error) {
	resources, err := c.getResources(ctx, version)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get resources")
	}
	objs := toObjects(resources...)

	status := c.GetObjectsStatus(ctx, objs...)
	if status.HasExistingResources() {
		return nil, errors.New("detected existing OLM resources: OLM must be completely uninstalled before installation")
	}

	log.Print("Creating CRDs and resources")
	if err := c.DoCreate(ctx, objs...); err != nil {
		return nil, errors.Wrap(err, "failed to create CRDs and resources")
	}

	log.Print("Waiting for deployment/olm-operator rollout to complete")
	if err := c.DoRolloutWait(ctx, olmOperatorKey); err != nil {
		return nil, errors.Wrapf(err, "deployment/%s failed to rollout", olmOperatorKey.Name)
	}

	log.Print("Waiting for deployment/catalog-operator rollout to complete")
	if err := c.DoRolloutWait(ctx, catalogOperatorKey); err != nil {
		return nil, errors.Wrapf(err, "deployment/%s failed to rollout", catalogOperatorKey.Name)
	}

	subscriptions := filterResources(resources, func(r unstructured.Unstructured) bool {
		return r.GroupVersionKind() == schema.GroupVersionKind{
			Group:   olmapiv1alpha1.GroupName,
			Version: olmapiv1alpha1.GroupVersion,
			Kind:    olmapiv1alpha1.SubscriptionKind,
		}
	})

	for _, sub := range subscriptions {
		subscriptionKey := types.NamespacedName{Namespace: sub.GetNamespace(), Name: sub.GetName()}
		log.Printf("Waiting for subscription/%s to install CSV", subscriptionKey.Name)
		csvKey, err := c.getSubscriptionCSV(ctx, subscriptionKey)
		if err != nil {
			return nil, errors.Wrapf(err, "subscription/%s failed to install CSV", subscriptionKey.Name)
		}
		log.Printf("Waiting for clusterserviceversion/%s to reach 'Succeeded' phase", csvKey.Name)
		if err := c.DoCSVWait(ctx, csvKey); err != nil {
			return nil, errors.Wrapf(err, "clusterserviceversion/%s failed to reach 'Succeeded' phase", csvKey.Name)
		}
	}

	log.Printf("Waiting for deployment/%s rollout to complete", packageServerKey.Name)
	if err := c.DoRolloutWait(ctx, packageServerKey); err != nil {
		return nil, errors.Wrapf(err, "deployment/%s failed to rollout", packageServerKey.Name)
	}

	status = c.GetObjectsStatus(ctx, objs...)
	return &status, nil
}

func (c Client) UninstallVersion(ctx context.Context, version string) error {
	resources, err := c.getResources(ctx, version)
	if err != nil {
		return errors.Wrap(err, "failed to get resources")
	}
	objs := toObjects(resources...)

	status := c.GetObjectsStatus(ctx, objs...)
	if !status.HasExistingResources() {
		return ErrOLMNotInstalled
	}

	log.Infof("Uninstalling resources for version %q", version)
	if err := c.DoDelete(ctx, objs...); err != nil {
		return err
	}
	return nil
}

func (c Client) GetStatus(ctx context.Context, version string) (*olmresourceclient.Status, error) {
	resources, err := c.getResources(ctx, version)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get resources")
	}
	objs := toObjects(resources...)

	status := c.GetObjectsStatus(ctx, objs...)
	if !status.HasExistingResources() {
		return nil, ErrOLMNotInstalled
	}
	return &status, nil
}

func (c Client) getResources(ctx context.Context, version string) ([]unstructured.Unstructured, error) {
	log.Infof("Fetching CRDs for version %q", version)
	crdResources, err := c.getCRDs(ctx, version)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch CRDs")
	}

	log.Infof("Fetching resources for version %q", version)
	olmResources, err := c.getOLM(ctx, version)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch resources")
	}

	resources := append(crdResources, olmResources...)
	return resources, nil
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

func (c Client) crdsURL(version string) string {
	return fmt.Sprintf("%s/crds.yaml", c.getBaseDownloadURL(version))
}

func (c Client) olmURL(version string) string {
	return fmt.Sprintf("%s/olm.yaml", c.getBaseDownloadURL(version))
}

func (c Client) getBaseDownloadURL(version string) string {
	if version == "latest" {
		return fmt.Sprintf("%s/%s/download", c.BaseDownloadURL, version)
	}
	return fmt.Sprintf("%s/download/%s", c.BaseDownloadURL, version)
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

func toObjects(us ...unstructured.Unstructured) (objs []runtime.Object) {
	for i := range us {
		objs = append(objs, &us[i])
	}
	return objs
}

func decodeResources(rds ...io.Reader) (objs []unstructured.Unstructured, err error) {
	for _, r := range rds {
		dec := yaml.NewYAMLOrJSONDecoder(r, 8)
		for {
			var u unstructured.Unstructured
			err = dec.Decode(&u)
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

func filterResources(resources []unstructured.Unstructured, filter func(unstructured.Unstructured) bool) (filtered []unstructured.Unstructured) {
	for _, r := range resources {
		if filter(r) {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

func (c Client) getSubscriptionCSV(ctx context.Context, subKey types.NamespacedName) (types.NamespacedName, error) {
	var csvKey types.NamespacedName
	subscriptionInstalledCSV := func() (bool, error) {
		sub := olmapiv1alpha1.Subscription{}
		err := c.KubeClient.Get(ctx, subKey, &sub)
		if err != nil {
			return false, err
		}
		installedCSV := sub.Status.InstalledCSV
		if installedCSV == "" {
			return false, nil
		}
		csvKey = types.NamespacedName{
			Namespace: subKey.Namespace,
			Name:      installedCSV,
		}
		log.Printf("  Found installed CSV %q", installedCSV)
		return true, nil
	}

	return csvKey, wait.PollImmediateUntil(time.Second, subscriptionInstalledCSV, ctx.Done())
}
