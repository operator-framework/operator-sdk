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

package test

import (
	"strings"
	"testing"

	"github.com/pborman/uuid"
)

func TestNewContextCreatesID(t *testing.T) {
	fakeNamespacedManPath := "fakePath"
	Global = &Framework{
		NamespacedManPath: &fakeNamespacedManPath,
	}

	ctx := NewContext(t)

	if strings.Index(ctx.GetID(), "osdk-e2e-") != 0 {
		t.Error("ID should start with osdk-e2e-")
	}

	idUUID := uuid.Parse(strings.Replace(ctx.GetID(), "osdk-e2e-", "", 1))
	if idUUID == nil {
		t.Error("ID should end with a UUID")
	}

	if len(ctx.GetID()) > 63 {
		t.Error("ID should be no more than 63 characters long, so that it may be a valid namespace name")
	}
}
