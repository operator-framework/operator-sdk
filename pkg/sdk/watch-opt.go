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

// WatchOp wraps all the options for Watch().
type watchOp struct {
	numWorkers int
}

// NewWatchOp create a new deafult WatchOp
func newWatchOp() *watchOp {
	op := &watchOp{}
	op.setDefaults()
	return op
}

func (op *watchOp) applyOpts(opts []watchOption) {
	for _, opt := range opts {
		opt(op)
	}
}

func (op *watchOp) setDefaults() {
	if op.numWorkers == 0 {
		op.numWorkers = 1
	}
}

// WatchOption configures WatchOp.
type watchOption func(*watchOp)

// WithNumWorkers sets the number of workers for the Watch() operation.
func WithNumWorkers(numWorkers int) watchOption {
	return func(op *watchOp) {
		op.numWorkers = numWorkers
	}
}
