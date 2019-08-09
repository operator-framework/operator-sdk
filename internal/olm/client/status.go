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
	"fmt"
	"text/tabwriter"

	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

type Status struct {
	Resources []ResourceStatus
}

type ResourceStatus struct {
	NamespacedName types.NamespacedName
	Resource       *unstructured.Unstructured
	GVK            schema.GroupVersionKind
	Error          error
}

func (c Client) GetObjectsStatus(ctx context.Context, objs ...runtime.Object) Status {
	var rss []ResourceStatus
	for _, obj := range objs {
		a, aerr := meta.Accessor(obj)
		if aerr != nil {
			log.Fatalf("Object %s: %v", obj.GetObjectKind().GroupVersionKind(), aerr)
		}
		nn := types.NamespacedName{
			Namespace: a.GetNamespace(),
			Name:      a.GetName(),
		}
		u := unstructured.Unstructured{}
		u.SetGroupVersionKind(obj.GetObjectKind().GroupVersionKind())
		err := c.KubeClient.Get(ctx, nn, &u)
		rs := ResourceStatus{
			NamespacedName: nn,
			GVK:            obj.GetObjectKind().GroupVersionKind(),
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
