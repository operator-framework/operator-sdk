package k8sutil

import (
	"encoding/json"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

// RuntimeObjectFromUnstructured unmarshals a runtime.Object from dynamic client's unstructured
func RuntimeObjectFromUnstructured(u *unstructured.Unstructured) (runtime.Object, error) {
	b, err := json.Marshal(u.Object)
	if err != nil {
		return nil, err
	}
	var ro runtime.Object
	if err := json.Unmarshal(b, ro); err != nil {
		return nil, err
	}
	return ro, nil
}

// UnstructuredFromRuntimeObject unmarshals a runtime.Object from dynamic client's unstructured
func UnstructuredFromRuntimeObject(ro runtime.Object) (*unstructured.Unstructured, error) {
	b, err := json.Marshal(ro)
	if err != nil {
		return nil, err
	}
	var u unstructured.Unstructured
	if err := json.Unmarshal(b, &u.Object); err != nil {
		return nil, err
	}
	return &u, nil
}
