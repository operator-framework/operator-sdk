## Webhook Support

Implementation Owner: theishshah
Status: Draft

[Background](#Background)
[Goals](#Goals)
[Design overview](#Design_overview)
[User facing usage](#User_facing_usage)
[Observations and open questions](#Observations_and_open_questions)

### Background

The upcoming stable version of controller runtime has support for running a webhook server and having webhooks to mutate or validate pods. The mutation webhook can change various attributes of a pod, and the validation webhook can read pod attributes and allow/deny a pod to run based on this information. 

### Goals

The goal of this proposal is to add the abilty for users to quickly add a webhook server and only implement the validation/mutation logic for their webhooks.

### Design overview

All of the necesary files and changes for the generated operator occur in the cmd/manager/ directory. The code to create and register the server is in the main.go file and can be completely generated with no additional input needed from the user. In addition the osdk will provide 2 files, 1 each for validation and mutation webhooks. These will have a template Handle function in which the user can define the desired behavior for their pod validation/mutation logic. 

### User facing usage (if needed)

My suggested method for interacting with this feature is to have a command in the osdk which can be run after generating the base operator. The new command `generate webhook` will write the files cmd/manager/main.go, cmd/manager/mutationwebhook.go, and cmd/manager/validationwebhook.go

### Observations and open questions

< Any open questions that need solving or implementation details should go here. These can be removed at the end of the proposal if they are resolved. >
