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

const ServiceFile = "service.yaml"

type Service struct {
	input.Input
}

func (s *Service) GetInput() (input.Input, error) {
	if s.Path == "" {
		s.Path = filepath.Join(fmt.Sprintf("%s/openshift/metrics/", DeployDir), ServiceFile)
	}
	s.TemplateBody = serviceTemplate

	return s.Input, nil
}

const serviceTemplate = `apiVersion: v1
kind: Service
metadata:
  annotations:
    service.alpha.openshift.io/serving-cert-secret-name: {{.ProjectName}}
  labels:
    name: {{.ProjectName}}
  name: {{.ProjectName}}
spec:
  ports:
  - name: cr-metrics
    port: 9696
    targetPort: cr-metrics
  - name: https-metrics
    port: 9393
    targetPort: https-metrics
  selector:
    name: {{.ProjectName}}
  type: ClusterIP
`
