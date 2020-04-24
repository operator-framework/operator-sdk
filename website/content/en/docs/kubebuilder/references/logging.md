

Operator SDK-generated operators use the [`logr`][godoc_logr] interface to log. This log interface has several backends such as [`zap`][repo_zapr], which the SDK uses in generated code by default. [`logr.Logger`][godoc_logr_logger] exposes [structured logging][site_struct_logging] methods that help create machine-readable logs and adding a wealth of information to log records.

## Default zap logger

Operator SDK uses a `zap`-based `logr` backend when scaffolding new projects. To assist with configuring and using this logger, the SDK includes several helper functions.

In the simple example below, we add the zap flagset to the operator's command line flags with `BindFlags()`, and then set the controller-runtime logger with `zap.Options{}`.

By default, `zap.Options{}` will return a logger that is ready for production use. It uses a JSON encoder, logs starting at the `info` level. To customize the default behavior, users can use the zap flagset and specify flags on the command line. The zap flagset includes the following flags that can be used to configure the logger:

* `--zap-devel`: Development Mode defaults(encoder=consoleEncoder,logLevel=Debug,stackTraceLevel=Warn)
			  Production Mode defaults(encoder=jsonEncoder,logLevel=Info,stackTraceLevel=Error)
* `--zap-encoder`: Zap log encoding ('json' or 'console')
* `--zap-log-level`: Zap Level to configure the verbosity of logging. Can be one of 'debug', 'info', 'error',
			       or any integer value > 0 which corresponds to custom debug levels of increasing verbosity")
* `--zap-stacktrace-level`: Zap Level at and above which stacktraces are captured (one of 'warn' or 'error')

### A simple example

Operators set the logger for all operator logging in [`cmd/manager/main.go`][code_set_logger]. To illustrate how this works, try out this simple example:

```Go
package main

import (
    "sigs.k8s.io/controller-runtime/pkg/log/zap"  
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var globalLog = logf.Log.WithName("global")
func main() {
	// Add the zap logger flag set to the CLI. The flag set must
	// be added before calling flag.Parse().
	opts := zap.Options{}
	opts.BindFlags(flag.CommandLine)
    flag.Parse()
    
    logger := zap.New(zap.UseFlagOptions(&opts))
    logf.SetLogger(logger)

	scopedLog := logf.Log.WithName("scoped")

	globalLog.Info("Printing at INFO level")
	globalLog.V(1).Info("Printing at DEBUG level")
	scopedLog.Info("Printing at INFO level")
	scopedLog.V(1).Info("Printing at DEBUG level")
}
```

#### Output using the defaults
```console
$ go run main.go
INFO[0000] Running the operator locally in namespace default. 
{"level":"info","ts":1587741740.407766,"logger":"global","msg":"Printing at INFO level"}
{"level":"info","ts":1587741740.407855,"logger":"scoped","msg":"Printing at INFO level"}
```

#### Output overriding the log level to 1 (debug)
```console
$ go run main.go --zap-log-level=debug
INFO[0000] Running the operator locally in namespace default. 
{"level":"info","ts":1587741837.602911,"logger":"global","msg":"Printing at INFO level"}
{"level":"debug","ts":1587741837.602964,"logger":"global","msg":"Printing at DEBUG level"}
{"level":"info","ts":1587741837.6029708,"logger":"scoped","msg":"Printing at INFO level"}
{"level":"debug","ts":1587741837.602973,"logger":"scoped","msg":"Printing at DEBUG level"}
```
## Custom zap logger

In order to use a custom zap logger, [`zap`][controller_runtime_zap] from controller-runtime can be utilized to wrap it in a logr implementation.

Below is an example illustrating the use of [`zap-logfmt`][logfmt_repo] in logging.

### Example

In your `main.go` file, replace the current implementation for logs inside the `main` function:

```Go
...
// Add the zap logger flag set to the CLI. The flag set must
// be added before calling flag.Parse().
	opts := zap.Options{}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	logger := zap.New(zap.UseFlagOptions(&opts))
	logf.SetLogger(logger)
...
```

With:

```Go
configLog := zap.NewProductionEncoderConfig()
configLog.EncodeTime = func(ts time.Time, encoder zapcore.PrimitiveArrayEncoder) {
	encoder.AppendString(ts.UTC().Format(time.RFC3339))
}
logfmtEncoder := zaplogfmt.NewEncoder(configLog)

// Construct a new logr.logger.
log = zapcr.New(zapcr.UseDevMode(true), zapcr.WriteTo(os.Stdout), zapcr.Encoder(logfmtEncoder))

// Set the controller logger to log, which will
// be propagated through the whole operator, generating
// uniform and structured logs.
logf.SetLogger(logger)
```

Ensure that the following additional imports are being used:

```Go
import(
	...
	zaplogfmt "github.com/sykesm/zap-logfmt"
	zapcr "sigs.k8s.io/controller-runtime/pkg/log/zap"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	...
)
```
**NOTE**: For this example, you will need to add the module `"github.com/sykesm/zap-logfmt"` to your project. Run `go get -u github.com/sykesm/zap-logfmt`.

To test, the following print statement can be added in the main function:

`log.Info("Printing at INFO LEVEL")`

#### Output using custom zap logger

```console
$ operator-sdk run --local
INFO[0000] Running the operator locally in namespace default. 
{"level":"info","ts":1587741740.407766,"logger":"global","msg":"Printing at INFO level"}
{"level":"info","ts":1587741740.407855,"logger":"scoped","msg":"Printing at INFO level"}
```

By using `sigs.k8s.io/controller-runtime/pkg/log`, your logger is propagated through `controller-runtime`. Any logs produced by `controller-runtime` code will be through your logger, and therefore have the same formatting and destination.

### Setting flags when running locally

When running locally with `operator-sdk run --local`, you can use the `--operator-flags` flag to pass additional flags to your operator, including the zap flags. For example:

```console
$ operator-sdk run --local --operator-flags="--zap-log-level=debug --zap-encoder=console"`
```

### Setting flags when deploying to a cluster

When deploying your operator to a cluster you can set additional flags using an `args` array in your operator's `container` spec. For example:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: memcached-operator
spec:
  replicas: 1
  selector:
    matchLabels:
      name: memcached-operator
  template:
    metadata:
      labels:
        name: memcached-operator
    spec:
      serviceAccountName: memcached-operator
      containers:
        - name: memcached-operator
          # Replace this with the built image name
          image: REPLACE_IMAGE
          command:
            - memcached-operator
          args:
            - "--zap-log-level=debug"
            - "--zap-encoder=console"
          imagePullPolicy: Always
          env:
            - name: WATCH_NAMESPACE
              valueFrom:
                fieldRef:
                  fieldPath: metadata.namespace
            - name: POD_NAME
              valueFrom:
                fieldRef:
                  fieldPath: metadata.name
            - name: OPERATOR_NAME
              value: "memcached-operator"
```

## Creating a structured log statement

There are two ways to create structured logs with `logr`. You can create new loggers using `log.WithValues(keyValues)` that include `keyValues`, a list of key-value pair `interface{}`'s, in each log record. Alternatively you can include `keyValues` directly in a log statement, as all `logr` log statements take some message and `keyValues`. The signature of `logr.Error()` has an `error`-type parameter, which can be `nil`.

An example from [`memcached_controller.go`][code_memcached_controller]:

```Go
package memcached

import (
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// Set a global logger for the memcached package. Each log record produced
// by this logger will have an identifier containing "controller_memcached".
// These names are hierarchical; the name attached to memcached log statements
// will be "operator-sdk.controller_memcached" because SDKLog has name
// "operator-sdk".
var log = logf.Log.WithName("controller_memcached")

func (r *ReconcileMemcached) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	// Create a logger for Reconcile() that includes "Request.Namespace"
	// and "Request.Name" in each log record from this log statement.
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Memcached.")

	memcached := &cachev1alpha1.Memcached{}
	err := r.client.Get(context.TODO(), request.NamespacedName, memcached)
	if err != nil {
		if errors.IsNotFound(err) {
			reqLogger.Info("Memcached resource not found. Ignoring since object must be deleted.")
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	found := &appsv1.Deployment{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: memcached.Name, Namespace: memcached.Namespace}, found)
	if err != nil {
		if errors.IsNotFound(err) {
			dep := r.deploymentForMemcached(memcached)
			// Include "Deployment.Namespace" and "Deployment.Name" in records
			// produced by this particular log statement. "Request.Namespace" and
			// "Request.Name" will also be included from reqLogger.
			reqLogger.Info("Creating a new Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
			err = r.client.Create(context.TODO(), dep)
			if err != nil {
				// Include the error in records produced by this log statement.
				reqLogger.Error(err, "Failed to create new Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
				return reconcile.Result{}, err
			}
		}
		return reconcile.Result{}, err
	}

	...
}
```

Log records will look like the following (from `reqLogger.Error()` above):

```
2018-11-08T00:00:25.700Z	ERROR	operator-sdk.controller_memcached pkg/controller/memcached/memcached_controller.go:118	Failed to create new Deployment	{"Request.Namespace", "memcached", "Request.Name", "memcached-operator", "Deployment.Namespace", "memcached", "Deployment.Name", "memcached-operator"}
```

## Non-default logging

If you do not want to use `logr` as your logging tool, you can remove `logr`-specific statements without issue from your operator's code, including the `logr` [setup code][code_set_logger] in `cmd/manager/main.go`, and add your own. Note that removing `logr` setup code will prevent `controller-runtime` from logging.


[godoc_logr]:https://godoc.org/github.com/go-logr/logr
[repo_zapr]:https://godoc.org/github.com/go-logr/zapr
[godoc_logr_logger]:https://godoc.org/github.com/go-logr/logr#Logger
[site_struct_logging]:https://www.client9.com/structured-logging-in-golang/
[code_memcached_controller]:https://github.com/operator-framework/operator-sdk/blob/master/example/memcached-operator/memcached_controller.go.tmpl
[code_set_logger]:https://github.com/operator-framework/operator-sdk/blob/4d66be409a69d169aaa29d470242a1defbaf08bb/internal/pkg/scaffold/cmd.go#L92-L96
[zap_sampling]:https://github.com/uber-go/zap/blob/master/FAQ.md#why-sample-application-logs
[logfmt_repo]:https://github.com/jsternberg/zap-logfmt
[controller_runtime_zap]:https://github.com/kubernetes-sigs/controller-runtime/tree/master/pkg/log/zap
