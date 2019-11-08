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
	"text/tabwriter"

	log "github.com/sirupsen/logrus"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/install"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes/scheme"
)

var sch = scheme.Scheme

func init() {
	install.Install(sch)
}

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
		return false, fmt.Errorf("error getting set of CRD kinds in resources: %w", err)
	}
	for _, r := range s.Resources {
		if r.Resource != nil {
			return true, nil
		} else if r.Error != nil && !apierrors.IsNotFound(r.Error) {
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
	dec := serializer.NewCodecFactory(sch).UniversalDeserializer()
	for _, r := range s.Resources {
		if r.GVK.Kind == "CustomResourceDefinition" {
			switch v := r.requestObject.(type) {
			case *unstructured.Unstructured:
				vb, err := v.MarshalJSON()
				if err != nil {
					return nil, err
				}
				// Use unversioned CustomResourceDefinition to avoid implementing cast
				// for all versions.
				obj, _, err := dec.Decode(vb, nil, nil)
				if err != nil {
					return nil, err
				}
				kind, err := getVersionedCRDKind(obj)
				if err != nil {
					return nil, err
				}
				crdKindSet.Insert(kind)
			default:
				kind, err := getVersionedCRDKind(v)
				if err != nil {
					return nil, err
				}
				crdKindSet.Insert(kind)
			}
		}
	}
	return crdKindSet, nil
}

// getVersionedCRDKind returns the kind of a CRD if its version is known,
// otherwise an error.
func getVersionedCRDKind(obj runtime.Object) (string, error) {
	switch crd := obj.(type) {
	case *apiextensions.CustomResourceDefinition:
		return crd.Spec.Names.Kind, nil
	case *v1beta1.CustomResourceDefinition:
		return crd.Spec.Names.Kind, nil
	}
	return "", fmt.Errorf("error getting CRD kind: gvk %q unknown", obj.GetObjectKind().GroupVersionKind())
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
