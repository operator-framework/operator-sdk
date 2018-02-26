package k8sutil

import (
	"encoding/json"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

// RuntimeObjectFromUnstructured converts an unstructured to a runtime object
func RuntimeObjectFromUnstructured(u *unstructured.Unstructured) runtime.Object {
	b, err := json.Marshal(u.Object)
	if err != nil {
		panic(err)
	}
	var ro runtime.Object
	if err := json.Unmarshal(b, ro); err != nil {
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
