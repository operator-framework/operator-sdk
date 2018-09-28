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

package scaffold

import (
	"io"
	"text/template"
)

type operator struct {
	in *OperatorInput
}

// OperatorInput is the input needed to generate a pkg/deploy/operator.yaml.
type OperatorInput struct {
	// ProjectName is the name of the operator project.
	ProjectName string
}

func NewOperatorCodegen(in *OperatorInput) Codegen {
	return &operator{in: in}
}

func (r *operator) Render(w io.Writer) error {
	t := template.New("operator.go")
	t, err := t.Parse(operatorTemplate)
	if err != nil {
		return err
	}

	return t.Execute(w, r.in)
}

const operatorTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.ProjectName}}
spec:
  replicas: 1
  selector:
    matchLabels:
      name: {{.ProjectName}}
  template:
    metadata:
      labels:
        name: {{.ProjectName}}
    spec:
      containers:
        - name: {{.ProjectName}}
          # Replace this with the built image name
          image: REPLACE_IMAGE
          ports:
          - containerPort: 60000
            name: metrics
          command:
          - {{.ProjectName}}
          imagePullPolicy: Always
          env:
            - name: WATCH_NAMESPACE
              valueFrom:
                fieldRef:
                  fieldPath: metadata.namespace
            - name: OPERATOR_NAME
              value: "{{.ProjectName}}"
`
