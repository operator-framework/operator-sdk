package k8sutil

import (
	"encoding/json"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	cgoscheme "k8s.io/client-go/kubernetes/scheme"
)

var (
	// scheme tracks the type registry for the sdk
	// This scheme is used to decode json data into the correct Go type based on the object's GVK
	// All types that the operator watches must be added to this scheme
	scheme = runtime.NewScheme()
	codecs = serializer.NewCodecFactory(scheme)
)

func init() {
	// Add the standard kubernetes [GVK:Types] type registry
	// e.g (v1,Pods):&v1.Pod{}
	metav1.AddToGroupVersion(scheme, schema.GroupVersion{Version: "v1"})
	cgoscheme.AddToScheme(scheme)
}

func decoder(gv schema.GroupVersion) runtime.Decoder {
	codec := codecs.UniversalDecoder(gv)
	return codec
}

type addToSchemeFunc func(*runtime.Scheme) error

// AddToSDKScheme allows CRDs to register their types with the sdk scheme
func AddToSDKScheme(addToScheme addToSchemeFunc) {
	addToScheme(scheme)
}

// RuntimeObjectFromUnstructured converts an unstructured to a runtime object
func RuntimeObjectFromUnstructured(u *unstructured.Unstructured) runtime.Object {
	gvk := u.GroupVersionKind()
	decoder := decoder(gvk.GroupVersion())

	b, err := u.MarshalJSON()
	if err != nil {
		panic(err)
	}
	ro, _, err := decoder.Decode(b, &gvk, nil)
	if err != nil {
		err = fmt.Errorf("failed to decode json data with gvk(%v): %v", gvk.String(), err)
		panic(err)
	}
	return ro
}

// UnstructuredFromRuntimeObject converts a runtime object to an unstructured
func UnstructuredFromRuntimeObject(ro runtime.Object) *unstructured.Unstructured {
	b, err := json.Marshal(ro)
	if err != nil {
		panic(err)
	}
	var u unstructured.Unstructured
	if err := json.Unmarshal(b, &u.Object); err != nil {
		panic(err)
	}
	return &u
}
