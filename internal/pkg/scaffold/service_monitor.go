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

package scaffold

import (
	"fmt"
	"path/filepath"

	"github.com/operator-framework/operator-sdk/internal/pkg/scaffold/input"
)

const ServiceMonitorYamlFile = "service-monitor.yaml"

type ServiceMonitor struct {
	input.Input
}

func (s *ServiceMonitor) GetInput() (input.Input, error) {
	if s.Path == "" {
		s.Path = filepath.Join(fmt.Sprintf("%s/openshift/metrics/", DeployDir), ServiceMonitorYamlFile)
	}
	s.TemplateBody = serviceMonitorTemplate
	return s.Input, nil
}

const serviceMonitorTemplate = `apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  labels:
    k8s-app: {{.ProjectName}}
  name: {{.ProjectName}}
spec:
  endpoints:
  - bearerTokenFile: /var/run/secrets/kubernetes.io/serviceaccount/token
    interval: 2m
    port: cr-metrics
    scheme: https
    scrapeTimeout: 2m
    tlsConfig:
      caFile: /etc/prometheus/configmaps/serving-certs-ca-bundle/service-ca.crt
      serverName: {{.ProjectName}}.default.svc
  - bearerTokenFile: /var/run/secrets/kubernetes.io/serviceaccount/token
    honorLabels: true
    interval: 2m
    port: https-metrics
    scheme: https
    scrapeTimeout: 2m
    tlsConfig:
      caFile: /etc/prometheus/configmaps/serving-certs-ca-bundle/service-ca.crt
      serverName: {{.ProjectName}}.default.svc
  jobLabel: {{.ProjectName}}
  selector:
    matchLabels:
        name: {{.ProjectName}}
`
