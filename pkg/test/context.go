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
)

// TestCtx contains the state of a test, which includes ID, namespace, and cleanup functions
type TestCtx struct {
	id         string
	cleanupFns []cleanupFn
	namespace  string
	t          *testing.T
}

// CleanupOptions allows for configuration of resource cleanup functions
type CleanupOptions struct {
	TestContext   *TestCtx
	Timeout       time.Duration
	RetryInterval time.Duration
}

type cleanupFn func() error

// NewTestCtx returns a new TestCtx object
func NewTestCtx(t *testing.T) *TestCtx {
	var prefix string
	if t != nil {
		// Use the name of the test as the prefix
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
		prefix = "operator-sdk"
	}

	// add a creation timestamp to the ID
	id := prefix + "-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	return &TestCtx{
		id: id,
		t:  t,
	}
}

// GetID returns the ID of the TestCtx
func (ctx *TestCtx) GetID() string {
	return ctx.id
}

// Cleanup runs all the TestCtx's cleanup function in reverse order of their insertion
func (ctx *TestCtx) Cleanup() {
	for i := len(ctx.cleanupFns) - 1; i >= 0; i-- {
		err := ctx.cleanupFns[i]()
		if err != nil {
			if ctx.t != nil {
				ctx.t.Errorf("A cleanup function failed with error: (%v)\n", err)
			} else {
				log.Errorf("A cleanup function failed with error: (%v)", err)
			}
		}
	}
}

// AddCleanupFn adds a new cleanup function to the TestCtx
func (ctx *TestCtx) AddCleanupFn(fn cleanupFn) {
	ctx.cleanupFns = append(ctx.cleanupFns, fn)
}
