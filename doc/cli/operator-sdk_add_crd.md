## operator-sdk add crd

Adds a Custom Resource Definition (CRD) and the Custom Resource (CR) files

### Synopsis

The operator-sdk add crd command will create a Custom Resource Definition (CRD) and the Custom Resource (CR) files for the specified api-version and kind.

Generated CRD filename: <project-name>/deploy/crds/<full group>_<resource>_crd.yaml
Generated CR  filename: <project-name>/deploy/crds/<full group>_<version>_<kind>_cr.yaml

	<project-name>/deploy path must already exist
	--api-version and --kind are required flags to generate the new operator application.


```
operator-sdk add crd [flags]
```

### Options

```
      --api-version string   Kubernetes apiVersion and has a format of $GROUP_NAME/$VERSION (e.g app.example.com/v1alpha1)
  -h, --help                 help for crd
      --kind string          Kubernetes CustomResourceDefintion kind. (e.g AppService)
```

### SEE ALSO

* [operator-sdk add](operator-sdk_add.md)	 - Adds a controller or resource to the project

