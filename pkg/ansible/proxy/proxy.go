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

	k8sRequest "github.com/operator-framework/operator-sdk/pkg/ansible/proxy/requestfactory"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

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
				log.Error(err, "failed to convert request")
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
				log.Info("cache miss", "GVR", gvr)
				break
			}

			un := unstructured.Unstructured{}
			un.SetGroupVersionKind(k)
			obj := client.ObjectKey{Namespace: r.Namespace, Name: r.Name}
			err = informerCache.Get(context.Background(), obj, &un)
			if err != nil {
				// break here in case resource doesn't exist in cache but exists on APIserver
				// This is very unlikely but provides user with expected 404
				log.Info(fmt.Sprintf("cache miss: %v, %v", k, obj))
				break
			}

			i := bytes.Buffer{}
			resp, err := json.Marshal(un.Object)
			if err != nil {
				// return will give a 500
				log.Error(err, "failed to marshal data")
				http.Error(w, "", http.StatusInternalServerError)
				return
			}

			// Set X-Cache header to signal that response is served from Cache
			w.Header().Set("X-Cache", "HIT")
			json.Indent(&i, resp, "", "  ")
			_, err = w.Write(i.Bytes())
			if err != nil {
				log.Error(err, "failed to write response")
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
func InjectOwnerReferenceHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodPost {
			log.Info("injecting owner reference")
			dump, _ := httputil.DumpRequest(req, false)
			log.V(1).Info("dumping request", "RequestDump", string(dump))

			user, _, ok := req.BasicAuth()
			if !ok {
				log.Error(errors.New("basic auth header not found"), "")
				w.Header().Set("WWW-Authenticate", "Basic realm=\"Operator Proxy\"")
				http.Error(w, "", http.StatusUnauthorized)
				return
			}
			authString, err := base64.StdEncoding.DecodeString(user)
			if err != nil {
				m := "could not base64 decode username"
				log.Error(err, m)
				http.Error(w, m, http.StatusBadRequest)
				return
			}
			owner := metav1.OwnerReference{}
			json.Unmarshal(authString, &owner)

			log.V(1).Info(fmt.Sprintf("%#+v", owner))

			body, err := ioutil.ReadAll(req.Body)
			if err != nil {
				m := "could not read request body"
				log.Error(err, m)
				http.Error(w, m, http.StatusInternalServerError)
				return
			}
			data := &unstructured.Unstructured{}
			err = json.Unmarshal(body, data)
			if err != nil {
				m := "could not deserialize request body"
				log.Error(err, m)
				http.Error(w, m, http.StatusBadRequest)
				return
			}
			data.SetOwnerReferences(append(data.GetOwnerReferences(), owner))
			newBody, err := json.Marshal(data.Object)
			if err != nil {
				m := "could not serialize body"
				log.Error(err, m)
				http.Error(w, m, http.StatusInternalServerError)
				return
			}
			log.V(1).Info("serialized body", "Body", string(newBody))
			req.Body = ioutil.NopCloser(bytes.NewBuffer(newBody))
			req.ContentLength = int64(len(newBody))
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
		server.Handler = InjectOwnerReferenceHandler(server.Handler)
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
