// Copyright 2021 The Operator-SDK Authors
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

package util

const WarnMessageRemovalV1beta1 = "The v1beta1 API version for CRDs and Webhooks is deprecated and is no longer offered since " +
	"Kubernetes 1.22. This flag will be removed in a future release. We " +
	"recommend that you no longer use the v1beta1 API version" +
	"More info: https://kubernetes.io/docs/reference/using-api/deprecation-guide/#v1-22"
