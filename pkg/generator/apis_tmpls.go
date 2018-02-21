package generator

// apisDocTmpl is the template for apis/../doc.go
const apisDocTmpl = `// +k8s:deepcopy-gen=package
// +groupName={{.GroupName}}
package {{.Version}}
`
