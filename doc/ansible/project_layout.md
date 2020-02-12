# Project Scaffolding Layout for Operator SDK Ansible-based Operators

After creating a new operator project using
`operator-sdk new --type ansible`, the project directory has numerous generated folders and files. The following table describes a basic rundown of each generated file/directory.


| File/Folders   | Purpose                           |
| :---           | :--- |
| deploy/ | Contains a generic set of Kubernetes manifests for deploying this operator on a Kubernetes cluster. |
| roles/\<kind> | Contains an Ansible Role initialized using [Ansible Galaxy](https://docs.ansible.com/ansible/latest/galaxy/user_guide.html) |
| build/ | Contains scripts that the `operator-sdk` uses for build and initialization. |
| watches.yaml | Contains Group, Version, Kind, and the Ansible invocation method. |
| molecule/ | Contains [Molecule](https://molecule.readthedocs.io/) scenarios for end-to-end testing of your role and operator |
