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
* /operator-sdk/pkg/ansible/controller
  * Will contain the ansible operator controller.
  * Will contain a exposed reconciler. But the default `Add` method will use this reconciler.
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
* /operator-sdk/pkg/ansible/events
  * Package for event handlers from ansible runner events.
  * Default has only the event logger.


### Commands
We are adding and updating existing commands to accommodate the ansible operator.  Changes to the `cmd` package as well as changes to the generator are needed.

`operator-sdk new <project-name> --type ansible --kind <kind> --api-version <group/version>`  This will be a new generation command under the hood. We will:
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

`operator-sdk up local` - This should use the known structure and the ansible operator code to run the operator from this location. This will need to be changed to determine if it is an ansible operator or a golang operator. The command works by running the operator-sdk binary, which includes the ansible operator code, as the operator process. This is slightly different than the current up local command.

`operator-sdk build <image-name>` - This builds the operator image. This will need to be changed to determine if ansbile operator or golang operator.


