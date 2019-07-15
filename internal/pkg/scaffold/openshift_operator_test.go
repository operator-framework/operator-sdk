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
	"testing"

	"github.com/operator-framework/operator-sdk/internal/util/diffutil"
)

func TestOpenshiftOperator(t *testing.T) {
	s, buf := setupScaffoldAndWriter()
	err := s.Execute(appConfig, &OpenshiftOperator{})
	if err != nil {
		t.Fatalf("Failed to execute the scaffold: (%v)", err)
	}

	if openshiftOperatorExp != buf.String() {
		diffs := diffutil.Diff(openshiftOperatorExp, buf.String())
		t.Fatalf("Expected vs actual differs.\n%v", diffs)
	}
}

const openshiftOperatorExp = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: app-operator
spec:
  replicas: 1
  selector:
    matchLabels:
      name: app-operator
  template:
    metadata:
      labels:
        name: app-operator
    spec:
      serviceAccountName: app-operator
      volumes:
      - name: app-operator
        secret:
          secretName: app-operator
      - emptyDir: {}
        name: volume-directive-shadow
      containers:
        - name: app-operator
          # Replace this with the built image name
          image: REPLACE_IMAGE
          command:
          - app-operator
          imagePullPolicy: Always
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
              value: "app-operator"
        - args:
          - --logtostderr
          - -v=8
          - --secure-listen-address=:9393
          - --tls-cipher-suites=TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,TLS_RSA_WITH_AES_128_CBC_SHA256,TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256,TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256
          - --upstream=http://127.0.0.1:8383/
          - --tls-cert-file=/etc/tls/private/tls.crt
          - --tls-private-key-file=/etc/tls/private/tls.key
          image: quay.io/coreos/kube-rbac-proxy:v0.4.1
          name: kube-rbac-proxy-https
          ports:
          - containerPort: 9393
            name: https-metrics
          resources:
            requests:
              cpu: 10m
              memory: 40Mi
          volumeMounts:
          - mountPath: /etc/tls/private
            name: app-operator
            readOnly: false
        - args:
          - --logtostderr
          - -v=8
          - --secure-listen-address=:9696
          - --tls-cipher-suites=TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,TLS_RSA_WITH_AES_128_CBC_SHA256,TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256,TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256
          - --upstream=http://127.0.0.1:8686/
          - --tls-cert-file=/etc/tls/private/tls.crt
          - --tls-private-key-file=/etc/tls/private/tls.key
          image: quay.io/coreos/kube-rbac-proxy:v0.4.1
          name: kube-rbac-proxy-cr
          ports:
          - containerPort: 9696
            name: cr-metrics
          resources:
            requests:
              cpu: 10m
              memory: 40Mi
          volumeMounts:
          - mountPath: /etc/tls/private
            name: app-operator
            readOnly: false
`
