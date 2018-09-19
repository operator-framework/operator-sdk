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
	"io/ioutil"
	"net/http"
	"net/http/httputil"

	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/rest"
)

// InjectOwnerReferenceHandler will handle proxied requests and inject the
// owner refernece found in the authorization header. The Authorization is
// then deleted so that the proxy can re-set with the correct authorization.
func InjectOwnerReferenceHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodPost {
			logrus.Info("injecting owner reference")
			dump, _ := httputil.DumpRequest(req, false)
			logrus.Debugf(string(dump))

			user, _, ok := req.BasicAuth()
			if !ok {
				logrus.Error("basic auth header not found")
				w.Header().Set("WWW-Authenticate", "Basic realm=\"Operator Proxy\"")
				http.Error(w, "", http.StatusUnauthorized)
				return
			}
			authString, err := base64.StdEncoding.DecodeString(user)
			if err != nil {
				m := "could not base64 decode username"
				logrus.Errorf("%s: %s", err.Error())
				http.Error(w, m, http.StatusBadRequest)
				return
			}
			owner := metav1.OwnerReference{}
			json.Unmarshal(authString, &owner)

			logrus.Debugf("%#+v", owner)

			body, err := ioutil.ReadAll(req.Body)
			if err != nil {
				m := "could not read request body"
				logrus.Errorf("%s: %s", err.Error())
				http.Error(w, m, http.StatusInternalServerError)
				return
			}
			data := &unstructured.Unstructured{}
			err = json.Unmarshal(body, data)
			if err != nil {
				m := "could not deserialize request body"
				logrus.Errorf("%s: %s", err.Error())
				http.Error(w, m, http.StatusBadRequest)
				return
			}
			data.SetOwnerReferences(append(data.GetOwnerReferences(), owner))
			newBody, err := json.Marshal(data.Object)
			if err != nil {
				m := "could not serialize body"
				logrus.Errorf("%s: %s", err.Error())
				http.Error(w, m, http.StatusInternalServerError)
				return
			}
			logrus.Debugf(string(newBody))
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
}

// RunProxy will start a proxy server in a go routine and return on the error
// channel if something is not correct on startup.
func RunProxy(done chan error, o Options) {
	server, err := newServer("/", o.KubeConfig)
	if err != nil {
		done <- err
		return
	}
	if o.Handler != nil {
		server.Handler = o.Handler(server.Handler)
	}

	if !o.NoOwnerInjection {
		server.Handler = InjectOwnerReferenceHandler(server.Handler)
	}
	l, err := server.Listen(o.Address, o.Port)
	if err != nil {
		done <- err
		return
	}
	go func() {
		logrus.Infof("Starting to serve on %s\n", l.Addr().String())
		done <- server.ServeOnListener(l)
	}()
}
