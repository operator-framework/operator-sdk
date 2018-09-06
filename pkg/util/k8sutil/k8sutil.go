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

package k8sutil

import (
	"encoding/json"
	"fmt"
	"os"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	intstr "k8s.io/apimachinery/pkg/util/intstr"
	cgoscheme "k8s.io/client-go/kubernetes/scheme"
)

var (
	// scheme tracks the type registry for the sdk
	// This scheme is used to decode json data into the correct Go type based on the object's GVK
	// All types that the operator watches must be added to this scheme
	scheme      = runtime.NewScheme()
	codecs      = serializer.NewCodecFactory(scheme)
	decoderFunc = decoder
)

func init() {
	// Add the standard kubernetes [GVK:Types] type registry
	// e.g (v1,Pods):&v1.Pod{}
	metav1.AddToGroupVersion(scheme, schema.GroupVersion{Version: "v1"})
	cgoscheme.AddToScheme(scheme)
}

// UtilDecoderFunc retrieve the correct decoder from a GroupVersion
// and the schemes codec factory.
type UtilDecoderFunc func(schema.GroupVersion, serializer.CodecFactory) runtime.Decoder

// SetDecoderFunc sets a non default decoder function
// This is used as a work around to add support for unstructured objects
func SetDecoderFunc(u UtilDecoderFunc) {
	decoderFunc = u
}

func decoder(gv schema.GroupVersion, codecs serializer.CodecFactory) runtime.Decoder {
	codec := codecs.UniversalDecoder(gv)
	return codec
}

type addToSchemeFunc func(*runtime.Scheme) error

// AddToSDKScheme allows CRDs to register their types with the sdk scheme
func AddToSDKScheme(addToScheme addToSchemeFunc) {
	addToScheme(scheme)
}

// RuntimeObjectFromUnstructured converts an unstructured to a runtime object
func RuntimeObjectFromUnstructured(u *unstructured.Unstructured) (runtime.Object, error) {
	gvk := u.GroupVersionKind()
	decoder := decoderFunc(gvk.GroupVersion(), codecs)

	b, err := u.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("error running MarshalJSON on unstructured object: %v", err)
	}
	ro, _, err := decoder.Decode(b, &gvk, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decode json data with gvk(%v): %v", gvk.String(), err)
	}
	return ro, nil
}

// UnstructuredFromRuntimeObject converts a runtime object to an unstructured
func UnstructuredFromRuntimeObject(ro runtime.Object) (*unstructured.Unstructured, error) {
	b, err := json.Marshal(ro)
	if err != nil {
		return nil, fmt.Errorf("error running MarshalJSON on runtime object: %v", err)
	}
	var u unstructured.Unstructured
	if err := json.Unmarshal(b, &u.Object); err != nil {
		return nil, fmt.Errorf("failed to unmarshal json into unstructured object: %v", err)
	}
	return &u, nil
}

// UnstructuredIntoRuntimeObject unmarshalls an unstructured into a given runtime object
// TODO: https://github.com/operator-framework/operator-sdk/issues/127
func UnstructuredIntoRuntimeObject(u *unstructured.Unstructured, into runtime.Object) error {
	gvk := u.GroupVersionKind()
	decoder := decoderFunc(gvk.GroupVersion(), codecs)

	b, err := u.MarshalJSON()
	if err != nil {
		return err
	}
	_, _, err = decoder.Decode(b, &gvk, into)
	if err != nil {
		return fmt.Errorf("failed to decode json data with gvk(%v): %v", gvk.String(), err)
	}
	return nil
}

// RuntimeObjectIntoRuntimeObject unmarshalls an runtime.Object into a given runtime object
func RuntimeObjectIntoRuntimeObject(from runtime.Object, into runtime.Object) error {
	b, err := json.Marshal(from)
	if err != nil {
		return err
	}
	gvk := from.GetObjectKind().GroupVersionKind()
	decoder := decoderFunc(gvk.GroupVersion(), codecs)
	_, _, err = decoder.Decode(b, &gvk, into)
	if err != nil {
		return fmt.Errorf("failed to decode json data with gvk(%v): %v", gvk.String(), err)
	}
	return nil
}

// GetNameAndNamespace extracts the name and namespace from the given runtime.Object
// and returns a error if any of those is missing.
func GetNameAndNamespace(object runtime.Object) (string, string, error) {
	accessor := meta.NewAccessor()
	name, err := accessor.Name(object)
	if err != nil {
		return "", "", fmt.Errorf("failed to get name for object: %v", err)
	}
	namespace, err := accessor.Namespace(object)
	if err != nil {
		return "", "", fmt.Errorf("failed to get namespace for object: %v", err)
	}
	return name, namespace, nil
}

func ObjectInfo(kind, name, namespace string) string {
	return kind + ": " + namespace + "/" + name
}

// GetWatchNamespace returns the namespace the operator should be watching for changes
func GetWatchNamespace() (string, error) {
	ns, found := os.LookupEnv(WatchNamespaceEnvVar)
	if !found {
		return "", fmt.Errorf("%s must be set", WatchNamespaceEnvVar)
	}
	return ns, nil
}

// GetOperatorName return the operator name
func GetOperatorName() (string, error) {
	operatorName, found := os.LookupEnv(OperatorNameEnvVar)
	if !found {
		return "", fmt.Errorf("%s must be set", OperatorNameEnvVar)
	}
	if len(operatorName) == 0 {
		return "", fmt.Errorf("%s must not be empty", OperatorNameEnvVar)
	}
	return operatorName, nil
}

// InitOperatorService return the static service which expose operator metrics
func InitOperatorService() (*v1.Service, error) {
	operatorName, err := GetOperatorName()
	if err != nil {
		return nil, err
	}
	namespace, err := GetWatchNamespace()
	if err != nil {
		return nil, err
	}
	service := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      operatorName,
			Namespace: namespace,
			Labels:    map[string]string{"name": operatorName},
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{
					Port:     PrometheusMetricsPort,
					Protocol: v1.ProtocolTCP,
					TargetPort: intstr.IntOrString{
						Type:   intstr.String,
						StrVal: PrometheusMetricsPortName,
					},
					Name: PrometheusMetricsPortName,
				},
			},
			Selector: map[string]string{"name": operatorName},
		},
	}
	return service, nil
}
