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

package tlsutil

import (
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// CertType defines the type of the cert.
type CertType int

const (
	// ClientAndServingCert defines both client and serving cert.
	ClientAndServingCert CertType = iota
	// ServingCert defines a serving cert.
	ServingCert
	// ClientCert defines a client cert.
	ClientCert
)

// CertConfig configures how to generate the Cert.
type CertConfig struct {
	// CertName is the name of the cert.
	CertName string
	// Optional CertType. Serving, client or both; defaults to both.
	CertType CertType
	// Optional CommonName is the common name of the cert; defaults to "".
	CommonName string
	// Optional Organization is Organization of the cert; defaults to "".
	Organization []string
	// Optional CA Key, if user wants to provide custom CA key via a file path.
	CAKey string
	// Optional CA Certificate, if user wants to provide custom CA cert via file path.
	CACert string
	// TODO: consider to add passed in SAN fields.
}

// CertGenerator is an operator specific TLS tool that generates TLS assets for the deploying a user's application.
type CertGenerator interface {
	// GenerateCert generates a secret containing TLS encryption key and cert, a Secret
	// containing the CA key, and a ConfigMap containing the CA Certificate given the Custom
	// Resource(CR) "cr", the Kubernetes Service "Service", and the CertConfig "config".
	//
	// GenerateCert creates and manages TLS key and cert and CA with the following:
	// CA creation and management:
	// - If CA is not given:
	//  - A unique CA is generated for the CR.
	//  - CA's key is packaged into a Secret as shown below.
	//  - CA's cert is packaged in a ConfigMap as shown below.
	//  - The CA Secret and ConfigMap are created on the k8s cluster in the CR's namespace before
	//    returned to the user. The CertGenerator manages the CA Secret and ConfigMap to ensure it's
	//    unqiue per CR.
	// - If CA is given:
	//  - CA's key is packaged into a Secret as shown below.
	//  - CA's cert is packaged in a ConfigMap as shown below.
	//  - The CA Secret and ConfigMap are returned but not created in the K8s cluster in the CR's
	//    namespace. The CertGenerator doesn't manage the CA because the user controls the lifecycle
	//    of the CA.
	//
	// TLS Key and Cert Creation and Management:
	// - A unique TLS cert and key pair is generated per CR + CertConfig.CertName.
	// - The CA is used to generate and sign the TLS cert.
	// - The signing process uses the passed in "service" to set the Subject Alternative Names(SAN)
	//   for the certificate. We assume that the deployed applications are typically communicated
	//   with via a Kubernetes Service. The SAN is set to the FQDN of the service
	//   `<service-name>.<service-namespace>.svc.cluster.local`.
	// - Once TLS key and cert are created, they are packaged into a secret as shown below.
	// - Finally, the secret are created on the k8s cluster in the CR's namespace before returned to
	//   the user. The CertGenerator manages this secret to ensure that it is unique per CR +
	//   CertConfig.CertName.
	//
	// TLS encryption key and cert Secret format:
	// kind: Secret
	// apiVersion: v1
	// metadata:
	//  name: <cr-kind>-<cr-name>-<CertConfig.CertName>
	//  namespace: <cr-namespace>
	// data:
	//  tls.crt: ...
	//  tls.key: ...
	//
	// CA Certificate ConfigMap format:
	// kind: ConfigMap
	//   apiVersion: v1
	//   metadata:
	//     name: <cr-kind>-<cr-name>-ca
	//     namespace: <cr-namespace>
	//   data:
	//     ca.crt: ...
	//
	// CA Key Secret format:
	//  kind: Secret
	//  apiVersion: v1
	//  metadata:
	//   name: <cr-kind>-<cr-name>-ca
	//   namespace: <cr-namespace>
	//  data:
	//   ca.key: ..
	GenerateCert(cr runtime.Object, service *v1.Service, config *CertConfig) (*v1.Secret, *v1.ConfigMap, *v1.Secret, error)
}
