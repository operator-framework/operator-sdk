/*
Copyright 2019 The Kubernetes Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package metric

import (
	"strings"
)

// Family represents a set of metrics with the same name and help text.
type Family struct {
	Name    string
	Metrics []*Metric
}

// ByteSlice returns the given Family in its string representation.
func (f Family) ByteSlice() []byte {
	b := strings.Builder{}
	for _, m := range f.Metrics {
		b.WriteString(f.Name)
		m.Write(&b)
	}

	return []byte(b.String())
}
