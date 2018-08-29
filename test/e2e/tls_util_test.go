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

package e2e

import (
	"reflect"
	"strings"
	"testing"

	"github.com/operator-framework/operator-sdk/pkg/tlsutil"
	framework "github.com/operator-framework/operator-sdk/test/e2e/framework"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestBothAppAndCATLSAssetsExist ensures that when both application
// and CA TLS assets exist in the k8s cluster for a given cr,
// the GenerateCert() simply returns those to the caller.
func TestBothAppAndCATLSAssetsExist(t *testing.T) {
	f := framework.Global
	ctx := f.NewTestCtx(t)
	defer ctx.Cleanup(t)
	namespace, err := ctx.GetNamespace()
	if err != nil {
		t.Fatal(err)
	}

	// treat the Pod manifest as the input CRfor the GenerateCert().
	crKind := "Pod"
	crName := "example-pod"
	mCR := &v1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind: crKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      crName,
			Namespace: namespace,
		},
	}

	certName := "app-cert"
	appSecretName := strings.ToLower(crKind) + "-" + crName + "-" + certName
	appSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: appSecretName,
		},
	}
	appSecret, err = f.KubeClient.CoreV1().Secrets(namespace).Create(appSecret)
	if err != nil {
		t.Fatal(err)
	}

	caConfigMapAndSecretName := strings.ToLower(crKind) + "-" + crName + "-ca"
	caConfigMap := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: caConfigMapAndSecretName,
		},
	}
	caConfigMap, err = f.KubeClient.CoreV1().ConfigMaps(namespace).Create(caConfigMap)
	if err != nil {
		t.Fatal(err)
	}

	caSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: caConfigMapAndSecretName,
		},
	}
	caSecret, err = f.KubeClient.CoreV1().Secrets(namespace).Create(caSecret)
	if err != nil {
		t.Fatal(err)
	}

	cg := tlsutil.NewSDKCertGenerator(f.KubeClient)
	ccfg := &tlsutil.CertConfig{
		CertName: certName,
	}
	actualAppSecret, actualCaConfigMap, actualCaSecret, err := cg.GenerateCert(mCR, nil, ccfg)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(appSecret, actualAppSecret) {
		t.Fatalf("expect %v, got %v", appSecret, actualAppSecret)
	}
	if !reflect.DeepEqual(caConfigMap, actualCaConfigMap) {
		t.Fatalf("expect %v, got %v", caConfigMap, actualCaConfigMap)
	}
	if !reflect.DeepEqual(caSecret, actualCaSecret) {
		t.Fatalf("expect %v, got %v", caSecret, actualCaSecret)
	}
}
