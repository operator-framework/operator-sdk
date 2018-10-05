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
	"io/ioutil"
	"reflect"
	"testing"

	"github.com/operator-framework/operator-sdk/pkg/tlsutil"
	framework "github.com/operator-framework/operator-sdk/test/e2e/framework"

	"k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

var (
	// TLS test variables.
	crKind                   = "Pod"
	crName                   = "example-pod"
	certName                 = "app-cert"
	caConfigMapAndSecretName = tlsutil.ToCASecretAndConfigMapName(crKind, crName)
	appSecretName            = tlsutil.ToAppSecretName(crKind, crName, certName)

	caConfigMap *v1.ConfigMap
	caSecret    *v1.Secret
	appSecret   *v1.Secret

	ccfg *tlsutil.CertConfig
)

// setup test variables.
func init() {
	caCertBytes, err := ioutil.ReadFile("./testdata/ca.crt")
	if err != nil {
		panic(err)
	}
	caConfigMap = &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: caConfigMapAndSecretName,
		},
		Data: map[string]string{tlsutil.TLSCACertKey: string(caCertBytes)},
	}

	caKeyBytes, err := ioutil.ReadFile("./testdata/ca.key")
	if err != nil {
		panic(err)
	}
	caSecret = &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: caConfigMapAndSecretName,
		},
		Data: map[string][]byte{tlsutil.TLSPrivateCAKeyKey: caKeyBytes},
	}

	appSecret = &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: appSecretName,
		},
	}

	ccfg = &tlsutil.CertConfig{
		CertName: certName,
	}
}

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
	actualAppSecret, actualCaConfigMap, actualCaSecret, err := cg.GenerateCert(newDummyCR(namespace), nil, ccfg)
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

// TestOnlyAppSecretExist tests a case where the application TLS asset exists but its
// correspoding CA asset doesn't. In this case, CertGenerator can't genereate a new CA because
// it won't verify the existing application TLS cert. Therefore, CertGenerator can't proceed
// and returns an error to the caller.
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
	_, _, _, err = cg.GenerateCert(newDummyCR(namespace), nil, ccfg)
	if err == nil {
		t.Fatal("expect error, but got none")
	}
	if err != tlsutil.ErrCANotFound {
		t.Fatalf("expect %v, but got %v", tlsutil.ErrCANotFound.Error(), err.Error())
	}
}

// TestOnlyCAExist tests the case where only the CA exists in the cluster;
// GenerateCert can retrieve the CA and uses it to create a new application secret.
func TestOnlyCAExist(t *testing.T) {
	f := framework.Global
	ctx := f.NewTestCtx(t)
	defer ctx.Cleanup(t)
	namespace, err := ctx.GetNamespace()
	if err != nil {
		t.Fatal(err)
	}

	_, err = f.KubeClient.CoreV1().ConfigMaps(namespace).Create(caConfigMap)
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.KubeClient.CoreV1().Secrets(namespace).Create(caSecret)
	if err != nil {
		t.Fatal(err)
	}

	cg := tlsutil.NewSDKCertGenerator(f.KubeClient)
	appSecret, _, _, err := cg.GenerateCert(newDummyCR(namespace), newAppSvc(namespace), ccfg)
	if err != nil {
		t.Fatal(err)
	}

	verifyAppSecret(t, appSecret, namespace)
}

// TestNoneOfCaAndAppSecretExist ensures that when none of the CA and Application TLS assets
// exist, GenerateCert() creates both and put them into the k8s cluster.
func TestNoneOfCaAndAppSecretExist(t *testing.T) {
	f := framework.Global
	ctx := f.NewTestCtx(t)
	defer ctx.Cleanup(t)
	namespace, err := ctx.GetNamespace()
	if err != nil {
		t.Fatal(err)
	}

	cg := tlsutil.NewSDKCertGenerator(f.KubeClient)
	appSecret, caConfigMap, caSecret, err := cg.GenerateCert(newDummyCR(namespace), newAppSvc(namespace), ccfg)
	if err != nil {
		t.Fatal(err)
	}

	verifyAppSecret(t, appSecret, namespace)
	verifyCaConfigMap(t, caConfigMap, namespace)
	verifyCASecret(t, caSecret, namespace)
}

// TestCustomCA ensures that if a user provides a custom Key and Cert and the CA and Application TLS assets
// do not exist, the GenerateCert method can use the custom CA to generate the TLS assest.
func TestCustomCA(t *testing.T) {
	f := framework.Global
	ctx := f.NewTestCtx(t)
	defer ctx.Cleanup(t)
	namespace, err := ctx.GetNamespace()
	if err != nil {
		t.Fatal(err)
	}

	cg := tlsutil.NewSDKCertGenerator(f.KubeClient)

	customConfig := &tlsutil.CertConfig{
		CertName: certName,
		CAKey:    "testdata/ca.key",
		CACert:   "testdata/ca.crt",
	}
	appSecret, _, _, err := cg.GenerateCert(newDummyCR(namespace), newAppSvc(namespace), customConfig)
	if err != nil {
		t.Fatal(err)
	}

	verifyAppSecret(t, appSecret, namespace)

	// ensure caConfigMap does not exist in k8s cluster.
	_, err = framework.Global.KubeClient.CoreV1().Secrets(namespace).Get(caConfigMapAndSecretName, metav1.GetOptions{})
	if !apiErrors.IsNotFound(err) {
		t.Fatal(err)
	}

	// ensure caConfigMap does not exist in k8s cluster.
	_, err = framework.Global.KubeClient.CoreV1().Secrets(namespace).Get(caConfigMapAndSecretName, metav1.GetOptions{})
	if !apiErrors.IsNotFound(err) {
		t.Fatal(err)
	}
}

func verifyCASecret(t *testing.T, caSecret *v1.Secret, namespace string) {
	// check if caConfigMap has the correct fields.
	if caConfigMapAndSecretName != caSecret.Name {
		t.Fatalf("expect the ca config name %v, but got %v", caConfigMapAndSecretName, caConfigMap.Name)
	}
	if namespace != caSecret.Namespace {
		t.Fatalf("expect the ca config namespace %v, but got %v", namespace, appSecret.Namespace)
	}
	if _, ok := caSecret.Data[tlsutil.TLSPrivateCAKeyKey]; !ok {
		t.Fatalf("expect the ca config to have the data field %v, but got none", tlsutil.TLSPrivateCAKeyKey)
	}

	// check if caConfigMap exists in k8s cluster.
	caSecretFromCluster, err := framework.Global.KubeClient.CoreV1().Secrets(namespace).Get(caConfigMapAndSecretName, metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	// check if caSecret returned from GenerateCert is the same as the one that exists in the k8s.
	if !reflect.DeepEqual(caSecret, caSecretFromCluster) {
		t.Fatalf("expect %+v, but got %+v", caSecret, caSecretFromCluster)
	}
}

func verifyCaConfigMap(t *testing.T, caConfigMap *v1.ConfigMap, namespace string) {
	// check if caConfigMap has the correct fields.
	if caConfigMapAndSecretName != caConfigMap.Name {
		t.Fatalf("expect the ca config name %v, but got %v", caConfigMapAndSecretName, caConfigMap.Name)
	}
	if namespace != caConfigMap.Namespace {
		t.Fatalf("expect the ca config namespace %v, but got %v", namespace, appSecret.Namespace)
	}
	if _, ok := caConfigMap.Data[tlsutil.TLSCACertKey]; !ok {
		t.Fatalf("expect the ca config to have the data field %v, but got none", tlsutil.TLSCACertKey)
	}

	// check if caConfigMap exists in k8s cluster.
	caConfigMapFromCluster, err := framework.Global.KubeClient.CoreV1().ConfigMaps(namespace).Get(caConfigMapAndSecretName, metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	// check if caConfigMap returned from GenerateCert is the same as the one that exists in the k8s.
	if !reflect.DeepEqual(caConfigMap, caConfigMapFromCluster) {
		t.Fatalf("expect %+v, but got %+v", caConfigMap, caConfigMapFromCluster)
	}
}

func verifyAppSecret(t *testing.T, appSecret *v1.Secret, namespace string) {
	// check if appSecret has the correct fields.
	if appSecretName != appSecret.Name {
		t.Fatalf("expect the secret name %v, but got %v", appSecretName, appSecret.Name)
	}
	if namespace != appSecret.Namespace {
		t.Fatalf("expect the secret namespace %v, but got %v", namespace, appSecret.Namespace)
	}
	if v1.SecretTypeTLS != appSecret.Type {
		t.Fatalf("expect the secret type %v, but got %v", v1.SecretTypeTLS, appSecret.Type)
	}
	if _, ok := appSecret.Data[v1.TLSCertKey]; !ok {
		t.Fatalf("expect the secret to have the data field %v, but got none", v1.TLSCertKey)
	}
	if _, ok := appSecret.Data[v1.TLSPrivateKeyKey]; !ok {
		t.Fatalf("expect the secret to have the data field %v, but got none", v1.TLSPrivateKeyKey)
	}

	// check if appSecret exists in k8s cluster.
	appSecretFromCluster, err := framework.Global.KubeClient.CoreV1().Secrets(namespace).Get(appSecretName, metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	// check if appSecret returned from GenerateCert is the same as the one that exists in the k8s.
	if !reflect.DeepEqual(appSecret, appSecretFromCluster) {
		t.Fatalf("expect %+v, but got %+v", appSecret, appSecretFromCluster)
	}
}

// newDummyCR returns a dummy runtime object for the CR input of GenerateCert().
func newDummyCR(namespace string) runtime.Object {
	return &v1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind: crKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      crName,
			Namespace: namespace,
		},
	}
}

func newAppSvc(namespace string) *v1.Service {
	return &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "app-service",
			Namespace: namespace,
		},
	}
}
