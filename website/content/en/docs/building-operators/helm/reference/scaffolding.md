---
title: Helm Based Operator Scaffolding
linkTitle: Scaffolding
weight: 20
---

After creating a new operator project using
`operator-sdk init --plugins=helm.operator-sdk.io/v1`, 
the project directory has numerous generated folders and files.
The following table describes a basic rundown of each generated file/directory.


| File/Folders | Purpose |
| :---         | :---    |
| config | Contains kustomize manifests for deploying this operator on a Kubernetes cluster. |
| helm-charts/\<kind> | Contains a Helm chart initialized using the equivalent of [`helm create`][docs_helm_create] |
| Dockerfile | Used to build the operator image with `make docker-build` |
| watches.yaml | Contains Group, Version, Kind, and Helm chart location. |
| Makefile | Contains the targets used to manage the project |
| PROJECT  | Contains the project configuration used by the tool |

[docs_helm_create]:https://helm.sh/docs/intro/using_helm/#creating-your-own-charts
