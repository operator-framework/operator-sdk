/*
Copyright 2018 The Kubernetes Authors.
Modifications copyright 2020 The Operator-SDK Authors

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

package manager

import (
	"fmt"
	"os"
	"path/filepath"

	"sigs.k8s.io/kubebuilder/pkg/model/file"
)

var _ file.Template = &Config{}

// Config scaffolds yaml config for the manager.
type Config struct {
	file.TemplateMixin

	// Image is controller manager image name
	Image string

	// OperatorName will be used to create the pods
	OperatorName string
}

// SetTemplateDefaults implements input.Template
func (f *Config) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("config", "manager", "manager.yaml")
	}

	f.TemplateBody = configTemplate

	if f.OperatorName == "" {
		dir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("error getting working directory: %v", err)
		}
		f.OperatorName = filepath.Base(dir)
	}
	return nil
}

const configTemplate = `apiVersion: v1
kind: Namespace
metadata:
  labels:
    control-plane: controller-manager
  name: system
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: controller-manager
  namespace: system
  labels:
    control-plane: controller-manager
spec:
  selector:
    matchLabels:
      control-plane: controller-manager
  replicas: 1
  template:
    metadata:
      labels:
        control-plane: controller-manager
    spec:
      containers:
      - image: {{ .Image }}
        args:
        - "--enable-leader-election"
        - "--leader-election-id={{ .OperatorName }}"
        name: manager
        resources:
          limits:
            cpu: 100m
            memory: 90Mi
          requests:
            cpu: 100m
            memory: 60Mi
      terminationGracePeriodSeconds: 10
`
