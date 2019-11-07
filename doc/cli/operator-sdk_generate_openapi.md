## operator-sdk generate openapi

Generates OpenAPI specs for API's

### Synopsis

generate openapi generates OpenAPI validation specs in Go from tagged types
in all pkg/apis/<group>/<version> directories. Go code is generated under
pkg/apis/<group>/<version>/zz_generated.openapi.go. CRD's are generated, or
updated if they exist for a particular group + version + kind, under
deploy/crds/<full group>_<resource>_crd.yaml; OpenAPI V3 validation YAML
is generated as a 'validation' object.

Example:

	$ operator-sdk generate openapi
	$ tree pkg/apis
	pkg/apis/
	└── app
		└── v1alpha1
			├── zz_generated.openapi.go
	$ tree deploy/crds
	├── deploy/crds/app.example.com_v1alpha1_appservice_cr.yaml
	├── deploy/crds/app.example.com_appservices_crd.yaml


```
operator-sdk generate openapi [flags]
```

### Options

```
  -h, --help   help for openapi
```

### SEE ALSO

* [operator-sdk generate](operator-sdk_generate.md)	 - Invokes specific generator

