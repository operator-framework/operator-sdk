## operator-sdk add controller

Adds a new controller pkg

### Synopsis

operator-sdk add controller --kind=<kind> --api-version=<group/version> creates a new
controller pkg under pkg/controller/<kind> that, by default, reconciles on a custom resource for the specified apiversion and kind.
The controller will expect to use the custom resource type that should already be defined under pkg/apis/<group>/<version>
via the "operator-sdk add api --kind=<kind> --api-version=<group/version>" command.
This command must be run from the project root directory.
If the controller pkg for that Kind already exists at pkg/controller/<kind> then the command will not overwrite and return an error.

Example:

	$ operator-sdk add controller --api-version=app.example.com/v1alpha1 --kind=AppService
	$ tree pkg/controller
	pkg/controller/
	├── add_appservice.go
	├── appservice
	│   └── appservice_controller.go
	└── controller.go



```
operator-sdk add controller [flags]
```

### Options

```
      --api-version string         Kubernetes APIVersion that has a format of $GROUP_NAME/$VERSION (e.g app.example.com/v1alpha1)
      --custom-api-import string   External Kubernetes resource import path of the form "host.com/repo/path[=import_identifier]". import_identifier is optional
  -h, --help                       help for controller
      --kind string                Kubernetes resource Kind name. (e.g AppService)
```

### SEE ALSO

* [operator-sdk add](operator-sdk_add.md)	 - Adds a controller or resource to the project

