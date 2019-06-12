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
	"strings"

	"github.com/operator-framework/operator-sdk/pkg/ansible/proxy/controllermap"
	"github.com/operator-framework/operator-sdk/pkg/ansible/proxy/kubeconfig"
	k8sRequest "github.com/operator-framework/operator-sdk/pkg/ansible/proxy/requestfactory"
	osdkHandler "github.com/operator-framework/operator-sdk/pkg/handler"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
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
		rf := k8sRequest.RequestInfoFactory{APIPrefixes: sets.NewString("api", "apis"), GrouplessAPIPrefixes: sets.NewString("api")}
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
			log.Info("Cache miss, can not find in rest mapper")
			break
		}

		// Determine if the resource is virtual. If it is then we should not attempt to use cache
		isVR, err := i.apiResources.IsVirtualResource(k)
		if err != nil {
			// break here in case we can not understand if virtual resource or not
			log.Info("Unable to determine if virual resource", "gvk", k)
			break
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

		addOwnerRef, err := shouldAddOwnerRef(data, owner, i.restMapper)
		if err != nil {
			m := "Could not determine if we should add owner ref"
			log.Error(err, m)
			http.Error(w, m, http.StatusBadRequest)
			return
		}
		if addOwnerRef {
			data.SetOwnerReferences(append(data.GetOwnerReferences(), owner.OwnerReference))
		} else {
			ownerGV, err := schema.ParseGroupVersion(owner.APIVersion)
			if err != nil {
				m := fmt.Sprintf("could not get broup version for: %v", owner)
				log.Error(err, m)
				http.Error(w, m, http.StatusBadRequest)
				return
			}
			a := data.GetAnnotations()
			if a == nil {
				a = map[string]string{}
			}
			a[osdkHandler.NamespacedNameAnnotation] = strings.Join([]string{owner.Namespace, owner.Name}, "/")
			a[osdkHandler.TypeAnnotation] = fmt.Sprintf("%v.%v", owner.Kind, ownerGV.Group)

			data.SetAnnotations(a)
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
			err = addWatchToController(owner, i.cMap, data, i.restMapper, addOwnerRef)
			if err != nil {
				m := "could not add watch to controller"
				log.Error(err, m)
				http.Error(w, m, http.StatusInternalServerError)
				return
			}
		}
	}
	i.next.ServeHTTP(w, req)
}

func shouldAddOwnerRef(data *unstructured.Unstructured, owner kubeconfig.NamespacedOwnerReference, restMapper meta.RESTMapper) (bool, error) {
	dataMapping, err := restMapper.RESTMapping(data.GroupVersionKind().GroupKind(), data.GroupVersionKind().Version)
	if err != nil {
		m := fmt.Sprintf("Could not get rest mapping for: %v", data.GroupVersionKind())
		log.Error(err, m)
		return false, err

	}
	// We need to determine whether or not the owner is a cluster-scoped
	// resource because enqueue based on an owner reference does not work if
	// a namespaced resource owns a cluster-scoped resource
	ownerGV, err := schema.ParseGroupVersion(owner.APIVersion)
	if err != nil {
		m := fmt.Sprintf("could not get group version for: %v", owner)
		log.Error(err, m)
		return false, err
	}
	ownerMapping, err := restMapper.RESTMapping(schema.GroupKind{Kind: owner.Kind, Group: ownerGV.Group}, ownerGV.Version)
	if err != nil {
		m := fmt.Sprintf("could not get rest mapping for: %v", owner)
		log.Error(err, m)
		return false, err
	}

	dataNamespaceScoped := dataMapping.Scope.Name() != meta.RESTScopeNameRoot
	ownerNamespaceScoped := ownerMapping.Scope.Name() != meta.RESTScopeNameRoot

	if dataNamespaceScoped && ownerNamespaceScoped && data.GetNamespace() == owner.Namespace {
		return true, nil
	}
	return false, nil
}
