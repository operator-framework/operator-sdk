---
title: Project Layout of Helm-based Operators
linkTitle: Project Layout
weight: 100
description: Overview of the files generated in Helm-based operators.
---

After creating a new operator project using `operator-sdk init --plugins=helm`,
the project directory has numerous generated folders and files. The following
table describes a basic rundown of each generated file/directory.


| File/Folders | Purpose                                                                           |
| :----------- | :-------------------------------------------------------------------------------- |
| config       | Contains kustomize manifests for deploying this operator on a Kubernetes cluster. |
| helm-charts/ | Contains a Helm chart initialized with `operator-sdk create api`.                 |
| Dockerfile   | Used to build the operator image with `make docker-build`.                        |
| watches.yaml | Contains Group, Version, Kind, and Helm chart location.                           |
| Makefile     | Contains the targets used to manage the project.                                  |
| PROJECT      | Contains meta information about the project.                                      |

[docs_helm_create]:https://helm.sh/docs/intro/using_helm/#creating-your-own-charts
