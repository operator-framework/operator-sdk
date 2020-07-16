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

package flags

import (
	"runtime"
	"time"

	"github.com/spf13/pflag"

	"github.com/operator-framework/operator-sdk/pkg/log/zap"
)

// Flags - Options to be used by a helm operator
type Flags struct {
	ReconcilePeriod time.Duration
	WatchesFile     string
	MaxWorkers      int
	MetricsAddress  string
}

// AddTo - Add the helm operator flags to the the flagset
func (f *Flags) AddTo(flagSet *pflag.FlagSet) {
	flagSet.AddFlagSet(zap.FlagSet())
	flagSet.DurationVar(&f.ReconcilePeriod,
		"reconcile-period",
		time.Minute,
		"Default reconcile period for controllers",
	)
	flagSet.StringVar(&f.WatchesFile,
		"watches-file",
		"./watches.yaml",
		"Path to the watches file to use",
	)
	flagSet.IntVar(&f.MaxWorkers,
		"max-workers",
		runtime.NumCPU(),
		"Maximum number of workers to use",
	)
	flagSet.StringVar(&f.MetricsAddress,
		"metrics-addr",
		":8080",
		"The address the metric endpoint binds to",
	)
}
