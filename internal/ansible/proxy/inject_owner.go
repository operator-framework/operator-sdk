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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"

	"github.com/operator-framework/operator-lib/handler"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/operator-framework/operator-sdk/internal/ansible/proxy/controllermap"
	k8sRequest "github.com/operator-framework/operator-sdk/internal/ansible/proxy/requestfactory"
	"github.com/operator-framework/operator-sdk/internal/util/k8sutil"
)

// injectOwnerReferenceHandler will handle proxied requests and inject the
// owner reference found in the authorization header. The Authorization is
// then deleted so that the proxy can re-set with the correct authorization.
type injectOwnerReferenceHandler struct {
	next              http.Handler
	cMap              *controllermap.ControllerMap
	restMapper        meta.RESTMapper
	watchedNamespaces map[string]interface{}
	apiResources      *apiResources
}

func (i *injectOwnerReferenceHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodPost:
		dump, _ := httputil.DumpRequest(req, false)
		log.V(2).Info("Dumping request", "RequestDump", string(dump))
		rf := k8sRequest.RequestInfoFactory{APIPrefixes: sets.NewString("api", "apis"),
			GrouplessAPIPrefixes: sets.NewString("api")}
		r, err := rf.NewRequestInfo(req)
		if err != nil {
			m := "Could not convert request"
			log.Error(err, m)
			http.Error(w, m, http.StatusBadRequest)
			return
		}
		if r.Subresource != "" {
			// Don't inject owner ref if we are POSTing to a subresource
			break
		}

		if i.restMapper == nil {
			i.restMapper = meta.NewDefaultRESTMapper([]schema.GroupVersion{schema.GroupVersion{
				Group:   r.APIGroup,
				Version: r.APIVersion,
			}})
		}

		k, err := getGVKFromRequestInfo(r, i.restMapper)
		if err != nil {
			// break here in case resource doesn't exist in cache
			log.Error(err, "Cache miss, can not find in rest mapper")
			break
		}

		// Determine if the resource is virtual. If it is then we should not attempt to use cache
		isVR, err := i.apiResources.IsVirtualResource(k)
		if err != nil {
			// Fail if we can't determine whether it's a virtual resource or not.
			// Otherwise we might create a resource without an ownerReference, which will prevent
			// dependentWatches from being re-established and garbage collection from deleting the
			// resource, unless a user manually adds the ownerReference.
			m := "Unable to determine if virtual resource"
			log.Error(err, m, "gvk", k)
			http.Error(w, m, http.StatusInternalServerError)
			return
		}

		if isVR {
			log.V(2).Info("Virtual resource, must ask the cluster API", "gvk", k)
			break
		}

		log.Info("Injecting owner reference")
		owner, err := getRequestOwnerRef(req)
		if err != nil {
			m := "Could not get owner reference"
			log.Error(err, m)
			http.Error(w, m, http.StatusInternalServerError)
			return
		}
		if owner != nil {
			body, err := ioutil.ReadAll(req.Body)
			if err != nil {
				m := "Could not read request body"
				log.Error(err, m)
				http.Error(w, m, http.StatusInternalServerError)
				return
			}
			data := &unstructured.Unstructured{}
			err = json.Unmarshal(body, data)
			if err != nil {
				m := "Could not deserialize request body"
				log.Error(err, m)
				http.Error(w, m, http.StatusBadRequest)
				return
			}
			ownerGV, err := schema.ParseGroupVersion(owner.APIVersion)
			if err != nil {
				m := fmt.Sprintf("could not get group version for: %v", owner)
				log.Error(err, m)
				http.Error(w, m, http.StatusBadRequest)
				return
			}
			ownerGVK := schema.GroupVersionKind{
				Group:   ownerGV.Group,
				Version: ownerGV.Version,
				Kind:    owner.Kind,
			}
			ownerObject := &unstructured.Unstructured{}
			ownerObject.SetGroupVersionKind(ownerGVK)
			ownerObject.SetNamespace(owner.Namespace)
			ownerObject.SetName(owner.Name)
			addOwnerRef, err := k8sutil.SupportsOwnerReference(i.restMapper, ownerObject, data, r.Namespace)
			if err != nil {
				m := "Could not determine if we should add owner ref"
				log.Error(err, m)
				http.Error(w, m, http.StatusBadRequest)
				return
			}
			if addOwnerRef {
				data.SetOwnerReferences(append(data.GetOwnerReferences(), owner.OwnerReference))
			} else {
				err := handler.SetOwnerAnnotations(ownerObject, data)
				if err != nil {
					m := "Could not set owner annotations"
					log.Error(err, m)
					http.Error(w, m, http.StatusBadRequest)
					return
				}
			}
			newBody, err := json.Marshal(data.Object)
			if err != nil {
				m := "Could not serialize body"
				log.Error(err, m)
				http.Error(w, m, http.StatusInternalServerError)
				return
			}
			log.V(2).Info("Serialized body", "Body", string(newBody))
			req.Body = ioutil.NopCloser(bytes.NewBuffer(newBody))
			req.ContentLength = int64(len(newBody))

			// add watch for resource
			// check if resource doesn't exist in watched namespaces
			// if watchedNamespaces[""] exists then we are watching all namespaces
			// and want to continue
			// This is making sure we are not attempting to watch a resource outside of the
			// namespaces that the cache can watch.
			_, allNsPresent := i.watchedNamespaces[metav1.NamespaceAll]
			_, reqNsPresent := i.watchedNamespaces[r.Namespace]
			if allNsPresent || reqNsPresent {
				err = addWatchToController(*owner, i.cMap, data, i.restMapper, addOwnerRef)
				if err != nil {
					m := "could not add watch to controller"
					log.Error(err, m)
					http.Error(w, m, http.StatusInternalServerError)
					return
				}
			}
		}
	}
	i.next.ServeHTTP(w, req)
}
