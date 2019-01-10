// Copyright 2018 The Operator-SDK Authors
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

// This file contains this project's custom code, as opposed to kubectl.go
// which contains code retrieved from the kubernetes project.

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"strings"
	"sync"

	k8sRequest "github.com/operator-framework/operator-sdk/pkg/ansible/proxy/requestfactory"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// ControllerMap - map of GVK to controller
type ControllerMap struct {
	sync.RWMutex
	internal map[schema.GroupVersionKind]controller.Controller
	watch    map[schema.GroupVersionKind]bool
}

// CacheResponseHandler will handle proxied requests and check if the requested
// resource exists in our cache. If it does then there is no need to bombard
// the APIserver with our request and we should write the response from the
// proxy.
func CacheResponseHandler(h http.Handler, informerCache cache.Cache, restMapper meta.RESTMapper) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		switch req.Method {
		case http.MethodGet:
			// GET request means we need to check the cache
			rf := k8sRequest.RequestInfoFactory{APIPrefixes: sets.NewString("api", "apis"), GrouplessAPIPrefixes: sets.NewString("api")}
			r, err := rf.NewRequestInfo(req)
			if err != nil {
				log.Error(err, "Failed to convert request")
				break
			}

			// check if resource is present on request
			if !r.IsResourceRequest {
				break
			}

			if strings.HasPrefix(r.Path, "/version") {
				// Temporarily pass along to API server
				// Ideally we cache this response as well
				break
			}

			gvr := schema.GroupVersionResource{
				Group:    r.APIGroup,
				Version:  r.APIVersion,
				Resource: r.Resource,
			}
			if restMapper == nil {
				restMapper = meta.NewDefaultRESTMapper([]schema.GroupVersion{schema.GroupVersion{
					Group:   r.APIGroup,
					Version: r.APIVersion,
				}})
			}

			k, err := restMapper.KindFor(gvr)
			if err != nil {
				// break here in case resource doesn't exist in cache
				log.Info("Cache miss, can not find in rest mapper", "GVR", gvr)
				break
			}

			un := unstructured.Unstructured{}
			un.SetGroupVersionKind(k)
			obj := client.ObjectKey{Namespace: r.Namespace, Name: r.Name}
			err = informerCache.Get(context.Background(), obj, &un)
			if err != nil {
				// break here in case resource doesn't exist in cache but exists on APIserver
				// This is very unlikely but provides user with expected 404
				log.Info(fmt.Sprintf("Cache miss: %v, %v", k, obj))
				break
			}

			i := bytes.Buffer{}
			resp, err := json.Marshal(un.Object)
			if err != nil {
				// return will give a 500
				log.Error(err, "Failed to marshal data")
				http.Error(w, "", http.StatusInternalServerError)
				return
			}

			// Set X-Cache header to signal that response is served from Cache
			w.Header().Set("X-Cache", "HIT")
			json.Indent(&i, resp, "", "  ")
			_, err = w.Write(i.Bytes())
			if err != nil {
				log.Error(err, "Failed to write response")
				http.Error(w, "", http.StatusInternalServerError)
				return
			}

			// Return so that request isn't passed along to APIserver
			return
		}
		h.ServeHTTP(w, req)
	})
}

// InjectOwnerReferenceHandler will handle proxied requests and inject the
// owner refernece found in the authorization header. The Authorization is
// then deleted so that the proxy can re-set with the correct authorization.
func InjectOwnerReferenceHandler(h http.Handler, cMap *ControllerMap, restMapper meta.RESTMapper) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodPost {
			log.Info("Injecting owner reference")
			dump, _ := httputil.DumpRequest(req, false)
			log.V(1).Info("Dumping request", "RequestDump", string(dump))

			user, _, ok := req.BasicAuth()
			if !ok {
				log.Error(errors.New("basic auth header not found"), "")
				w.Header().Set("WWW-Authenticate", "Basic realm=\"Operator Proxy\"")
				http.Error(w, "", http.StatusUnauthorized)
				return
			}
			authString, err := base64.StdEncoding.DecodeString(user)
			if err != nil {
				m := "Could not base64 decode username"
				log.Error(err, m)
				http.Error(w, m, http.StatusBadRequest)
				return
			}
			owner := metav1.OwnerReference{}
			json.Unmarshal(authString, &owner)
			log.Info(fmt.Sprintf("Owner: %#v", owner))

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
			data.SetOwnerReferences(append(data.GetOwnerReferences(), owner))
			newBody, err := json.Marshal(data.Object)
			if err != nil {
				m := "Could not serialize body"
				log.Error(err, m)
				http.Error(w, m, http.StatusInternalServerError)
				return
			}
			log.V(1).Info("Serialized body", "Body", string(newBody))
			req.Body = ioutil.NopCloser(bytes.NewBuffer(newBody))
			req.ContentLength = int64(len(newBody))
			dataMapping, err := restMapper.RESTMapping(data.GroupVersionKind().GroupKind(), data.GroupVersionKind().Version)
			if err != nil {
				m := fmt.Sprintf("Could not get rest mapping for: %v", data.GroupVersionKind())
				log.Error(err, m)
				http.Error(w, m, http.StatusInternalServerError)
				return
			}
			// We need to determine whether or not the owner is a cluster-scoped
			// resource because enqueue based on an owner reference does not work if
			// a namespaced resource owns a cluster-scoped resource
			ownerGV, err := schema.ParseGroupVersion(owner.APIVersion)
			ownerMapping, err := restMapper.RESTMapping(schema.GroupKind{Kind: owner.Kind, Group: ownerGV.Group}, ownerGV.Version)
			if err != nil {
				m := fmt.Sprintf("could not get rest mapping for: %v", owner)
				log.Error(err, m)
				http.Error(w, m, http.StatusInternalServerError)
				return
			}

			dataClusterScoped := dataMapping.Scope.Name() != meta.RESTScopeNameRoot
			ownerClusterScoped := ownerMapping.Scope.Name() != meta.RESTScopeNameRoot
			if !ownerClusterScoped || dataClusterScoped {
				// add watch for resource
				err = addWatchToController(owner, cMap, data)
				if err != nil {
					m := "could not add watch to controller"
					log.Error(err, m)
					http.Error(w, m, http.StatusInternalServerError)
					return
				}
			}
		}
		// Removing the authorization so that the proxy can set the correct authorization.
		req.Header.Del("Authorization")
		h.ServeHTTP(w, req)
	})
}

// HandlerChain will be used for users to pass defined handlers to the proxy.
// The hander chain will be run after InjectingOwnerReference if it is added
// and before the proxy handler.
type HandlerChain func(http.Handler) http.Handler

// Options will be used by the user to specify the desired details
// for the proxy.
type Options struct {
	Address          string
	Port             int
	Handler          HandlerChain
	NoOwnerInjection bool
	KubeConfig       *rest.Config
	Cache            cache.Cache
	RESTMapper       meta.RESTMapper
	ControllerMap    *ControllerMap
}

// Run will start a proxy server in a go routine that returns on the error
// channel if something is not correct on startup. Run will not return until
// the network socket is listening.
func Run(done chan error, o Options) error {
	server, err := newServer("/", o.KubeConfig)
	if err != nil {
		return err
	}
	if o.Handler != nil {
		server.Handler = o.Handler(server.Handler)
	}
	if o.ControllerMap == nil {
		return fmt.Errorf("failed to get controller map from options")
	}

	if o.Cache == nil {
		// Need to initialize cache since we don't have one
		log.Info("Initializing and starting informer cache...")
		informerCache, err := cache.New(o.KubeConfig, cache.Options{})
		if err != nil {
			return err
		}
		stop := make(chan struct{})
		go func() {
			informerCache.Start(stop)
			defer close(stop)
		}()
		log.Info("Waiting for cache to sync...")
		synced := informerCache.WaitForCacheSync(stop)
		if !synced {
			return fmt.Errorf("failed to sync cache")
		}
		log.Info("Cache sync was successful")
		o.Cache = informerCache
	}

	if !o.NoOwnerInjection {
		server.Handler = InjectOwnerReferenceHandler(server.Handler, o.ControllerMap, o.RESTMapper)
	}
	// Always add cache handler
	server.Handler = CacheResponseHandler(server.Handler, o.Cache, o.RESTMapper)

	l, err := server.Listen(o.Address, o.Port)
	if err != nil {
		return err
	}
	go func() {
		log.Info("Starting to serve", "Address", l.Addr().String())
		done <- server.ServeOnListener(l)
	}()
	return nil
}

func addWatchToController(owner metav1.OwnerReference, cMap *ControllerMap, resource *unstructured.Unstructured) error {
	gv, err := schema.ParseGroupVersion(owner.APIVersion)
	if err != nil {
		return err
	}
	gvk := schema.GroupVersionKind{
		Group:   gv.Group,
		Version: gv.Version,
		Kind:    owner.Kind,
	}
	c, watch, ok := cMap.Get(gvk)
	if !ok {
		return errors.New("failed to find controller in map")
	}
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(gvk)
	// Add a watch to controller
	if watch {
		log.Info("Watching child resource", "kind", resource.GroupVersionKind(), "enqueue_kind", u.GroupVersionKind())
		err = c.Watch(&source.Kind{Type: resource}, &handler.EnqueueRequestForOwner{OwnerType: u})
		if err != nil {
			return err
		}
	}
	return nil
}

// NewControllerMap returns a new object that contains a mapping between GVK
// and controller
func NewControllerMap() *ControllerMap {
	return &ControllerMap{
		internal: make(map[schema.GroupVersionKind]controller.Controller),
		watch:    make(map[schema.GroupVersionKind]bool),
	}
}

// Get - Returns a controller given a GVK as the key. `watch` in the return
// specifies whether or not the operator will watch dependent resources for
// this controller. `ok` returns whether the query was successful. `controller`
// is the associated controller-runtime controller object.
func (cm *ControllerMap) Get(key schema.GroupVersionKind) (controller controller.Controller, watch, ok bool) {
	cm.RLock()
	defer cm.RUnlock()
	result, ok := cm.internal[key]
	if !ok {
		return result, false, ok
	}
	watch, ok = cm.watch[key]
	return result, watch, ok
}

// Delete - Deletes associated GVK to controller mapping from the ControllerMap
func (cm *ControllerMap) Delete(key schema.GroupVersionKind) {
	cm.Lock()
	defer cm.Unlock()
	delete(cm.internal, key)
}

// Store - Adds a new GVK to controller mapping. Also creates a mapping between
// GVK and a boolean `watch` that specifies whether this controller is watching
// dependent resources.
func (cm *ControllerMap) Store(key schema.GroupVersionKind, value controller.Controller, watch bool) {
	cm.Lock()
	defer cm.Unlock()
	cm.internal[key] = value
	cm.watch[key] = watch
}
