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
	"path/filepath"

	"github.com/operator-framework/operator-sdk/pkg/scaffold/input"
)

const GenerateGroupsScriptFile = "generate-groups.sh"

type GenerateGroupsScript struct {
	input.Input
}

func (s *GenerateGroupsScript) GetInput() (input.Input, error) {
	if s.Path == "" {
		s.Path = filepath.Join(BuildScriptDir, GenerateGroupsScriptFile)
	}
	s.IsExec = true
	s.IfExistsAction = input.Overwrite
	s.TemplateBody = genGroupsScriptTmpl
	return s.Input, nil
}

const genGroupsScriptTmpl = `#!/usr/bin/env bash

# Modified from:
# https://github.com/kubernetes/code-generator/blob/878d878e0a09473450edace96eff33de24076488/generate-groups.sh

set -o errexit
set -o nounset
set -o pipefail

if [ "$#" -lt 4 ]; then
  echo "too few args"
  exit 1
fi

GENS="$1"
OUTPUT_PKG="$2"
APIS_PKG="$3"
GROUPS_WITH_VERSIONS="$4"
shift 4

(
  CODEGEN_REPO="k8s.io/code-generator"
  go get -d -u "$CODEGEN_REPO" > /dev/null 2>&1 || true
  cd "${GOPATH}/src/${CODEGEN_REPO}"
	git checkout -q kubernetes-1.11.2
  go install ${GOFLAGS:-} ./cmd/{defaulter-gen,client-gen,lister-gen,informer-gen,deepcopy-gen}
)

function codegen::join() { local IFS="$1"; shift; echo "$*"; }

FQ_APIS=()
for GVs in ${GROUPS_WITH_VERSIONS}; do
  IFS=: read G Vs <<<"${GVs}"
  for V in ${Vs//,/ }; do
    FQ_APIS+=(${APIS_PKG}/${G}/${V})
  done
done

if [ "${GENS}" = "all" ] || grep -qw "deepcopy" <<<"${GENS}"; then
  echo "Generating deepcopy funcs"
  ${GOPATH}/bin/deepcopy-gen \
            --input-dirs $(codegen::join , "${FQ_APIS[@]}") \
            -O zz_generated.deepcopy \
            --bounding-dirs ${APIS_PKG} \
            "$@"
fi

if [ "${GENS}" = "all" ] || grep -qw "client" <<<"${GENS}"; then
  echo "Generating clientset for ${GROUPS_WITH_VERSIONS} at ${OUTPUT_PKG}/${CLIENTSET_PKG_NAME:-clientset}"
  ${GOPATH}/bin/client-gen \
            --clientset-name ${CLIENTSET_NAME_VERSIONED:-versioned} \
            --input-base "" \
            --input $(codegen::join , "${FQ_APIS[@]}") \
            --output-package ${OUTPUT_PKG}/${CLIENTSET_PKG_NAME:-clientset} \
            "$@"
fi

if [ "${GENS}" = "all" ] || grep -qw "lister" <<<"${GENS}"; then
  echo "Generating listers for ${GROUPS_WITH_VERSIONS} at ${OUTPUT_PKG}/listers"
  ${GOPATH}/bin/lister-gen \
            --input-dirs $(codegen::join , "${FQ_APIS[@]}") \
            --output-package ${OUTPUT_PKG}/listers \
            "$@"
fi

if [ "${GENS}" = "all" ] || grep -qw "informer" <<<"${GENS}"; then
  echo "Generating informers for ${GROUPS_WITH_VERSIONS} at ${OUTPUT_PKG}/informers"
  ${GOPATH}/bin/informer-gen \
           --input-dirs $(codegen::join , "${FQ_APIS[@]}") \
           --versioned-clientset-package ${OUTPUT_PKG}/${CLIENTSET_PKG_NAME:-clientset}/${CLIENTSET_NAME_VERSIONED:-versioned} \
           --listers-package ${OUTPUT_PKG}/listers \
           --output-package ${OUTPUT_PKG}/informers \
           "$@"
fi
`
