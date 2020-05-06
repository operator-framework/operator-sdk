// Copyright 2020 The Operator-SDK Authors
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

package ansible

import (
	"path/filepath"

	"github.com/operator-framework/operator-sdk/internal/scaffold/input"
)

const MoleculeTemplatesOperatorFile = "operator.yaml.j2"

type MoleculeTemplatesOperator struct {
	input.Input
}

// GetInput - gets the input
func (m *MoleculeTemplatesOperator) GetInput() (input.Input, error) {
	if m.Path == "" {
		m.Path = filepath.Join(MoleculeTemplatesDir, MoleculeTemplatesOperatorFile)
	}
	m.TemplateBody = moleculeTemplatesOperatorAnsibleTmpl
	m.Delims = AnsibleDelims

	return m.Input, nil
}

const moleculeTemplatesOperatorAnsibleTmpl = `---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: [[.ProjectName]]
spec:
  replicas: 1
  selector:
    matchLabels:
      name: [[.ProjectName]]
  template:
    metadata:
      labels:
        name: [[.ProjectName]]
    spec:
      serviceAccountName: [[.ProjectName]]
      containers:
        - name: [[.ProjectName]]
          # Replace this with the built image name
          image: "{{ image }}"
          imagePullPolicy: "{{ pull_policy }}"
          volumeMounts:
          - mountPath: /tmp/ansible-operator/runner
            name: runner
          env:
            - name: WATCH_NAMESPACE
              valueFrom:
                fieldRef:
                  fieldPath: metadata.namespace
            - name: POD_NAME
              valueFrom:
                fieldRef:
                  fieldPath: metadata.name
            - name: OPERATOR_NAME
              value: "[[.ProjectName]]"
            - name: ANSIBLE_GATHERING
              value: explicit
          livenessProbe:
            httpGet:
              path: /healthz
              port: 6789
            initialDelaySeconds: 5
            periodSeconds: 3

      volumes:
        - name: runner
          emptyDir: {}
`
