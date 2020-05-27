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

package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/operator-framework/operator-sdk/pkg/ansible/proxy/controllermap"
	"github.com/operator-framework/operator-sdk/pkg/ansible/proxy/requestfactory"
	k8sRequest "github.com/operator-framework/operator-sdk/pkg/ansible/proxy/requestfactory"
	osdkHandler "github.com/operator-framework/operator-sdk/pkg/handler"

	"k8s.io/apimachinery/pkg/api/meta"
	metainternalscheme "k8s.io/apimachinery/pkg/apis/meta/internalversion/scheme"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type marshaler interface {
	MarshalJSON() ([]byte, error)
}

type cacheResponseHandler struct {
	next              http.Handler
	informerCache     cache.Cache
	restMapper        meta.RESTMapper
	watchedNamespaces map[string]interface{}
	cMap              *controllermap.ControllerMap
	injectOwnerRef    bool
	apiResources      *apiResources
	skipPathRegexp    []*regexp.Regexp
}

func (c *cacheResponseHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		// GET request means we need to check the cache
		rf := k8sRequest.RequestInfoFactory{APIPrefixes: sets.NewString("api", "apis"),
			GrouplessAPIPrefixes: sets.NewString("api")}
		r, err := rf.NewRequestInfo(req)
		if err != nil {
			log.Error(err, "Failed to convert request")
			break
		}

		// Skip cache for non-cacheable requests, not a part of skipCacheLookup for performance.
		if !r.IsResourceRequest || !(r.Subresource == "" || r.Subresource == "status") {
			log.Info("Skipping cache lookup", "resource", r)
			break
		}

		if c.restMapper == nil {
			c.restMapper = meta.NewDefaultRESTMapper([]schema.GroupVersion{schema.GroupVersion{
				Group:   r.APIGroup,
				Version: r.APIVersion,
			}})
		}
		k, err := getGVKFromRequestInfo(r, c.restMapper)
		if err != nil {
			// break here in case resource doesn't exist in cache
			log.Error(err, "Cache miss, can not find in rest mapper")
			break
		}

		if c.skipCacheLookup(r, k, req) {
			log.Info("Skipping cache lookup", "resource", r)
			break
		}

		// Determine if the resource is virtual. If it is then we should not attempt to use cache
		isVR, err := c.apiResources.IsVirtualResource(k)
		if err != nil {
			// break here in case we can not understand if virtual resource or not
			log.Error(err, "Unable to determine if virtual resource", "gvk", k)
			break
		}

		if isVR {
			log.V(2).Info("Virtual resource, must ask the cluster API", "gvk", k)
			break
		}

		var m marshaler

		log.V(2).Info("Get resource in our cache", "r", r)
		if r.Verb == "list" {
			m, err = c.getListFromCache(r, req, k)
			if err != nil {
				break
			}
		} else {
			m, err = c.getObjectFromCache(r, req, k)
			if err != nil {
				break
			}
		}

		i := bytes.Buffer{}
		resp, err := m.MarshalJSON()
		if err != nil {
			// return will give a 500
			log.Error(err, "Failed to marshal data")
			http.Error(w, "", http.StatusInternalServerError)
			return
		}

		// Set Content-Type header
		w.Header().Set("Content-Type", "application/json")
		// Set X-Cache header to signal that response is served from Cache
		w.Header().Set("X-Cache", "HIT")
		if err := json.Indent(&i, resp, "", "  "); err != nil {
			log.Error(err, "Failed to indent json")
		}
		_, err = w.Write(i.Bytes())
		if err != nil {
			log.Error(err, "Failed to write response")
			http.Error(w, "", http.StatusInternalServerError)
			return
		}

		// Return so that request isn't passed along to APIserver
		log.Info("Read object from cache", "resource", r)
		return
	}
	c.next.ServeHTTP(w, req)
}

// skipCacheLookup - determine if we should skip the cache lookup
func (c *cacheResponseHandler) skipCacheLookup(r *requestfactory.RequestInfo, gvk schema.GroupVersionKind,
	req *http.Request) bool {

	skip := matchesRegexp(req.URL.String(), c.skipPathRegexp)
	if skip {
		return true
	}

	owner, err := getRequestOwnerRef(req)
	if err != nil {
		log.Error(err, "Could not get owner reference from proxy.")
		return false
	}
	if owner != nil {
		ownerGV, err := schema.ParseGroupVersion(owner.APIVersion)
		if err != nil {
			m := fmt.Sprintf("Could not get group version for: %v.", owner)
			log.Error(err, m)
			return false
		}
		ownerGVK := schema.GroupVersionKind{
			Group:   ownerGV.Group,
			Version: ownerGV.Version,
			Kind:    owner.Kind,
		}

		relatedController, ok := c.cMap.Get(ownerGVK)
		if !ok {
			log.Info("Could not find controller for gvk.", "ownerGVK:", ownerGVK)
			return false
		}
		if relatedController.Blacklist[gvk] {
			log.Info("Skipping, because gvk is blacklisted", "GVK", gvk)
			return true
		}
	}
	// check if resource doesn't exist in watched namespaces
	// if watchedNamespaces[""] exists then we are watching all namespaces
	// and want to continue
	_, allNsPresent := c.watchedNamespaces[metav1.NamespaceAll]
	_, reqNsPresent := c.watchedNamespaces[r.Namespace]
	if !allNsPresent && !reqNsPresent {
		return true
	}

	if strings.HasPrefix(r.Path, "/version") {
		// Temporarily pass along to API server
		// Ideally we cache this response as well
		return true
	}

	return false
}

func (c *cacheResponseHandler) recoverDependentWatches(req *http.Request, un *unstructured.Unstructured) {
	ownerRef, err := getRequestOwnerRef(req)
	if err != nil {
		log.Error(err, "Could not get ownerRef from proxy")
		return
	}
	// This happens when a request unrelated to reconciliation hits the proxy
	if ownerRef == nil {
		return
	}

	for _, oRef := range un.GetOwnerReferences() {
		if oRef.APIVersion == ownerRef.APIVersion && oRef.Kind == ownerRef.Kind {
			err := addWatchToController(*ownerRef, c.cMap, un, c.restMapper, true)
			if err != nil {
				log.Error(err, "Could not recover dependent resource watch", "owner", ownerRef)
				return
			}
		}
	}
	if typeString, ok := un.GetAnnotations()[osdkHandler.TypeAnnotation]; ok {
		ownerGV, err := schema.ParseGroupVersion(ownerRef.APIVersion)
		if err != nil {
			m := fmt.Sprintf("could not get group version for: %v", ownerGV)
			log.Error(err, m)
			return
		}
		if typeString == fmt.Sprintf("%v.%v", ownerRef.Kind, ownerGV.Group) {
			err := addWatchToController(*ownerRef, c.cMap, un, c.restMapper, false)
			if err != nil {
				log.Error(err, "Could not recover dependent resource watch", "owner", ownerRef)
				return
			}
		}
	}
}

func (c *cacheResponseHandler) getListFromCache(r *requestfactory.RequestInfo, req *http.Request,
	k schema.GroupVersionKind) (marshaler, error) {
	k8sListOpts := &metav1.ListOptions{}
	if err := metainternalscheme.ParameterCodec.DecodeParameters(req.URL.Query(), metav1.SchemeGroupVersion,
		k8sListOpts); err != nil {
		log.Error(err, "Unable to decode list options from request")
		return nil, err
	}
	clientListOpts := []client.ListOption{
		client.InNamespace(r.Namespace),
	}
	if k8sListOpts.LabelSelector != "" {
		sel, err := labels.ConvertSelectorToLabelsMap(k8sListOpts.LabelSelector)
		if err != nil {
			log.Error(err, "Unable to convert label selectors for the client")
			return nil, err
		}
		clientListOpts = append(clientListOpts, client.MatchingLabels(sel))
	}
	if k8sListOpts.FieldSelector != "" {
		sel, err := fields.ParseSelector(k8sListOpts.FieldSelector)
		if err != nil {
			log.Error(err, "Unable to parse field selectors for the client")
			return nil, err
		}
		clientListOpts = append(clientListOpts, client.MatchingFieldsSelector{Selector: sel})
	}
	k.Kind = k.Kind + "List"
	un := unstructured.UnstructuredList{}
	un.SetGroupVersionKind(k)
	ctx, cancel := context.WithTimeout(context.Background(), cacheEstablishmentTimeout)
	defer cancel()
	err := c.informerCache.List(ctx, &un, clientListOpts...)
	if err != nil {
		// break here in case resource doesn't exist in cache but exists on APIserver
		// This is very unlikely but provides user with expected 404
		log.Info(fmt.Sprintf("cache miss: %v err-%v", k, err))
		return nil, err
	}
	return &un, nil
}

func (c *cacheResponseHandler) getObjectFromCache(r *requestfactory.RequestInfo, req *http.Request,
	k schema.GroupVersionKind) (marshaler, error) {
	un := &unstructured.Unstructured{}
	un.SetGroupVersionKind(k)
	obj := client.ObjectKey{Namespace: r.Namespace, Name: r.Name}
	ctx, cancel := context.WithTimeout(context.Background(), cacheEstablishmentTimeout)
	defer cancel()
	err := c.informerCache.Get(ctx, obj, un)
	if err != nil {
		// break here in case resource doesn't exist in cache but exists on APIserver
		// This is very unlikely but provides user with expected 404
		log.Info(fmt.Sprintf("Cache miss: %v, %v", k, obj))
		return nil, err
	}
	// Once we get the resource, we are going to attempt to recover the dependent watches here,
	// This will happen in the background, and log errors.
	if c.injectOwnerRef {
		go c.recoverDependentWatches(req, un)
	}
	return un, nil
}
