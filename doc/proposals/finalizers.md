# Finalizer Helper proposal

## Background:
In kubernetes, object reclamation on delete is controlled by finalizers. Giving a resource a finalizer will 
stop the deletion of the resource. The resource will not be reclaimed until the finalizer is removed by a controller 
(finalizer controller). Finalizers cannot be added to resources on creation and require a seperate call to the API to
add the finalizer for your finalizer controller to the resource. Best practice is to use a MutatingAdmissionWebhook to
add the finalizer to the resource, you can also add it with an update or patch.

## Goals:
* Abstract the process of adding and managing finalizers on resources
* Ease the building of finalizer controller
* Support abstracting the creation of the MutatingAdmissionWebhook for adding the finalizer

## Details:

## Implementation:

## Q&A:

## Future Plans:
