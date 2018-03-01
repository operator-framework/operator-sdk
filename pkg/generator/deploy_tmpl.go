package generator

const operatorYamlTmpl = `apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: {{.KindPlural}}.{{.GroupName}}
spec:
  group: {{.GroupName}}
  names:
    kind: {{.Kind}}
    listKind: {{.Kind}}List
    plural: {{.KindPlural}}
    singular: {{.KindSingular}}
  scope: Namespaced
  version: {{.Version}}
---
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: {{.ProjectName}}
spec:
  replicas: 1
  template:
    metadata:
      labels:
        name: {{.ProjectName}}
    spec:
      containers:
        - name: {{.ProjectName}}
          image: {{.Image}}
          command:
          - {{.ProjectName}}
`
