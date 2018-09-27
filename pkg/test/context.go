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
	goctx "context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"testing"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	dynclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type TestCtx struct {
	ID         string
	CleanUpFns []finalizerFn
	Namespace  string
	t          *testing.T
}

type CleanupOptions struct {
	Timeout       time.Duration
	RetryInterval time.Duration
	SkipPolling   bool
}

type finalizerFn func() error

func NewTestCtx(t *testing.T) *TestCtx {
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
	return &TestCtx{
		ID: id,
		t:  t,
	}
}

func (ctx *TestCtx) GetID() string {
	return ctx.ID
}

func (ctx *TestCtx) Cleanup() {
	for i := len(ctx.CleanUpFns) - 1; i >= 0; i-- {
		err := ctx.CleanUpFns[i]()
		if err != nil {
			ctx.t.Errorf("a cleanup function failed with error: %v\n", err)
		}
	}
}

// CleanupNoT is a modified version of Cleanup; does not use t for logging, instead uses log
// intended for use by MainEntry, which does not have a testing.T
func (ctx *TestCtx) CleanupNoT() {
	failed := false
	for i := len(ctx.CleanUpFns) - 1; i >= 0; i-- {
		err := ctx.CleanUpFns[i]()
		if err != nil {
			failed = true
			log.Printf("a cleanup function failed with error: %v\n", err)
		}
	}
	if failed {
		log.Fatal("a cleanup function failed")
	}
}

func (ctx *TestCtx) AddFinalizerFn(fn finalizerFn) {
	ctx.CleanUpFns = append(ctx.CleanUpFns, fn)
}

// TODO: figure out how to properly retrieve object kind info from object and enable
// commented out logging

// CreateWithFinalizer uses the dynamic client to create an object and then adds a
// finalizer function to delete it when Cleanup is called. In addition to the standard
// controller-runtime client options
func (ctx *TestCtx) CreateWithFinalizer(gCtx goctx.Context, obj runtime.Object, cleanupOptions *CleanupOptions) error {
	err := Global.DynamicClient.Create(gCtx, obj)
	if err != nil {
		return err
	}
	key, err := dynclient.ObjectKeyFromObject(obj)
	//ctx.t.Logf("resource type %+v with namespace/name \"%+v\" created\n", obj.GetObjectKind(), key)
	ctx.AddFinalizerFn(func() error {
		err = Global.DynamicClient.Delete(gCtx, obj)
		if err != nil {
			return err
		}
		if cleanupOptions != nil && !cleanupOptions.SkipPolling {
			return wait.PollImmediate(time.Second*1, time.Second*5, func() (bool, error) {
				err = Global.DynamicClient.Get(gCtx, key, obj)
				if err != nil {
					if apierrors.IsNotFound(err) {
						//ctx.t.Logf("resource type %+v with namespace/name \"%+v\" successfully deleted\n", obj.GetObjectKind(), key)
						return true, nil
					}
					return false, fmt.Errorf("error encountered during deletion of resource type %v with namespace/name \"%+v\": %v", obj.GetObjectKind(), key, err)
				}
				//ctx.t.Logf("waiting for deletion of resource type %+v with namespace/name \"%+v\"\n", obj.GetObjectKind(), key)
				return false, nil
			})
		}
		return nil
	})
	return nil
}
