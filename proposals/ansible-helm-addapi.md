---
title: Ansible/Helm add api command proposal for Operator SDK
authors:
  - "@bharathi-tenneti"
reviewers:
  - "@fabianvf"
  - "@cmacedo"
approvers:
  - "@fabianvf"
  - "@jlanford"
  - "@dmesser"
creation-date: 2020-03-10
last-updated: 2020-03-15
status: implementable
---


# Ansible/Helm add API command proposal for Operator SDK


## Release Signoff Checklist

- \[x\] Enhancement is `implementable`
- \[x\] Design details are appropriately documented from clear requirements
- \[ \] Test plan is defined
- \[ \] Accceptance criteria
- \[ \] User-facing documentation is created



## Summary

The proposal is to enable Ansible/Helm operator developers to create additional APIs, through SDK CLI.

## Motivation

As of today, SDK CLI can be used only to add additonal APIs for Go based Operators.
Ansible/Helm operator developers are not able to create additonal APIs via CLI, once the original project scaffolds. Today, developers have to manually add necessary files to the project scaffold.

## Goals

* Ansible/Helm operator developer can use existing SDK CLI commands to create additonal APIs as needed.
* Ansible/Helm operator developer should be able to use flags necessary for Ansible/Helm as used in `operator-sdk new`CLI. for adding additional APIs as well.
* Ansible/Helm operator developer can find supported documentation for the same.

## Non-Goals

* Updating Molecule tests for additional APIs created for Anible based operators.<**TBD**>
* Generating/Updating Playbook for additional APIs for Ansible based operators. <**TBD**>

## Proposal

### User Stories

#### Story 1 - Ansible operator additional API
As an  Ansible operator developer, I would like to scaffold additional API, once the original Ansible operator project has been created. Goal is to use  following command, to create additonal APIs.
    `operator-sdk add api --kind <kind> --api-version <group/version> [flags]`

##### Acceptance Criteria

* Ansible operator developer should be able to scaffold all resources needed for the additional API with following command.
    `operator-sdk add api --kind <kind> --api-version <group/version> [flags]`
* Flags options available for `operator-sdk new`, should also be made available for `operator-sdk add api`, as shown below.
```
  --api-version string - CRD APIVersion in the format $GROUP_NAME/$VERSION (e.g app.example.com/v1alpha1)
  --kind string - CRD Kind. (e.g AppService)
  --generate-playbook - Generate a playbook skeleton. (Only used for --type ansible) [**TBD**]
```
* Documentation for [SDK CLI reference][sdkclidoc] is updated with steps to add additonal APIs for Ansible based operator.
* Documentation is updated for [operator-sdk add api][addapidoc] for ansible.


#### Story 2 - Helm operator additional API

As Helm operator developer, I would like to scaffold additional API, once the original Helm operator project has been created, using following command.
    `operator-sdk add api --kind <kind> --api-version <group/version> [flags]`

##### Acceptance Criteria
* Helm operator developer should be able to scaffold all resources needed for the additional API with any of below commands,
      `operator-sdk add api --kind <kind> --api-version <group/version> [flags]`
* Flags options available for `operator-sdk new`, should also be made available for `operator-sdk add api`
```
  --api-version string - CRD APIVersion in the format $GROUP_NAME/$VERSION (e.g app.example.com/v1alpha1)
  --kind string - CRD Kind. (e.g AppService)
  --helm-chart string - Initialize helm operator with existing helm chart (<URL>, <repo>/<name>, or local path)
  --helm-chart-repo string - Chart repository URL for the requested helm chart
  --helm-chart-version string - Specific version of the helm chart (default is latest version)
  ```
* Documentation for [SDK CLI reference][sdkclidoc] is updated with steps to add additonal APIs for Helm based operator.
* Documentation is updated for [operator-sdk add api][addapidoc] for helm.



### Implementation Details/Notes/Constraints

* The `operator-sdk new memcached-operator --api-version=cache.example.com/v1alpha1 --kind=Memcached --type=ansible` command scaffolds new ansible based operator for the user with given API.Please refer below logic being used to determine the type of operator:
```go
	case projutil.OperatorTypeAnsible:
		if err := doAnsibleScaffold(); err != nil {
			return err
		}
	case projutil.OperatorTypeHelm:
		if err := doHelmScaffold(); err != nil {
			return err
		}
```
and subsequently [`func doAnsibleScaffold()`][doansible] or [`func doHelmScaffold()`][dohelm] is being called to perform the scaffold.

 Currently, `operator-sdk add api` only allows Go-based operators to create further APIs, after the original project is scaffolded. By posing restriction as shown [here][onlygorestriction].
```go
// Only Go projects can add apis.
	if err := projutil.CheckGoProjectCmd(cmd); err != nil {
		return err
	}
```
* This proposal is to enhance [`func apiRun(cmd *cobra.Command, args []string)`][addapifunc] to add APIs for Ansible/Helm operators, by re-using pre-existing functions as shown above to check for `--type`, and perform necessary scaffolds for the new resource.

* To this extent, PoCs have been done for both [Ansible][ansiblepoc] and [Helm][helmpoc] by manually adding necessary files in the scaffold. Please see below for project layout.
  **NOTE**: To test/check the POCs locally used the makefile targets `make install` and `make uninstall`.

  * Ansible roles scaffold after adding APIs:
  ```
  ── roles
  │   ├── memcached
  │   │   ├── README.md
  │   │   ├── defaults
  │   │   │   └── main.yml
  │   │   ├── files
  │   │   ├── handlers
  │   │   │   └── main.yml
  │   │   ├── meta
  │   │   │   └── main.yml
  │   │   ├── tasks
  │   │   │   └── main.yml
  │   │   ├── templates
  │   │   └── vars
  │   │       └── main.yml
  │   ├── myapp
  │   │   ├── README.md
  │   │   ├── defaults
  │   │   │   └── main.yml
  │   │   ├── files
  │   │   ├── handlers
  │   │   │   └── main.yml
  │   │   ├── meta
  │   │   │   └── main.yml
  │   │   ├── tasks
  │   │   │   └── main.yml
  │   │   ├── templates
  │   │   └── vars
  │   │       └── main.yml
  ```

  * Helm charts are scaffolded as shown below for new APIs:
  ```bash
  ├── helm-charts
  │   ├── memcached
  │   │   ├── Chart.yaml
  │   │   ├── README.md
  │   │   ├── templates
  │   │   │   ├── NOTES.txt
  │   │   │   ├── _helpers.tpl
  │   │   │   ├── pdb.yaml
  │   │   │   ├── statefulset.yaml
  │   │   │   └── svc.yaml
  │   │   └── values.yaml
  │   ├── mongodb
  │   │   ├── Chart.yaml
  │   │   ├── OWNERS
  │   │   ├── README.md
  │   │   ├── files
  │   │   │   └── docker-entrypoint-initdb.d
  │   │   │       └── README.md
  │   │   ├── templates
  │   │   │   ├── NOTES.txt
  │   │   │   ├── _helpers.tpl
  │   │   │   ├── configmap.yaml
  │   │   │   ├── deployment-standalone.yaml
  │   │   │   ├── ingress.yaml
  │   │   │   ├── initialization-configmap.yaml
  │   │   │   ├── poddisruptionbudget-arbiter-rs.yaml
  │   │   │   ├── poddisruptionbudget-secondary-rs.yaml
  │   │   │   ├── prometheus-alerting-rule.yaml
  │   │   │   ├── prometheus-service-monitor.yaml
  │   │   │   ├── pvc-standalone.yaml
  │   │   │   ├── secrets.yaml
  │   │   │   ├── statefulset-arbiter-rs.yaml
  │   │   │   ├── statefulset-primary-rs.yaml
  │   │   │   ├── statefulset-secondary-rs.yaml
  │   │   │   ├── svc-headless-rs.yaml
  │   │   │   ├── svc-primary-rs.yaml
  │   │   │   └── svc-standalone.yaml
  │   │   ├── values-production.yaml
  │   │   ├── values.schema.json
  │   │   └── values.yaml
  │   └── nginx
  │       ├── Chart.yaml
  │       ├── charts
  │       ├── templates
  │       │   ├── NOTES.txt
  │       │   ├── _helpers.tpl
  │       │   ├── deployment.yaml
  │       │   ├── ingress.yaml
  │       │   ├── service.yaml
  │       │   ├── serviceaccount.yaml
  │       │   └── tests
  │       │       └── test-connection.yaml
  │       └── values.yaml
  ```
  * Along with above changes,`/deploy/role.yaml` will be updated to reflect new apiGroups.
  ``` yaml
  - apiGroups:
    - cache.example.com
    resources:
    - '*'
    verbs:
    - create
    - delete
    - get
    - list
    - patch
    - update
    - watch
  - apiGroups:
    - app.example.com
    resources:
    - '*'
    verbs:
    - create
    - delete
    - get
    - list
    - patch
    - update
    - watch
  ```
* CRD and CR files will be generated at `/deploy/crds`, as shown below.
```
── deploy
│   ├── crds
│   │   ├── cache.example.com_memcacheds_crd.yaml
│   │   ├── cache.example.com_v1alpha1_memcached_cr.yaml
│   │   ├── charts.helm.k8s.io_mongodbs_crd.yaml
│   │   ├── charts.helm.k8s.io_v1alpha1_mongodb_cr.yaml
│   │   ├── example.com_nginxes_crd.yaml
│   │   └── example.com_v1alpha1_nginx_cr.yaml
```
* watches.yaml at `/watches.yaml` gets updated with new API resource as shown below.
```yaml
- version: v1alpha1
  group: cache.example.com
  kind: Memcached
  role: /opt/ansible/roles/memcached

- version: v1alpha1
  group: cache.example.com
  kind: Mykind
  role: /opt/ansible/roles/mykind
```
[addapidoc]: https://sdk.operatorframework.io/docs/cli/operator-sdk_add_api/
[sdkclidoc]: https://sdk.operatorframework.io/docs/cli/
[onlygorestriction]:https://github.com/operator-framework/operator-sdk/blob/master/cmd/operator-sdk/add/api.go#L95
[doansible]:https://github.com/operator-framework/operator-sdk/blob/master/cmd/operator-sdk/new/cmd.go#L228
[dohelm]:https://github.com/operator-framework/operator-sdk/blob/master/cmd/operator-sdk/new/cmd.go#L320
[addapifunc]:https://github.com/operator-framework/operator-sdk/blob/master/cmd/operator-sdk/add/api.go#L91
[ansiblepoc]:https://github.com/bharathi-tenneti/memcached-ansible-demo
[helmpoc]:https://github.com/bharathi-tenneti/helm-operator-demo
