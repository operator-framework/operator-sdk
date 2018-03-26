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

package action

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DeleteOp wraps all the options for Delete().
type DeleteOp struct {
	metaDeleteOptions *metav1.DeleteOptions
}

// DeleteOption configures DeleteOp.
type DeleteOption func(*DeleteOp)

func NewDeleteOp() *DeleteOp {
	op := &DeleteOp{}
	op.setDefaults()
	return op
}

func (op *DeleteOp) applyOpts(opts []DeleteOption) {
	for _, opt := range opts {
		opt(op)
	}
}

func (op *DeleteOp) setDefaults() {
	if op.metaDeleteOptions == nil {
		op.metaDeleteOptions = &metav1.DeleteOptions{}
	}
}

// WithDeleteOptions sets the metav1.DeleteOptions for the Delete() operation.
func WithDeleteOptions(metaDeleteOptions *metav1.DeleteOptions) DeleteOption {
	return func(op *DeleteOp) {
		op.metaDeleteOptions = metaDeleteOptions
	}
}
