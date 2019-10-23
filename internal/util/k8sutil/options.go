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

package k8sutil

import (
	"k8s.io/apimachinery/pkg/fields"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NOTE: use controller-runtime's version of MatchingFields if the following
// issue is accepted as a feature:
// https://github.com/kubernetes-sigs/controller-runtime/issues/576

// MatchingFields implements the client.ListOption and client.DeleteAllOfOption
// interfaces so fields.Selector can be used directly in client.List and
// client.DeleteAllOf.
type MatchingFields struct {
	Sel fields.Selector
}

var _ client.ListOption = MatchingFields{}

func (m MatchingFields) ApplyToList(opts *client.ListOptions) {
	opts.FieldSelector = m.Sel
}

var _ client.DeleteAllOfOption = MatchingFields{}

func (m MatchingFields) ApplyToDeleteAllOf(opts *client.DeleteAllOfOptions) {
	opts.FieldSelector = m.Sel
}
