## operator-sdk generate k8s

Generates Kubernetes code for custom resource

### Synopsis

k8s generator generates code for custom resources given the API
specs in pkg/apis/<group>/<version> directories to comply with kube-API
requirements. Go code is generated under
pkg/apis/<group>/<version>/zz_generated.deepcopy.go.
Example:

	$ operator-sdk generate k8s
	$ tree pkg/apis
	pkg/apis/
	└── app
		└── v1alpha1
			├── zz_generated.deepcopy.go


```
operator-sdk generate k8s [flags]
```

### Options

```
  -h, --help   help for k8s
```

### SEE ALSO

* [operator-sdk generate](operator-sdk_generate.md)	 - Invokes specific generator

