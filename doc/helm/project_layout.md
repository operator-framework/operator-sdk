# Project Scaffolding Layout

After creating a new operator project using `operator-sdk new --type helm`,
the project directory has numerous generated folders and files. The following
table describes a basic rundown of each generated file/directory.

| File/Folders | Purpose |
| :---         | :---    |
| deploy | Contains a generic set of Kubernetes manifests for deploying this operator on a Kubernetes cluster. |
| helm-charts/\<kind> | Contains a Helm chart initialized using the equivalent of [`helm create`](https://docs.helm.sh/helm/#helm-create) |
| build | Contains scripts that the operator-sdk uses for build and initialization. |
| watches.yaml | Contains Group, Version, Kind, and Helm chart location. |
