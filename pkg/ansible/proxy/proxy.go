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
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"

	"github.com/operator-framework/operator-sdk/pkg/ansible/proxy/controllermap"
	"github.com/operator-framework/operator-sdk/pkg/ansible/proxy/kubeconfig"
	k8sRequest "github.com/operator-framework/operator-sdk/pkg/ansible/proxy/requestfactory"
	osdkHandler "github.com/operator-framework/operator-sdk/pkg/handler"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// RequestLogHandler - log the requests that come through the proxy.
func RequestLogHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// read body
		body, err := ioutil.ReadAll(req.Body)
		if err != nil {
			log.Error(err, "Could not read request body")
		}
		// fix body
		req.Body = ioutil.NopCloser(bytes.NewBuffer(body))
		log.Info("Request Info", "method", req.Method, "uri", req.RequestURI, "body", string(body))
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
	Address           string
	Port              int
	Handler           HandlerChain
	OwnerInjection    bool
	LogRequests       bool
	KubeConfig        *rest.Config
	Cache             cache.Cache
	RESTMapper        meta.RESTMapper
	ControllerMap     *controllermap.ControllerMap
	WatchedNamespaces []string
	DisableCache      bool
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
	if o.WatchedNamespaces == nil {
		return fmt.Errorf("failed to get list of watched namespaces from options")
	}

	watchedNamespaceMap := make(map[string]interface{})
	// Convert string list to map
	for _, ns := range o.WatchedNamespaces {
		watchedNamespaceMap[ns] = nil
	}

	// Create apiResources and
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(o.KubeConfig)
	if err != nil {
		return err
	}
	resources := &apiResources{
		mu:               &sync.RWMutex{},
		gvkToAPIResource: map[string]metav1.APIResource{},
		discoveryClient:  discoveryClient,
	}

	if o.Cache == nil && !o.DisableCache {
		// Need to initialize cache since we don't have one
		log.Info("Initializing and starting informer cache...")
		informerCache, err := cache.New(o.KubeConfig, cache.Options{})
		if err != nil {
			return err
		}
		stop := make(chan struct{})
		go func() {
			if err := informerCache.Start(stop); err != nil {
				log.Error(err, "Failed to start informer cache")
			}
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

	// Remove the authorization header so the proxy can correctly inject the header.
	server.Handler = removeAuthorizationHeader(server.Handler)

	if o.OwnerInjection {
		server.Handler = &injectOwnerReferenceHandler{
			next:              server.Handler,
			cMap:              o.ControllerMap,
			restMapper:        o.RESTMapper,
			watchedNamespaces: watchedNamespaceMap,
			apiResources:      resources,
		}
	} else {
		log.Info("Warning: injection of owner references and dependent watches is turned off")
	}
	if o.LogRequests {
		server.Handler = RequestLogHandler(server.Handler)
	}
	if !o.DisableCache {
		server.Handler = &cacheResponseHandler{
			next:              server.Handler,
			informerCache:     o.Cache,
			restMapper:        o.RESTMapper,
			watchedNamespaces: watchedNamespaceMap,
			cMap:              o.ControllerMap,
			injectOwnerRef:    o.OwnerInjection,
			apiResources:      resources,
		}
	}

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

// Helper function used by cache response and owner injection
func addWatchToController(owner kubeconfig.NamespacedOwnerReference, cMap *controllermap.ControllerMap, resource *unstructured.Unstructured, restMapper meta.RESTMapper, useOwnerRef bool) error {
	dataMapping, err := restMapper.RESTMapping(resource.GroupVersionKind().GroupKind(), resource.GroupVersionKind().Version)
	if err != nil {
		m := fmt.Sprintf("Could not get rest mapping for: %v", resource.GroupVersionKind())
		log.Error(err, m)
		return err

	}
	ownerGV, err := schema.ParseGroupVersion(owner.APIVersion)
	if err != nil {
		m := fmt.Sprintf("could not get broup version for: %v", owner)
		log.Error(err, m)
		return err
	}
	ownerMapping, err := restMapper.RESTMapping(schema.GroupKind{Kind: owner.Kind, Group: ownerGV.Group}, ownerGV.Version)
	if err != nil {
		m := fmt.Sprintf("could not get rest mapping for: %v", owner)
		log.Error(err, m)
		return err
	}

	dataNamespaceScoped := dataMapping.Scope.Name() != meta.RESTScopeNameRoot
	contents, ok := cMap.Get(ownerMapping.GroupVersionKind)
	if !ok {
		return errors.New("failed to find controller in map")
	}
	owMap := contents.OwnerWatchMap
	awMap := contents.AnnotationWatchMap
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(ownerMapping.GroupVersionKind)
	// Add a watch to controller
	if contents.WatchDependentResources {
		// Store watch in map
		// Use EnqueueRequestForOwner unless user has configured watching cluster scoped resources and we have to
		switch {
		case useOwnerRef:
			_, exists := owMap.Get(resource.GroupVersionKind())
			// If already watching resource no need to add a new watch
			if exists {
				return nil
			}

			owMap.Store(resource.GroupVersionKind())
			log.Info("Watching child resource", "kind", resource.GroupVersionKind(), "enqueue_kind", u.GroupVersionKind())
			// Store watch in map
			err := contents.Controller.Watch(&source.Kind{Type: resource}, &handler.EnqueueRequestForOwner{OwnerType: u})
			if err != nil {
				return err
			}
		case (!useOwnerRef && dataNamespaceScoped) || contents.WatchClusterScopedResources:
			_, exists := awMap.Get(resource.GroupVersionKind())
			// If already watching resource no need to add a new watch
			if exists {
				return nil
			}
			awMap.Store(resource.GroupVersionKind())
			typeString := fmt.Sprintf("%v.%v", owner.Kind, ownerGV.Group)
			log.Info("Watching child resource", "kind", resource.GroupVersionKind(), "enqueue_annotation_type", typeString)
			err = contents.Controller.Watch(&source.Kind{Type: resource}, &osdkHandler.EnqueueRequestForAnnotation{Type: typeString})
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func removeAuthorizationHeader(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		req.Header.Del("Authorization")
		h.ServeHTTP(w, req)
	})
}

// Helper function used by recovering dependent watches and owner ref injection.
func getRequestOwnerRef(req *http.Request) (kubeconfig.NamespacedOwnerReference, error) {
	owner := kubeconfig.NamespacedOwnerReference{}
	user, _, ok := req.BasicAuth()
	if !ok {
		return owner, errors.New("basic auth header not found")
	}
	authString, err := base64.StdEncoding.DecodeString(user)
	if err != nil {
		m := "Could not base64 decode username"
		log.Error(err, m)
		return owner, err
	}
	// Set owner to NamespacedOwnerReference, which has metav1.OwnerReference
	// as a subset along with the Namespace of the owner. Please see the
	// kubeconfig.NamespacedOwnerReference type for more information. The
	// namespace is required when creating the reconcile requests.
	json.Unmarshal(authString, &owner)
	if err := json.Unmarshal(authString, &owner); err != nil {
		m := "Could not unmarshal auth string"
		log.Error(err, m)
		return owner, err
	}
	return owner, err
}

func getGVKFromRequestInfo(r *k8sRequest.RequestInfo, restMapper meta.RESTMapper) (schema.GroupVersionKind, error) {
	gvr := schema.GroupVersionResource{
		Group:    r.APIGroup,
		Version:  r.APIVersion,
		Resource: r.Resource,
	}
	return restMapper.KindFor(gvr)
}

type apiResources struct {
	mu               *sync.RWMutex
	gvkToAPIResource map[string]metav1.APIResource
	discoveryClient  discovery.DiscoveryInterface
}

func (a *apiResources) resetResources() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	apisResourceList, err := a.discoveryClient.ServerResources()
	if err != nil {
		return err
	}

	a.gvkToAPIResource = map[string]metav1.APIResource{}

	for _, apiResource := range apisResourceList {
		gv, err := schema.ParseGroupVersion(apiResource.GroupVersion)
		if err != nil {
			return err
		}
		for _, resource := range apiResource.APIResources {
			// Names containing a "/" are subresources and should be ignored
			if strings.Contains(resource.Name, "/") {
				continue
			}
			gvk := schema.GroupVersionKind{
				Group:   gv.Group,
				Version: gv.Version,
				Kind:    resource.Kind,
			}

			a.gvkToAPIResource[gvk.String()] = resource
		}
	}

	return nil
}

func (a *apiResources) IsVirtualResource(gvk schema.GroupVersionKind) (bool, error) {
	a.mu.RLock()
	apiResource, ok := a.gvkToAPIResource[gvk.String()]
	a.mu.RUnlock()

	if !ok {
		//reset the resources
		err := a.resetResources()
		if err != nil {
			return false, err
		}
		// retry to get the resource
		a.mu.RLock()
		apiResource, ok = a.gvkToAPIResource[gvk.String()]
		a.mu.RUnlock()
		if !ok {
			return false, fmt.Errorf("unable to get api resource for gvk: %v", gvk)
		}
	}

	allVerbs := discovery.SupportsAllVerbs{
		Verbs: []string{"watch", "get", "list"},
	}

	if !allVerbs.Match(gvk.GroupVersion().String(), &apiResource) {
		return true, nil
	}

	return false, nil
}
