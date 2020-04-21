---
title: "operator-sdk add api"
---
## operator-sdk add api

Adds a new api definition under pkg/apis

### Synopsis

operator-sdk add api --kind=<kind> --api-version<group/version> 
creates an API definition for a new custom resource.
This command must be run from the project root directory.

For Go-based operators:

  - Creates the api definition for a new custom resource under pkg/apis.
  - By default, this command runs Kubernetes deepcopy and CRD generators on
  tagged types in all paths under pkg/apis. Go code is generated under
  pkg/apis/<group>/<version>/zz_generated.deepcopy.go. Generation can be disabled with the
  --skip-generation flag for Go-based operators.

For Ansible-based operators:

  - Creates resource folder under /roles.
  - watches.yaml is updated with new resource.
  - deploy/role.yaml will be updated with apiGroup for new API.

For Helm-based operators:
  - Creates resource folder under /helm-charts.
  - watches.yaml is updated with new resource.
  - deploy/role.yaml will be updated to reflact new rules for the incoming API.

CRD's are generated, or updated if they exist for a particular group + version + kind, under
deploy/crds/<full group>_<resource>_crd.yaml; OpenAPI V3 validation YAML
is generated as a 'validation' object.

```
operator-sdk add api [flags]
```

### Examples

```
  # Create a new API, under an existing project. This command must be run from the project root directory.
# Go Example:
  $ operator-sdk add api --api-version=app.example.com/v1alpha1 --kind=AppService

# Ansible Example
  $ operator-sdk add api  \
  --api-version=app.example.com/v1alpha1 \
  --kind=AppService

# Helm Example:
  $ operator-sdk add api \
  --api-version=app.example.com/v1alpha1 \
  --kind=AppService

  $ operator-sdk add api \
  --api-version=app.example.com/v1alpha1 \
  --kind=AppService
  --helm-chart=myrepo/app

  $ operator-sdk add api \
  --helm-chart=myrepo/app

  $ operator-sdk add api \
  --helm-chart=myrepo/app \
  --helm-chart-version=1.2.3

  $ operator-sdk add api \
  --helm-chart=app \
  --helm-chart-repo=https://charts.mycompany.com/

  $ operator-sdk add api \
  --helm-chart=app \
  --helm-chart-repo=https://charts.mycompany.com/ \
  --helm-chart-version=1.2.3

  $ operator-sdk add api \
  --helm-chart=/path/to/local/chart-directories/app/

  $ operator-sdk add api \
  --helm-chart=/path/to/local/chart-archives/app-1.2.3.tgz

```

### Options

```
      --api-version string          Kubernetes apiVersion and has a format of $GROUP_NAME/$VERSION (e.g app.example.com/v1alpha1)
      --crd-version string          CRD version to generate (default "v1")
      --helm-chart string           Initialize helm operator with existing helm chart (<URL>, <repo>/<name>, or local path). Valid only for --type helm
      --helm-chart-repo string      Chart repository URL for the requested helm chart, Valid only for --type helm
      --helm-chart-version string   Specific version of the helm chart (default is latest version). Valid only for --type helm
  -h, --help                        help for api
      --kind string                 Kubernetes resource Kind name. (e.g AppService)
      --skip-generation             Skip generation of deepcopy and OpenAPI code and OpenAPI CRD specs
```

### SEE ALSO

* [operator-sdk add](../operator-sdk_add)	 - Adds a controller or resource to the project

