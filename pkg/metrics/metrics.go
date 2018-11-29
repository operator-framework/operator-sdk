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

package metrics

import (
	"fmt"
	"net"
	"strconv"

	"github.com/operator-framework/operator-sdk/pkg/k8sutil"

	v1 "k8s.io/api/core/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var log = logf.Log.WithName("metrics")

// ExposeMetricsPort generates a Kubernetes Service to expose the metrics port
func ExposeMetricsPort(address string) (*v1.Service, error) {
	// Split out port from address, to pass to Service object.
	// We do not need to check the validity of the port, as controller-runtime
	// would error out and we would never get to this stage.
	_, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, fmt.Errorf("failed to split metrics address %s: %v", address, err)
	}
	port64, err := strconv.ParseInt(port, 0, 32)
	if err != nil {
		return nil, fmt.Errorf("failed to split metrics address %s: %v", address, err)
	}
	service, err := k8sutil.InitOperatorService(int32(port64), PrometheusPortName)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize service object for metrics: %v", err)
	}
	return service, nil
}
