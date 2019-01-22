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

package scaffold

import (
	"testing"

	"github.com/operator-framework/operator-sdk/internal/util/diffutil"
)

func TestDockerfile(t *testing.T) {
	s, buf := setupScaffoldAndWriter()
	err := s.Execute(appConfig, &Dockerfile{})
	if err != nil {
		t.Fatalf("Failed to execute the scaffold: (%v)", err)
	}

	if dockerfileExp != buf.String() {
		diffs := diffutil.Diff(dockerfileExp, buf.String())
		t.Fatalf("Expected vs actual differs.\n%v", diffs)
	}
}

const dockerfileExp = `FROM alpine:3.8

ENV OPERATOR=/usr/local/bin/app-operator \
    USER_UID=1001 \
    USER_NAME=app-operator

# install operator binary
COPY build/_output/bin/app-operator ${OPERATOR}

COPY build/bin /usr/local/bin
RUN  /usr/local/bin/user_setup

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
`
