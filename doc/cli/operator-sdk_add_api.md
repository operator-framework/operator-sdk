## operator-sdk add api

Adds a new api definition under pkg/apis

### Synopsis

operator-sdk add api --kind=<kind> --api-version=<group/version> creates
the api definition for a new custom resource under pkg/apis. This command
must be run from the project root directory. If the api already exists at
pkg/apis/<group>/<version> then the command will not overwrite and return
an error.

By default, this command runs Kubernetes deepcopy and CRD generators on
tagged types in all paths under pkg/apis. Go code is generated under
pkg/apis/<group>/<version>/zz_generated.deepcopy.go. CRD's are generated,
or updated if they exist for a particular group + version + kind, under
deploy/crds/<full group>_<resource>_crd.yaml; OpenAPI V3 validation YAML
is generated as a 'validation' object. Generation can be disabled with the
--skip-generation flag.

Example:

	$ operator-sdk add api --api-version=app.example.com/v1alpha1 --kind=AppService
	$ tree pkg/apis
	pkg/apis/
	├── addtoscheme_app_appservice.go
	├── apis.go
	└── app
		└── v1alpha1
			├── doc.go
			├── register.go
			├── appservice_types.go
			├── zz_generated.deepcopy.go
	$ tree deploy/crds
	├── deploy/crds/app.example.com_v1alpha1_appservice_cr.yaml
	├── deploy/crds/app.example.com_appservices_crd.yaml


```
operator-sdk add api [flags]
```

### Options

```
      --api-version string   Kubernetes APIVersion that has a format of $GROUP_NAME/$VERSION (e.g app.example.com/v1alpha1)
  -h, --help                 help for api
      --kind string          Kubernetes resource Kind name. (e.g AppService)
      --skip-generation      Skip generation of deepcopy and OpenAPI code and OpenAPI CRD specs
```

### SEE ALSO

* [operator-sdk add](operator-sdk_add.md)	 - Adds a controller or resource to the project

