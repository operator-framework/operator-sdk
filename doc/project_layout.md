# Project Scaffolding Layout

After creating a new operator project using 
`operator-sdk new`, the project directory has numerous generated folders and files. The following table describes a basic rundown of each generated file/directory.


| File/Folders   | Purpose                           |
| :---           | :--- |
| Gopkg.toml Gopkg.lock | The [Go Dep][dep] manifests that describe the external dependencies of this operator. |
| cmd       | Contains `main.go` which is the entry point to initialize and start this operator using the operator-sdk APIs. |
| config | Contains metadata about state of this project such as project name, kind, api-version, and so forth. The operator-sdk commands use this metadata to perform actions that require knowing the state. |
| deploy | Contains a generic set of kubernetes manifests for deploying this operator on a kubernetes cluster. |
| pkg/apis | Contains the directory tree that defines the APIs and types of Custom Resource Definitions(CRD). These files allow the sdk to do code generation for CRD types and register the schemes for all types in order to correctly decode Custom Resource objects. |
| pkg/stub | Contains `handler.go` which is the place for a user to write all the operating business logic. |
| tmp | Contains scripts that the operator-sdk uses for build and code generation. |
| vendor | The golang [vendor][Vendor] folder that contains the local copies of the external dependencies that satisfy the imports of this project. [Go Dep][dep] manages the vendor directly. |

[Vendor]: https://golang.org/cmd/go/#hdr-Vendor_Directories
[dep]: https://github.com/golang/dep
