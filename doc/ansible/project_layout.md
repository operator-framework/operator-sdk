# Project Scaffolding Layout

After creating a new operator project using 
`operator-sdk new --type ansible`, the project directory has numerous generated folders and files. The following table describes a basic rundown of each generated file/directory.


| File/Folders   | Purpose                           |
| :---           | :--- |
| deploy | Contains a generic set of kubernetes manifests for deploying this operator on a kubernetes cluster. |
| roles/<kind> | Contains an Ansible Role initialized using [Ansible Galaxy](https://docs.ansible.com/ansible/latest/reference_appendices/galaxy.html) |
| build | Contains scripts that the operator-sdk uses for build and initialization. |
| watches.yaml | Contains Group, Version, Kind, and Ansible invocation method. |
