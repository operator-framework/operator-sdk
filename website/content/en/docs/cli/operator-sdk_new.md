---
title: "operator-sdk new"
---
## operator-sdk new

Creates a new operator application

### Synopsis

The operator-sdk new command creates a new operator application and
generates a default directory layout based on the input &lt;project-name&gt;.

&lt;project-name&gt; is the project name of the new operator. (e.g app-operator)


```
operator-sdk new <project-name> [flags]
```

### Examples

```
  # Create a new project directory
  $ mkdir $HOME/projects/example.com/
  $ cd $HOME/projects/example.com/

  # Ansible project
  $ operator-sdk new app-operator --type=ansible \
    --api-version=app.example.com/v1alpha1 \
    --kind=AppService

```

### Options

```
      --api-version string   Kubernetes apiVersion and has a format of $GROUP_NAME/$VERSION (e.g app.example.com/v1alpha1)
      --crd-version string   CRD version to generate (default "v1")
      --generate-playbook    Generate a playbook skeleton. (Only used for --type ansible)
  -h, --help                 help for new
      --kind string          Kubernetes resource Kind name. (e.g AppService)
      --skip-generation      Skip generation of deepcopy and OpenAPI code and OpenAPI CRD specs
```

### Options inherited from parent commands

```
      --verbose   Enable verbose logging
```

### SEE ALSO

* [operator-sdk](../operator-sdk)	 - Development kit for building Kubernetes extensions and tools.

