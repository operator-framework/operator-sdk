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
	"log"
	"strconv"
	"strings"
	"testing"
	"time"
)

type TestCtx struct {
	ID         string
	CleanUpFns []finalizerFn
	Namespace  string
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
	}
}

func (ctx *TestCtx) GetID() string {
	return ctx.ID
}

func (ctx *TestCtx) Cleanup(t *testing.T) {
	for i := len(ctx.CleanUpFns) - 1; i >= 0; i-- {
		err := ctx.CleanUpFns[i]()
		if err != nil {
			t.Errorf("a cleanup function failed with error: %v\n", err)
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
