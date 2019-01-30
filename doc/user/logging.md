# Logging in operators

Operator SDK-generated operators use the [`logr`][godoc_logr] interface to log. This log interface has several backends such as [`zap`][repo_zapr], which the SDK uses in generated code by default. [`logr.Logger`][godoc_logr_logger] exposes [structured logging][site_struct_logging] methods that help create machine-readable logs and adding a wealth of information to log records.

## Setting the logger

Operators set the logger for all operator logging in [`cmd/manager/main.go`][code_set_logger]:

```Go
import (
	"github.com/operator-framework/operator-sdk/pkg/log/zap"
	"github.com/spf13/pflag"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

func main() {
  pflag.CommandLine.AddFlagSet(zap.FlagSet())
  pflag.Parse()
  logf.SetLogger(zap.Logger())
  log := logf.Log.WithName("cmd")

  ...

  log.Info("Starting the Cmd.")

  ...
}
```

By using `controller-runtime/pkg/runtime/log`, your logger is propagated through `controller-runtime`. Any logs produced by `controller-runtime` code will be through your logger, and therefore have the same formatting and destination.

### Default zap logger

Operator SDK uses a `zap`-based `logr` backend when scaffolding new projects. To assist with configuring and using this logger, the SDK includes several helper functions.

In the above example, we add the zap flagset to the operator's command line flags with `zap.FlagSet()`, and then set the controller-runtime logger with `zap.Logger()`.

By default, `zap.Logger()` will return a logger that is ready for production use. It uses a JSON encoder, logs starting at the `info` level, and has [sampling][zap_sampling] enabled. To customize the default behavior, users can use the zap flagset and specify flags on the command line. The zap flagset includes the following flags that can be used to configure the logger:
* `--zap-devel` - Enables the zap development config (changes defaults to console encoder, debug log level, and disables sampling) (default: `false`)
* `--zap-encoder` string - Sets the zap log encoding (`json` or `console`)
* `--zap-level` string - Sets the zap log level (`debug`, `info`, or `error`)
* `--zap-sample` - Enables zap's sampling mode

**NOTE:** Although the `logr` interface supports multiple debug levels (e.g. `log.V(1).Info()`, `log.V(2).Info()`, etc.), zap supports only a single debug level with `log.V(1).Info()`. Log statements with higher debug levels will not be printed with the zap's `logr` backend.

## Creating a structured log statement

There are two ways to create structured logs with `logr`. You can create new loggers using `log.WithValues(keyValues)` that include `keyValues`, a list of key-value pair `interface{}`'s, in each log record. Alternatively you can include `keyValues` directly in a log statement, as all `logr` log statements take some message and `keyValues`. The signature of `logr.Error()` has an `error`-type parameter, which can be `nil`.

An example from [`memcached_controller.go`][code_memcached_controller]:

```Go
package memcached

import (
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
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
[code_memcached_controller]:../../example/memcached-operator/memcached_controller.go.tmpl
[code_set_logger]:https://github.com/operator-framework/operator-sdk/blob/ecd02000616f11303f1adecd3d4ceb4a8561a9ec/pkg/scaffold/cmd.go#L90-L94
[zap_sampling]:https://github.com/uber-go/zap/blob/master/FAQ.md#why-sample-application-logs
