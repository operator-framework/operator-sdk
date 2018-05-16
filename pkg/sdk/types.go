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

package sdk

import "k8s.io/apimachinery/pkg/runtime"

// Object is the Kubernetes runtime.Object interface expected
// of all resources that the user can watch.
type Object runtime.Object

// Event is triggered when some change has happened on the watched resources.
// If created or updated, Object would be the current state and Deleted=false.
// If deleted, Object would be the last known state and Deleted=true.
type Event struct {
	Object  Object
	Deleted bool
}
