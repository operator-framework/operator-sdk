package generator

// apisDocTmpl is the template for apis/../doc.go
const apisDocTmpl = `// +k8s:deepcopy-gen=package
// +groupName={{.GroupName}}
package {{.Version}}
`

// apisRegisterTmpl is the template for apis/../register.go
const apisRegisterTmpl = `package {{.Version}}

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	version   = "{{.Version}}"
	groupName = "{{.GroupName}}"
)

var (
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)
	AddToScheme   = SchemeBuilder.AddToSchemes
	// SchemeGroupVersion is the group version used to register these objects.
	SchemeGroupVersion = schema.GroupVersion{Group: groupName, Version: version}
)

// addKnownTypes adds the set of types defined in this package to the supplied scheme.
func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&{{.Kind}}{},
		&{{.Kind}}List{},
	)
	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}
`
