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

package test

import (
	"strconv"
	"strings"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/restmapper"
)

type TestCtx struct { //nolint:golint
	// todo(camilamacedo86): The no lint here is for type name will be used as test.TestCtx by other packages, and that stutters; consider calling this Ctx (golint)
	// However, was decided to not move forward with it now in order to not introduce breakchanges with the task to add the linter. We should to do it after.
	id         string
	cleanupFns []cleanupFn
	namespace  string
	t          *testing.T

	namespacedManPath string
	client            *frameworkClient
	kubeclient        kubernetes.Interface
	restMapper        *restmapper.DeferredDiscoveryRESTMapper
}

type CleanupOptions struct {
	TestContext   *TestCtx
	Timeout       time.Duration
	RetryInterval time.Duration
}

type cleanupFn func() error

func (f *Framework) newTestCtx(t *testing.T) *TestCtx {
	var prefix string
	if t != nil {
		// TestCtx is used among others for namespace names where '/' is forbidden
		prefix = strings.TrimPrefix(
			strings.Replace(
				strings.ToLower(t.Name()),
				"/",
				"-",
				-1,
			),
			"test",
		)
	} else {
		prefix = "main"
	}

	id := prefix + "-" + strconv.FormatInt(time.Now().Unix(), 10)

	var namespace string
	if f.singleNamespaceMode {
		namespace = f.Namespace
	}
	return &TestCtx{
		id:                id,
		t:                 t,
		namespace:         namespace,
		namespacedManPath: *f.NamespacedManPath,
		client:            f.Client,
		kubeclient:        f.KubeClient,
		restMapper:        f.restMapper,
	}
}

func NewTestCtx(t *testing.T) *TestCtx {
	return Global.newTestCtx(t)
}

func (ctx *TestCtx) GetID() string {
	return ctx.id
}

func (ctx *TestCtx) Cleanup() {
	failed := false
	for i := len(ctx.cleanupFns) - 1; i >= 0; i-- {
		err := ctx.cleanupFns[i]()
		if err != nil {
			failed = true
			if ctx.t != nil {
				ctx.t.Errorf("A cleanup function failed with error: (%v)\n", err)
			} else {
				log.Errorf("A cleanup function failed with error: (%v)", err)
			}
		}
	}
	if ctx.t == nil && failed {
		log.Fatal("A cleanup function failed")
	}
}

func (ctx *TestCtx) AddCleanupFn(fn cleanupFn) {
	ctx.cleanupFns = append(ctx.cleanupFns, fn)
}
