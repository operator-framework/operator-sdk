# Ansible Operator Status Proposal


## Current status
The operator can currently surface basic information from the Ansible code in a generic fashion - it can set the conditions and it can surface a failure message if the run failed.

## Problem

There is currently no way to intentionally surface information to the status object from Ansible. If users want to surface custom information in their Custom Resource’s status field, they would need to change the Go code for the operator.

There are two main approaches we see for users wanting to manage the status:

1. The user wants to pass a message to the operator to set on the status field, but otherwise wants the operator to manage the status as normal (ie, setting the conditions and failure messages)
2. The user wants to manage the entire status manually from Ansible, and the Go-side of Ansible Operator does nothing with statuses


## Proposal

Add a new field to the watches.yaml entry, which tells the operator whether or not it is allowed to manage status.

Add a new Ansible module, named `k8s_status` that is usable when running inside Ansible Operator.

The `k8s_status` module would take the apiVersion, kind, name, namespace as well as a status blob. It would then set the status on the specified resource using that blob.\
_**Note - likely this module would inherit from k8s_common, and include the same general authentication/etc options as the other k8s modules*_

In the Ansible Operator proxy, any update calls to the status subresource will be intercepted:

- If the operator is managing status, it will prevent this call from reaching the real k8s API (without failing the request). Then, when a task event for the `k8s_status` module reaches the event handler, the operator will read the status update and perform it, in addition to the normal status condition management the operator is responsible for.
- If the operator is not managing status, it will allow the call to go forward as normal.


## Complications / Further Discussion

Not all clusters and Custom Resource Definitions have the status subresource enabled. In the case where the status subresource is not enabled for a CRD, we can do one of two things:

1. Error out - if the user is attempting to manage the status of an object that doesn’t have the subresource enabled, we call it a misconfiguration and fail without attempting to update the status.
2. If a resource doesn’t have the status subresource enabled, the status field can still be updated. Instead of updating the status subresource, you would `GET` the resource in its current state, update the status field manually, and `PUT` the whole object back. 

    The major downside to this approach is that it would be much more difficult to intercept the status updates from the proxy, as there is no difference in appearance from a status update and a normal resource update.
