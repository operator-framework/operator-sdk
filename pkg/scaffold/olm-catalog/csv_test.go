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

package catalog

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/operator-framework/operator-sdk/internal/util/diffutil"
	"github.com/operator-framework/operator-sdk/pkg/scaffold"
	"github.com/operator-framework/operator-sdk/pkg/scaffold/input"

	"github.com/coreos/go-semver/semver"
	"github.com/ghodss/yaml"
	olmApi "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	olmInstall "github.com/operator-framework/operator-lifecycle-manager/pkg/controller/install"
)

var testDataDir = "testdata"

func TestCSV(t *testing.T) {
	buf := &bytes.Buffer{}
	s := &scaffold.Scaffold{
		GetWriter: func(_ string, _ os.FileMode) (io.Writer, error) {
			return buf, nil
		},
	}
	cfg := &input.Config{
		Repo:           "github.com/example-org/app-operator",
		AbsProjectPath: "/home/go/src/github.com/example-org/app-operator",
		ProjectName:    "app-operator",
	}
	opVer := "1.0.1"

	err := s.Execute(cfg,
		&CSV{
			CSVVersion:     opVer,
			DeployDir:      filepath.Join(testDataDir, scaffold.DeployDir),
			ConfigFilePath: filepath.Join(testDataDir, scaffold.OlmCatalogDir, CSVConfigYamlFile),
		},
	)
	if err != nil {
		t.Fatalf("failed to execute the scaffold: (%v)", err)
	}

	if csvExp != buf.String() {
		diffs := diffutil.Diff(csvExp, buf.String())
		t.Fatalf("expected vs actual differs.\n%v", diffs)
	}
}

const csvExp = `apiVersion: operators.coreos.com/v1alpha1
kind: ClusterServiceVersion
metadata:
  creationTimestamp: null
  name: app-operator.v1.0.1
  namespace: placeholder
spec:
  apiservicedefinitions: {}
  customresourcedefinitions:
    owned:
    - kind: App
      name: apps.example.com
      version: v1alpha1
  description: Placeholder description
  displayName: App Operator
  install:
    spec:
      deployments:
      - name: app-operator
        spec:
          replicas: 1
          selector:
            matchLabels:
              name: app-operator
          strategy: {}
          template:
            metadata:
              creationTimestamp: null
              labels:
                name: app-operator
            spec:
              containers:
              - command:
                - app-operator
                env:
                - name: WATCH_NAMESPACE
                  valueFrom:
                    fieldRef:
                      fieldPath: metadata.namespace
                - name: OPERATOR_NAME
                  value: app-operator
                image: quay.io/example-org/operator:v1.0.1
                imagePullPolicy: Always
                name: app-operator
                ports:
                - containerPort: 60000
                  name: metrics
                resources: {}
              serviceAccountName: app-operator
      permissions:
      - rules:
        - apiGroups:
          - ""
          resources:
          - pods
          - services
          - endpoints
          - persistentvolumeclaims
          - events
          - configmaps
          - secrets
          verbs:
          - '*'
        - apiGroups:
          - apps
          resources:
          - deployments
          - daemonsets
          - replicasets
          - statefulsets
          verbs:
          - '*'
        - apiGroups:
          - app.example.com
          resources:
          - '*'
          verbs:
          - '*'
        serviceAccountName: app-operator
    strategy: deployment
  maturity: alpha
  provider: {}
  version: 1.0.1
status:
  certsLastUpdated: null
  certsRotateAt: null
  lastTransitionTime: null
  lastUpdateTime: null
`

func TestUpdateVersion(t *testing.T) {
	csv := new(olmApi.ClusterServiceVersion)
	if err := yaml.Unmarshal([]byte(csvExp), csv); err != nil {
		t.Fatal(err)
	}

	newOpVer := "1.0.2"
	c := &CSV{
		Input: input.Input{
			ProjectName: "app-operator",
		},
		CSVVersion: newOpVer,
	}
	if err := c.updateCSVVersions(csv); err != nil {
		t.Fatalf("update csv with ver %s: (%v)", newOpVer, err)
	}

	wantedSemver := semver.New(newOpVer)
	if !csv.Spec.Version.Equal(*wantedSemver) {
		t.Errorf("wanted csv version %v, got %v", *wantedSemver, csv.Spec.Version)
	}
	wantedName := getCSVName("app-operator", newOpVer)
	if csv.ObjectMeta.Name != wantedName {
		t.Errorf("wanted csv name %s, got %s", wantedName, csv.ObjectMeta.Name)
	}

	var resolver *olmInstall.StrategyResolver
	stratInterface, err := resolver.UnmarshalStrategy(csv.Spec.InstallStrategy)
	if err != nil {
		t.Fatal(err)
	}
	strat := stratInterface.(*olmInstall.StrategyDetailsDeployment)
	csvPodImage := strat.DeploymentSpecs[0].Spec.Template.Spec.Containers[0].Image
	// updateCSVVersions should not update podspec image.
	wantedImage := "quay.io/example-org/operator:v1.0.1"
	if csvPodImage != wantedImage {
		t.Errorf("podspec image changed from %s to %s", wantedImage, csvPodImage)
	}

	wantedReplaces := getCSVName("app-operator", "1.0.1")
	if csv.Spec.Replaces != wantedReplaces {
		t.Errorf("wanted csv replaces %s, got %s", wantedReplaces, csv.Spec.Replaces)
	}
}
