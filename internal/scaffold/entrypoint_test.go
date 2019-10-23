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

package scaffold

import (
	"testing"

	"github.com/operator-framework/operator-sdk/internal/util/diffutil"
)

func EntrypointTest(t *testing.T) {
	s, buf := setupScaffoldAndWriter()
	err := s.Execute(appConfig, &Entrypoint{})
	if err != nil {
		t.Fatalf("Failed to execute the scaffold: (%v)", err)
	}

	if entrypointExp != buf.String() {
		diffs := diffutil.Diff(entrypointExp, buf.String())
		t.Fatalf("Expected vs actual differs.\n%v", diffs)
	}
}

const entrypointExp = `#!/bin/sh -e

# This is documented here:
# https://docs.openshift.com/container-platform/3.11/creating_images/guidelines.html#openshift-specific-guidelines

if ! whoami &>/dev/null; then
  if [ -w /etc/passwd ]; then
    echo "${USER_NAME:-app-operator}:x:$(id -u):$(id -g):${USER_NAME:-app-operator} user:${HOME}:/sbin/nologin" >> /etc/passwd
  fi
fi

exec ${OPERATOR} $@

`
