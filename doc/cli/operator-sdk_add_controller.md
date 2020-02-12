## operator-sdk add controller

Adds a new controller pkg

### Synopsis


Add a new controller package to your operator project.

This command creates a new controller package under pkg/controller/<kind> that, by default, reconciles on a custom resource for the specified apiversion and kind. The controller will expect to use the custom resource type that should already be defined under pkg/apis/<group>/<version> via the "operator-sdk add api" command. 

Note that, if the controller pkg for that Kind already exists at pkg/controller/<kind> then the command will not overwrite and return an error.

This command MUST be run from the project root directory.

```
operator-sdk add controller [flags]
```

### Examples

```

The following example will create a controller to manage, watch and reconcile as primary resource the <v1.AppService> from the domain <app.example.com>.    

Example:

	$ operator-sdk add controller --api-version=app.example.com/v1 --kind=AppService
	$ tree pkg/controller
	pkg/controller/
	├── add_appservice.go
	├── appservice
	│   └── appservice_controller.go
	└── controller.go

The following example will create a controller to manage, watch and reconcile as a primary resource the <v1.Deployment> from the domain <k8s.io.api>, which is not defined in the project (external). Note that, it can be used to create controllers for any External API. 	

Example:

	$ operator-sdk add controller  --api-version=k8s.io.api/v1 --kind=Deployment  --custom-api-import=k8s.io/api/apps
	$ tree pkg/controller
	pkg/controller/
	├── add_deployment.go
	├── deployment
	│   └── deployment_controller.go 
	└── controller.go
		
```

### Options

```
      --api-version string         Kubernetes APIVersion that has a format of $GROUP_NAME/$VERSION (e.g app.example.com/v1alpha1)
      --custom-api-import string   The External API import path of the form "host.com/repo/path[=import_identifier]" Note that import_identifier is optional. ( E.g. --custom-api-import=k8s.io/api/apps )
  -h, --help                       help for controller
      --kind string                Kubernetes resource Kind name. (e.g AppService)
```

### SEE ALSO

* [operator-sdk add](operator-sdk_add.md)	 - Adds a controller or resource to the project

