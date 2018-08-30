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
	"testing"

	"github.com/operator-framework/operator-sdk/pkg/tlsutil"
	framework "github.com/operator-framework/operator-sdk/test/e2e/framework"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	// TLS test variables.
	crKind   = "Pod"
	crName   = "example-pod"
	certName = "app-cert"

	caConfigMapAndSecretName = tlsutil.ToCASecretAndConfigMapName(crKind, crName)
	caConfigMap              = &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: caConfigMapAndSecretName,
		},
	}
	caSecret = &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: caConfigMapAndSecretName,
		},
	}

	appSecret = &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: tlsutil.ToAppSecretName(crKind, crName, certName),
		},
	}

	ccfg = &tlsutil.CertConfig{
		CertName: certName,
	}
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

	appSecret, err := f.KubeClient.CoreV1().Secrets(namespace).Create(appSecret)
	if err != nil {
		t.Fatal(err)
	}

	caConfigMap, err := f.KubeClient.CoreV1().ConfigMaps(namespace).Create(caConfigMap)
	if err != nil {
		t.Fatal(err)
	}

	caSecret, err := f.KubeClient.CoreV1().Secrets(namespace).Create(caSecret)
	if err != nil {
		t.Fatal(err)
	}

	cg := tlsutil.NewSDKCertGenerator(f.KubeClient)
	// Use Pod as a dummy runtime object for the CR input of GenerateCert().
	mCR := &v1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind: crKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      crName,
			Namespace: namespace,
		},
	}
	actualAppSecret, actualCaConfigMap, actualCaSecret, err := cg.GenerateCert(mCR, nil, ccfg)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(appSecret, actualAppSecret) {
		t.Fatalf("expect %+v, but got %+v", appSecret, actualAppSecret)
	}
	if !reflect.DeepEqual(caConfigMap, actualCaConfigMap) {
		t.Fatalf("expect %+v, but got %+v", caConfigMap, actualCaConfigMap)
	}
	if !reflect.DeepEqual(caSecret, actualCaSecret) {
		t.Fatalf("expect %+v, but got %+v", caSecret, actualCaSecret)
	}
}

// TestOnlyAppSecretExist tests a case where the application TLS asset exists but its correspoding CA asset doesn't. In this case, CertGenerator can't genereate a new CA because it won't verify the existing application TLS cert. Therefore, CertGenerator can't proceed and returns an error to the caller.
func TestOnlyAppSecretExist(t *testing.T) {
	f := framework.Global
	ctx := f.NewTestCtx(t)
	defer ctx.Cleanup(t)
	namespace, err := ctx.GetNamespace()
	if err != nil {
		t.Fatal(err)
	}

	_, err = f.KubeClient.CoreV1().Secrets(namespace).Create(appSecret)
	if err != nil {
		t.Fatal(err)
	}

	cg := tlsutil.NewSDKCertGenerator(f.KubeClient)
	// Use Pod as a dummy runtime object for the CR input of GenerateCert().
	mCR := &v1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind: crKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      crName,
			Namespace: namespace,
		},
	}
	_, _, _, err = cg.GenerateCert(mCR, nil, ccfg)
	if err == nil {
		t.Fatal("expect error, but got none")
	}
	expErrMsg := "ca secret and configMap are not found"
	if err.Error() != expErrMsg {
		t.Fatalf("expect %v, but got %v", expErrMsg, err.Error())
	}
}
