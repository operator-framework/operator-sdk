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
	"context"
	"fmt"
	"net"
	"strconv"

	"github.com/operator-framework/operator-sdk/pkg/k8sutil"

	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var log = logf.Log.WithName("metrics")

// PrometheusPortName defines the port name used in kubernetes deployment and service resources
const PrometheusPortName = "metrics"

// ExposeMetricsPort creates a Kubernetes Service to expose the metrics port which is extracted,
// from the address passed.
func ExposeMetricsPort(address string, mgr manager.Manager) (*v1.Service, error) {
	// Split out port from address, to pass to Service object.
	// We do not need to check the validity of the port, as controller-runtime
	// would error out and we would never get to this stage.
	_, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, fmt.Errorf("failed to split metrics address %s: %v", address, err)
	}
	port64, err := strconv.ParseInt(port, 0, 32)
	if err != nil {
		return nil, fmt.Errorf("failed to parse metrics address %s: %v", address, err)
	}
	s, err := initOperatorService(int32(port64), PrometheusPortName)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize service object for metrics: %v", err)
	}
	err = createService(mgr, s)

	return s, nil
}

func createService(mgr manager.Manager, s *v1.Service) error {
	client := mgr.GetClient()
	if err := client.Create(context.TODO(), s); err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

// initOperatorService returns the static service which exposes specifed port.
func initOperatorService(port int32, portName string) (*v1.Service, error) {
	operatorName, err := k8sutil.GetOperatorName()
	if err != nil {
		return nil, err
	}
	namespace, err := k8sutil.GetOperatorNamespace()
	if err != nil {
		return nil, err
	}
	service := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      operatorName,
			Namespace: namespace,
			Labels:    map[string]string{"name": operatorName},
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{
					Port:     port,
					Protocol: v1.ProtocolTCP,
					TargetPort: intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: port,
					},
					Name: portName,
				},
			},
			Selector: map[string]string{"name": operatorName},
		},
	}
	return service, nil
}
