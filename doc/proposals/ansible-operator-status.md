# Ansible Operator Status Proposal


## Current status
The operator can currently surface basic information from the Ansible code in a generic fashion - it can set the conditions and it can surface a failure message if the run failed.

## Problem

There is currently no way to intentionally surface information to the status object from Ansible. If users want to surface custom information in their Custom Resource’s status field, they would need to change the Go code for the operator.

There are two main approaches we see for users wanting to manage the status:

1. The user wants to change the status of the Custom Resource over the course of an Ansible reconciliation run, but still wants to make use of the operator's builtin status management utilities to set certain conditions and failure messages
1. The user wants to manage the entire status manually from Ansible, and the Go-side of Ansible Operator does nothing with statuses


## Proposal

Add a new field to the watches.yaml entry, which tells the operator whether or not it is allowed to manage status.

Add a new Ansible module, named `k8s_status` that is usable when running inside Ansible Operator.

The `k8s_status` module would take the apiVersion, kind, name, namespace as well as a status blob and list of conditions. It would then set the status on the specified resource using that blob. The conditions would be validated to conform to the [Kubernetes API conventions](https://github.com/kubernetes/community/blob/master/contributors/devel/api-conventions.md#typical-status-properties). It would then take this status blob and call to update the status subresource for the specified resource (which should be the CR that is being reconciled).\
_**Note - likely this module would inherit from k8s_common, and include the same general authentication/etc options as the other k8s modules*_

The Go-side of the Ansible Operator would need two changes:
1. It needs to not attempt to manage the status when the watches.yaml tells it not to
1. It needs to `GET` the current state of the object before updating the status after an Ansible reconciliation run, so that it does not attempt to write a stale version of the object.



## Complications / Further Discussion

Not all clusters and Custom Resource Definitions have the status subresource enabled. In the case where the status subresource is not enabled for a CRD, we can do one of two things:

1. Error out - if the user is attempting to manage the status of an object that doesn’t have the subresource enabled, we call it a misconfiguration and fail without attempting to update the status.
2. If a resource doesn’t have the status subresource enabled, the status field can still be updated. Instead of updating the status subresource, you would `GET` the resource in its current state, update the status field manually, and `PUT` the whole object back.
