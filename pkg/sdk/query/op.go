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

package query

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetOp represents an operation that Get can execute.
type GetOp struct {
	metaGetOptions *metav1.GetOptions
}

func NewGetOp() *GetOp {
	op := &GetOp{}
	op.setDefaults()
	return op
}

func (op *GetOp) applyOpts(opts []GetOption) {
	for _, opt := range opts {
		opt(op)
	}
}

func (op *GetOp) setDefaults() {
	if op.metaGetOptions == nil {
		op.metaGetOptions = &metav1.GetOptions{}
	}
}

// GetOption configures Get operation.
type GetOption func(*GetOp)

// WithGetOptions specifies the kubernetes metav1.GetOptions for Get operation.
func WithGetOptions(metaGetOptions *metav1.GetOptions) GetOption {
	return func(op *GetOp) {
		op.metaGetOptions = metaGetOptions
	}
}
