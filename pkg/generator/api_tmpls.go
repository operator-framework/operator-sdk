package generator

// apiDocTmpl is the template for apis/../doc.go
const apiDocTmpl = `// +k8s:deepcopy-gen=package
// +groupName={{.GroupName}}
package {{.Version}}
`

// apiRegisterTmpl is the template for apis/../register.go
const apiRegisterTmpl = `package {{.Version}}

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

// apiTypesTmpl is the template for apis/../types.go
const apiTypesTmpl = `package {{.Version}}

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type {{.Kind}}List struct {
	metav1.TypeMeta ` + "`" + `json:",inline"` + "`\n" +
	`	metav1.ListMeta ` + "`" + `json:"metadata"` + "`\n" +
	`	Items           []{{.Kind}} ` + "`" + `json:"items"` + `
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type {{.Kind}} struct {
	metav1.TypeMeta   ` + "`" + `json:",inline"` + "`\n" +
	`	metav1.ObjectMeta ` + "`" + `json:"metadata"` + "`\n" +
	`	Spec              {{.Kind}}Spec   ` + "`" + `json:"spec"` + "`\n" +
	`	Status            {{.Kind}}Status ` + "`" + `json:"status,omitempty"` + `
}

type {{.Kind}}Spec struct {
	// Fills me
}
type {{.Kind}}Status struct {
	// Fills me
}
`
