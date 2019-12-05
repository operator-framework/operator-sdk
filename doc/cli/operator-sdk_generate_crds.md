## operator-sdk generate crds

Generates CRDs for API's

### Synopsis

generate crds generates CRDs or updates them if they exist,
under deploy/crds/<full group>_<resource>_crd.yaml; OpenAPI
V3 validation YAML is generated as a 'validation' object.

Example:

	$ operator-sdk generate crds
	$ tree deploy/crds
	├── deploy/crds/app.example.com_v1alpha1_appservice_cr.yaml
	├── deploy/crds/app.example.com_appservices_crd.yaml


```
operator-sdk generate crds [flags]
```

### Options

```
  -h, --help   help for crds
```

### SEE ALSO

* [operator-sdk generate](operator-sdk_generate.md)	 - Invokes specific generator

