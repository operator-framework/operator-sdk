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
	"errors"
	"fmt"
	"sort"
	"text/tabwriter"

	log "github.com/sirupsen/logrus"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
)

type Status struct {
	Resources []ResourceStatus
}

type ResourceStatus struct {
	NamespacedName types.NamespacedName
	Resource       *unstructured.Unstructured
	GVK            schema.GroupVersionKind
	Error          error

	requestObject runtime.Object // Needed for context on errors from requests on an object.
}

func (c Client) GetObjectsStatus(ctx context.Context, objs ...runtime.Object) Status {
	var rss []ResourceStatus
	for _, obj := range objs {
		gvk := obj.GetObjectKind().GroupVersionKind()
		a, aerr := meta.Accessor(obj)
		if aerr != nil {
			log.Fatalf("Object %s: %v", gvk, aerr)
		}
		nn := types.NamespacedName{
			Namespace: a.GetNamespace(),
			Name:      a.GetName(),
		}
		rs := ResourceStatus{
			NamespacedName: nn,
			GVK:            gvk,
			requestObject:  obj,
		}
		u := unstructured.Unstructured{}
		u.SetGroupVersionKind(gvk)
		rs.Error = c.KubeClient.Get(ctx, nn, &u)
		if rs.Error == nil {
			rs.Resource = &u
		}
		rss = append(rss, rs)
	}

	return Status{Resources: rss}
}

// HasInstalledResources only returns true if at least one resource in s
// was returned successfully by the API server. A resource error status
// containing any error except "not found", or "no kind match" errors
// for Custom Resources, will result in HasInstalledResources returning
// false and the error.
func (s Status) HasInstalledResources() (bool, error) {
	crdKindSet, err := s.getCRDKindSet()
	if err != nil {
		return false, fmt.Errorf("error getting set of CRD kinds in resources: %v", err)
	}
	// Sort resources by whether they're installed or not to get consistent
	// return values.
	sort.Slice(s.Resources, func(i int, j int) bool {
		return s.Resources[i].Resource != nil
	})
	for _, r := range s.Resources {
		if r.Resource != nil {
			return true, nil
		} else if r.Error != nil && !apierrors.IsNotFound(r.Error) {
			// We know the error is not a "resource not found" error at this point.
			// It still may be the equivalent for a CR, "no kind match", if its
			// corresponding CRD has been deleted already. We want to make sure
			// we're only allowing "no kind match" errors to be skipped for CR's
			// since we do not know if a kind is a CR kind, hence checking
			// crdKindSet for existence of a resource's kind.
			nkmerr := &meta.NoKindMatchError{}
			if !errors.As(r.Error, &nkmerr) || !crdKindSet.Has(r.GVK.Kind) {
				return false, r.Error
			}
		}
	}
	return false, nil
}

// getCRDKindSet returns the set of all kinds specified by all CRDs in s.
func (s Status) getCRDKindSet() (sets.String, error) {
	crdKindSet := sets.NewString()
	for _, r := range s.Resources {
		if r.GVK.Kind == "CustomResourceDefinition" {
			u := &unstructured.Unstructured{}
			switch v := r.requestObject.(type) {
			case *unstructured.Unstructured:
				u = v
			default:
				uObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&v)
				if err != nil {
					return nil, err
				}
				// Other fields are not important, just the CRD kind.
				u.Object = uObj
			}
			// Use unversioned CustomResourceDefinition to avoid implementing cast
			// for all versions.
			crd := &apiextensions.CustomResourceDefinition{}
			err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &crd)
			if err != nil {
				return nil, err
			}
			crdKindSet.Insert(crd.Spec.Names.Kind)
		}
	}
	return crdKindSet, nil
}

func (s Status) String() string {
	out := &bytes.Buffer{}
	tw := tabwriter.NewWriter(out, 8, 4, 4, ' ', 0)
	fmt.Fprintf(tw, "NAME\tNAMESPACE\tKIND\tSTATUS\n")
	for _, r := range s.Resources {
		nn := r.NamespacedName
		kind := r.GVK.Kind
		var status string
		if r.Error != nil {
			status = r.Error.Error()
		} else if r.Resource != nil {
			status = "Installed"
		} else {
			status = "Unknown"
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", nn.Name, nn.Namespace, kind, status)
	}
	tw.Flush()

	return out.String()
}
