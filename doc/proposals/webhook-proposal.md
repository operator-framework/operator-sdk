## Webhook Support

Implementation Owner: theishshah

Status: Draft

[Background](#Background)

[Goals](#Goals)

[Design overview](#Design_overview)

[User facing usage](#User_facing_usage)

[Observations and open questions](#Observations_and_open_questions)

### Background

The upcoming stable version of controller runtime (`v0.2.0`) has support for running a webhook server and having webhooks to mutate or validate pods. The mutation webhook can change various attributes of a pod, and the validation webhook can read pod attributes and allow/deny a pod to run based on this information. 

### Goals

The goal of this proposal is to add the abilty for users to quickly add a webhook server and only implement the validation/mutation logic for their webhooks. Ideally this will support per-API webhooks, however, the proposal is not there yet.

### Design overview

All of the necesary files and changes for the generated operator occur in the `cmd/manager/` directory.

`osdk-webhook.go` will be responsible for containing the logic to start up a webhook server, including a `webhookServer` struct to contain the necesary information to start the server.

The webhook initilization logic function will be a function which returns successfully _without_ starting the server if the user does not enable webhook support via the command line (planned command is `$ operator-sdk generate webhooks`, more detail is outlined in the next section). If a user chooses to generate webhooks, the osdk-webhook.go file will be overwritten with a new file to contain a server init function with full functionality. Additionally it will generate the server information struct populated with the deafult values. Controller runtime defaults for these values are `Port: 9876` and `CertDir: "/tmp/cert"`.

```go
type webhookServer struct {
    Port int
    CertDir string
} 
```

The initialization function will be called in the main.go file, regardless of whether or not a user opts in, however no actions are taken unless the user chooses to generate webhooks. The main function is unmodified by calling for `generate webhook` as the function call to initialize the webhook will exist when the operator is initially created. Until a user chooses to generate webhooks the webhook initiating function will simply return error free without performaning any action. 

```go
func WebhookInit() error {
    log.Info("Starting webhook server")
    hookServer := &webhook.Server{
        Port: hookServerCfg.Port,
        CertDir: hookServerCfg.CertDir,
    }
    
    if err := mgr.Add(hookServer); err != nil {
        log.Error(err, "Unable to register webhook server")
        return err
    }

    log.Info("Registering webhooks to the webhook server")
    
    hookServer.Register("/mutate-pods", &webhook.Admission{Handler: &podAnnotator{}})
    hookServer.Register("/validate-pods", &webhook.Admission{Handler: &podValidator{}})
    
    return nil
}
```


In addition to the above file being used to start up the server, additional  files (`mutationswebhook.go` and `validationwebhooks.go`) will generated when a user chooses to include webhooks. These are the files which have the necessary boilerplate for a user to implement the `Handle` function in order to run custom logic on validation and admission of pods.

```go
type podInteraction struct {
	client  client.Client
	decoder *admission.Decoder
}

func (i *podInteraction) Handle(ctx context.Context, req admission.Request) admission.Response {
	pod := &corev1.Pod{}

	err := i.decoder.Decode(req, pod)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

    //Insert mutation/validation logic here

    //BRANCH
    //Block A contains the boiler plate logic for patching a mutation
    //Block B contains the boilder plate logic for 

    //BLOCK A
	marshaledPod, err := json.Marshal(pod)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

    return admission.PatchResponseFromRaw(req.Object.Raw, marshaledPod)
    
    //BLOCK B
    if FAILURE_CONDITION {
		return admission.Denied(fmt.Sprintf("reason for failure"))
	}

    return admission.Allowed("")
    
    //END BRANCH
}

// podInteraction implements inject.Client.
// A client will be automatically injected.

// InjectClient injects the client.
func (i *podInteraction) InjectClient(c client.Client) error {
	i.client = c
	return nil
}

// podInteraction implements admission.DecoderInjector.
// A decoder will be automatically injected.

// InjectDecoder injects the decoder.
func (i *podInteraction) InjectDecoder(d *admission.Decoder) error {
	i.decoder = d
	return nil
}

```


### User facing usage (if needed)

My suggested method for interacting with this feature is to have a command in the osdk which can be run after generating the base operator. The new command `$ operator-sdk generate webhook` will write the files `pkg/osdk-webhook/osdk-webhook.go`, `pkg/osdk-webhook/mutationwebhook.go`, and `pkg/osdk-webhook/validationwebhook.go`

These flags are only used as part of the generate webhook command:

* `--port` int - Port to start webhook server on
* `--cert-dir` string - Certificate directory

