---
title: "operator-sdk new"
---
## operator-sdk new

Creates a new operator application

### Synopsis

The operator-sdk new command creates a new operator application and
generates a default directory layout based on the input <project-name>.

<project-name> is the project name of the new operator. (e.g app-operator)


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

  # Helm project
  $ operator-sdk new app-operator --type=helm \
  --api-version=app.example.com/v1alpha1 \
  --kind=AppService

  $ operator-sdk new app-operator --type=helm \
  --api-version=app.example.com/v1alpha1 \
  --kind=AppService \
  --helm-chart=myrepo/app

  $ operator-sdk new app-operator --type=helm \
  --helm-chart=myrepo/app

  $ operator-sdk new app-operator --type=helm \
  --helm-chart=myrepo/app \
  --helm-chart-version=1.2.3

  $ operator-sdk new app-operator --type=helm \
  --helm-chart=app \
  --helm-chart-repo=https://charts.mycompany.com/

  $ operator-sdk new app-operator --type=helm \
  --helm-chart=app \
  --helm-chart-repo=https://charts.mycompany.com/ \
  --helm-chart-version=1.2.3

  $ operator-sdk new app-operator --type=helm \
  --helm-chart=/path/to/local/chart-directories/app/

  $ operator-sdk new app-operator --type=helm \
  --helm-chart=/path/to/local/chart-archives/app-1.2.3.tgz

```

### Options

```
      --api-version string          Kubernetes apiVersion and has a format of $GROUP_NAME/$VERSION (e.g app.example.com/v1alpha1)
      --crd-version string          CRD version to generate (default "v1")
      --generate-playbook           Generate a playbook skeleton. (Only used for --type ansible)
      --helm-chart string           Initialize helm operator with existing helm chart (<URL>, <repo>/<name>, or local path). Valid only for --type helm
      --helm-chart-repo string      Chart repository URL for the requested helm chart, Valid only for --type helm
      --helm-chart-version string   Specific version of the helm chart (default is latest version). Valid only for --type helm
  -h, --help                        help for new
      --kind string                 Kubernetes resource Kind name. (e.g AppService)
      --skip-generation             Skip generation of deepcopy and OpenAPI code and OpenAPI CRD specs
      --type string                 Type of operator to initialize (choices: "ansible" or "helm")
```

### SEE ALSO

* [operator-sdk](../operator-sdk)	 - An SDK for building operators with ease

