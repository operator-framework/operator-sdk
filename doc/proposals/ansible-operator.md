## Ansible Operator Proposal

### Background

Not everyone is a golang developer, and therefore gaining adoption for the operator-sdk is capped by the number of golang developers. Also, tooling for kubernetes in other languages is lacking support for things such as informers,  caches, and listers.  

Operators purpose is to codify the operations of an application on kubernetes. [Ansible](https://www.ansible.com/) is already an industry standard tool for automation and is a good fit for the kind of work that kubernetes operators need to do. Adding the ability for users of the SDK to choose which between ansible and golang to follow will increase the number of potential users, and will grant existing users even more behavior. 

### Goals

The goal of the Ansible Operator will be to create a fully functional framework for Ansible developers to create operators. It will also expose a library for golang users to use ansible in their operator if they so choose. These two goals in conjunction will allow users to select the best technology for their project or skillset. 

### New Operator Type

This proposal creates a new type of operator called `ansible`.  The new type is used to tell the tooling to act on that type of operator. 

### Package Structure
Packages will be added to the operator-sdk. These packages are designed to be usable by the end user if they choose to and should have a well documented public API. The proposed packages are:
* /operator-sdk/pkg/ansible/handler
  * Will contain a new Handler Interface to allow users to define how to handle ansible events.
  * Will contain a wrapper type that conforms to the sdk.Handler, for a user to use with standard operator-sdk workflows. The wrapper type has default methods for everything, but they are overridable.
* /operator-sdk/pkg/ansible/runner
  * Contains the runner types and interfaces
  * Implementation is behind an internal package (/operator-sdk/pkg/ansible/runner/internal)
  * New Methods are exposed and are the main way a user interacts with the package
  * Runner interface for running ansible from the operator.
  * NewForWatchers - the method that returns a map of GVK to Runner types based on the watchers file.
  * NewPlaybookRunner - the method that returns a new Runner for a playbook.
  * NewRoleRunner - the method that returns a new Runner for a role.
  * This contains the events API code and public methods. Implementation should probably be in the internal package. The events API is used for recieving events from ansible runner. 

* /operator-sdk/pkg/ansible/proxy
  * This is a reverse proxy for the kubernetes API that is used for owner reference injection.
* /operator-sdk/pkg/ansible/proxy/kubeconfig
  * Code needed to generate the kubeconfig for the proxy.


### Commands
We are adding and updating existing commands to accommodate the ansible operator.  Changes to the `cmd` package as well as changes to the generator are needed. 

`operator-sdk new --type ansible --kind <kind> --api-version <group/version>`  This will be a new generation command under the hood. We will:
* Create a new ansible role in the roles directory
* Create a new watchers file. The role and GVK are defaulted based on input to the command. 
* A CRD is generated. This CRD does not have any validations defined. 
* A dockerfile is created using the watchers file and the ansible role with the base image being the ansible operator base image.

The resulting structure should be
```
|- Dockerfile
|
|- roles
\ | - <kind>
|  \ | - generated ansible role
|
| - watchers.yaml
|
| - deploy
\  | - <kind>-CRD.yaml
|  | - rbac.yaml
|  | - operator.yaml
|  | - cr.yaml
```

`operator-sdk generate crd <api-version> <kind> ` This will be used to generate new CRDs based on ansible code. The command helps when a user wants to watch more than one CRD for their operator.
Args:
Required kind - the kind for the object.  
Required api-version - the <group>/<version> for the CRD.
Flags:
Optional: --defaults-file - A path to the defaults file to use to generate a new CRD.yaml. If this is not defined, then an empty CRD is created.

`operator-sdk up ansible` - This should use the known structure and the ansible operator code to run the operator from this location. The command works by running the operator-sdk binary, which includes the ansible operator code, as the operator process. This is slightly different than the current up local command. For debugging a user could then set a flag --verbose which tells ansible to be more verbose.

`operator-sdk build --type=ansible <image-name>` - This builds the operator image. The command calls ansible-galaxy install for the role dependencies and creates the image from the Dockerfile.

`operator-sdk new --type ansible --source`  - This converts an ansible operator to a ansible operator. A user can now change the base ansible operator. This will also allow users to continue using their ansible code while also using the operator-sdk. This is intended for advanced users to allow developers full customization of their operator. The generated operator should now function the same as standard operator-sdk operators.
* Creates the roles directory
* Creates the watchers file.
* Generate the same structure of a normal Operator-SDK
* Generate the main file, that will run the same as the ansible-operator.
* Generate a dockerfile that uses the watchers file and the ansible role as well as the built operator. The command uses all of the default values and does not attempt to reconcile the already created and used dockerfile. The command warns that the old dockerfile is not used.

The resulting structure should be
```
|- Operator-sdk file structure
||
|- roles
\ | - <kind>
|  \ | - generated ansible role
|
| - watchers.yaml
|
| - deploy
\  | - <kind>-CRD.yaml
|  | - rbac.yaml
|  | - operator.yaml
|  | - cr.yaml
```




