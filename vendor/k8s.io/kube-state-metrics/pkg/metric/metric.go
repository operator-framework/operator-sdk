/*
Copyright 2018 The Kubernetes Authors All rights reserved.

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
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"
)

const (
	initialNumBufSize = 24
)

var (
	numBufPool = sync.Pool{
		New: func() interface{} {
			b := make([]byte, 0, initialNumBufSize)
			return &b
		},
	}
)

// Type represents the type of a metric e.g. a counter. See
// https://prometheus.io/docs/concepts/metric_types/.
type Type string

// Gauge defines a Prometheus gauge.
var Gauge Type = "gauge"

// Counter defines a Prometheus counter.
var Counter Type = "counter"

// Metric represents a single time series.
type Metric struct {
	// The name of a metric is injected by its family to reduce duplication.
	LabelKeys   []string
	LabelValues []string
	Value       float64
}

func (m *Metric) Write(s *strings.Builder) {
	if len(m.LabelKeys) != len(m.LabelValues) {
		panic(fmt.Sprintf(
			"expected labelKeys %q to be of same length as labelValues %q",
			m.LabelKeys, m.LabelValues,
		))
	}

	labelsToString(s, m.LabelKeys, m.LabelValues)
	s.WriteByte(' ')
	writeFloat(s, m.Value)
	s.WriteByte('\n')
}

func labelsToString(m *strings.Builder, keys, values []string) {
	if len(keys) > 0 {
		var separator byte = '{'

		for i := 0; i < len(keys); i++ {
			m.WriteByte(separator)
			m.WriteString(keys[i])
			m.WriteString("=\"")
			escapeString(m, values[i])
			m.WriteByte('"')
			separator = ','
		}

		m.WriteByte('}')
	}
}

var (
	escapeWithDoubleQuote = strings.NewReplacer("\\", `\\`, "\n", `\n`, "\"", `\"`)
)

// escapeString replaces '\' by '\\', new line character by '\n', and '"' by
// '\"'.
// Taken from github.com/prometheus/common/expfmt/text_create.go.
func escapeString(m *strings.Builder, v string) {
	escapeWithDoubleQuote.WriteString(m, v)
}

// writeFloat is equivalent to fmt.Fprint with a float64 argument but hardcodes
// a few common cases for increased efficiency. For non-hardcoded cases, it uses
// strconv.AppendFloat to avoid allocations, similar to writeInt.
// Taken from github.com/prometheus/common/expfmt/text_create.go.
func writeFloat(w *strings.Builder, f float64) {
	switch {
	case f == 1:
		w.WriteByte('1')
	case f == 0:
		w.WriteByte('0')
	case f == -1:
		w.WriteString("-1")
	case math.IsNaN(f):
		w.WriteString("NaN")
	case math.IsInf(f, +1):
		w.WriteString("+Inf")
	case math.IsInf(f, -1):
		w.WriteString("-Inf")
	default:
		bp := numBufPool.Get().(*[]byte)
		*bp = strconv.AppendFloat((*bp)[:0], f, 'g', -1, 64)
		w.Write(*bp)
		numBufPool.Put(bp)
	}
}
