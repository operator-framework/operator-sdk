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

package generator

const gopkgLockTmpl = `[[projects]]
  name = "k8s.io/api"
  packages = [
    "admissionregistration/v1alpha1",
    "admissionregistration/v1beta1",
    "apps/v1",
    "apps/v1beta1",
    "apps/v1beta2",
    "authentication/v1",
    "authentication/v1beta1",
    "authorization/v1",
    "authorization/v1beta1",
    "autoscaling/v1",
    "autoscaling/v2beta1",
    "batch/v1",
    "batch/v1beta1",
    "batch/v2alpha1",
    "certificates/v1beta1",
    "core/v1",
    "events/v1beta1",
    "extensions/v1beta1",
    "networking/v1",
    "policy/v1beta1",
    "rbac/v1",
    "rbac/v1alpha1",
    "rbac/v1beta1",
    "scheduling/v1alpha1",
    "settings/v1alpha1",
    "storage/v1",
    "storage/v1alpha1",
    "storage/v1beta1"
  ]
  revision = "acf347b865f29325eb61f4cd2df11e86e073a5ee"
  version = "kubernetes-1.9.3"

[[projects]]
  name = "k8s.io/apimachinery"
  packages = [
    "pkg/api/errors",
    "pkg/api/meta",
    "pkg/api/resource",
    "pkg/apis/meta/internalversion",
    "pkg/apis/meta/v1",
    "pkg/apis/meta/v1/unstructured",
    "pkg/apis/meta/v1alpha1",
    "pkg/conversion",
    "pkg/conversion/queryparams",
    "pkg/fields",
    "pkg/labels",
    "pkg/runtime",
    "pkg/runtime/schema",
    "pkg/runtime/serializer",
    "pkg/runtime/serializer/json",
    "pkg/runtime/serializer/protobuf",
    "pkg/runtime/serializer/recognizer",
    "pkg/runtime/serializer/streaming",
    "pkg/runtime/serializer/versioning",
    "pkg/selection",
    "pkg/types",
    "pkg/util/cache",
    "pkg/util/clock",
    "pkg/util/diff",
    "pkg/util/errors",
    "pkg/util/framer",
    "pkg/util/intstr",
    "pkg/util/json",
    "pkg/util/net",
    "pkg/util/runtime",
    "pkg/util/sets",
    "pkg/util/validation",
    "pkg/util/validation/field",
    "pkg/util/wait",
    "pkg/util/yaml",
    "pkg/version",
    "pkg/watch",
    "third_party/forked/golang/reflect"
  ]
  revision = "19e3f5aa3adca672c153d324e6b7d82ff8935f03"
  version = "kubernetes-1.9.3"

[[projects]]
  name = "k8s.io/client-go"
  packages = [
    "discovery",
    "discovery/cached",
    "dynamic",
    "kubernetes",
    "kubernetes/scheme",
    "kubernetes/typed/admissionregistration/v1alpha1",
    "kubernetes/typed/admissionregistration/v1beta1",
    "kubernetes/typed/apps/v1",
    "kubernetes/typed/apps/v1beta1",
    "kubernetes/typed/apps/v1beta2",
    "kubernetes/typed/authentication/v1",
    "kubernetes/typed/authentication/v1beta1",
    "kubernetes/typed/authorization/v1",
    "kubernetes/typed/authorization/v1beta1",
    "kubernetes/typed/autoscaling/v1",
    "kubernetes/typed/autoscaling/v2beta1",
    "kubernetes/typed/batch/v1",
    "kubernetes/typed/batch/v1beta1",
    "kubernetes/typed/batch/v2alpha1",
    "kubernetes/typed/certificates/v1beta1",
    "kubernetes/typed/core/v1",
    "kubernetes/typed/events/v1beta1",
    "kubernetes/typed/extensions/v1beta1",
    "kubernetes/typed/networking/v1",
    "kubernetes/typed/policy/v1beta1",
    "kubernetes/typed/rbac/v1",
    "kubernetes/typed/rbac/v1alpha1",
    "kubernetes/typed/rbac/v1beta1",
    "kubernetes/typed/scheduling/v1alpha1",
    "kubernetes/typed/settings/v1alpha1",
    "kubernetes/typed/storage/v1",
    "kubernetes/typed/storage/v1alpha1",
    "kubernetes/typed/storage/v1beta1",
    "pkg/version",
    "rest",
    "rest/watch",
    "tools/cache",
    "tools/clientcmd/api",
    "tools/metrics",
    "tools/pager",
    "tools/reference",
    "transport",
    "util/buffer",
    "util/cert",
    "util/flowcontrol",
    "util/integer",
    "util/workqueue"
  ]
  revision = "9389c055a838d4f208b699b3c7c51b70f2368861"
  version = "kubernetes-1.9.3"
`

const gopkgTomlTmpl = `[[override]]
  name = "k8s.io/api"
  version = "kubernetes-1.9.3"

[[override]]
  name = "k8s.io/apimachinery"
  version = "kubernetes-1.9.3"

[[override]]
  name = "k8s.io/client-go"
  version = "kubernetes-1.9.3"

[[constraint]]
  name = "github.com/operator-framework/operator-sdk"
  # The version rule is used for a specific release and the master branch for in between releases.
  branch = "master"
  # version = "=v0.0.5"
`
