---
title: "Resource pruning"
linkTitle: "Resource Pruning"
weight: 3
description: Recommendations for pruning resources
---

## Overview

Operators can create [Jobs][jobs] or Pods as part of their normal operation, and when those Jobs or Pods
complete, they can remain on the Kubernetes cluster if not specifically removed. These resources
can consume valuable cluster resources like disk storage (e.g. etcd). These resources
are not tied to a Custom Resource using an `ownerReference`.

Operator authors have traditionally had two pruning options:

 * leave the resource cleanup to a system admin to perform
 * implement some form of pruning within their operator solution

For our purposes when we say, *prune*, we mean to remove a resource (e.g. kubectl delete) from
a Kubernetes cluster for a given namespace.

This documentation describes the pattern and library useful for implementing a solution within an operator.

## operator-lib prune library

A simple pruning implementation can be found in the [operator-lib prune package][operator-lib-prune]. This 
package is written in Go and is meant to be used within Go-based operators. This package was 
developed to include common pruning strategies as found in common operators. The package also allow 
for customization of hooks and strategies.

### Pruning Configuration

Users can configure the pruning library by creating code similar to this example:
```golang
cfg = Config{
	Log:           logf.Log.WithName("prune"),
	DryRun:        false,
	Clientset:     client,
	LabelSelector: "app=churro",
	Resources: []schema.GroupVersionKind{
		{Group: "", Version: "", Kind: PodKind},
	},
	Namespaces: []string{"default"},
	Strategy: StrategyConfig{
		Mode:            MaxCountStrategy,
		MaxCountSetting: 1,
	},
	PreDeleteHook: myhook,
}
```


| Config Field | Description
| ------------ | -----------
| Log          | a logr.Logger. It is optional if a logger is provided through the context to the Execute method, which is the case with the context of the Reconcile function of operator-sdk and controller-runtime
| DryRun       | a boolean determines whether to actually remove resources; `true` means to execute but not to remove resources
| Clientset    | a client-go Kubernetes ClientSet that will be used for Kube API calls by the library
| LabelSelector| Kubernetes label selector expression used to find resources to prune
| Resources    | Kube resource Kinds, currently PodKind and JobKind are supported by the library
| Namespaces   | a list of Kube Namespaces to search for resources
| Strategy     | specifies the pruning strategy to execute
| Strategy.Mode| currently MaxCountStrategy, MaxAgeStrategy, or CustomStrategy are supported
| Strategy.MaxCountSetting| integer value for *maxcount* strategy, specifies how many resources should remain after pruning executes
| Strategy.MaxAgeSetting| golang time.Duration string value (e.g. 48h), specifies age of resources to prune
| Strategy.CustomSettings| a golang map of values that can be passed into a Custom strategy function
| PreDeleteHook| optionally specifies a golang function to call before pruning a resource
| CustomStrategy | optionally specifies a golang function that implements a custom pruning strategy


### Pruning Execution

Users can invoke the pruning by running the *Execute* function on the pruning configuration
as follows:
```golang
err := cfg.Execute(ctx)
```

Users might want to implement pruning execution by means of a cron package or simply call the prune
library based on some other triggering event.

If a logger has been configured in the Config structure it takes precedence on the one provided through ctx.
Adding a logger.Logger to the context can be done with [logr.NewContext][logr-newcontext].

## Pruning Strategies

### maxcount Strategy

A strategy of leaving a finite set of resources is implemented called *maxcount*. This strategy
seeks to leave a specific number of resources, sorted by latest, on your cluster. For example, if
you have 10 resources that would be pruned, and you specified a *maxcount* value of 4, then 6 
resources would be pruned (removed) from your cluster starting with the oldest resources.

### maxage Strategy

A strategy of removing resources greater than a specific time is called *maxage*.  This strategy
seeks to remove resources older than a specified *maxage* duration.  For example, a library
user might specify a value of *48h* to indicate that any resource older than 48 hours would be
pruned.  Durations are specified using golang's [time.Duration formatting] (e.g. 48h).

## Pruning Customization

### preDelete Hook

Users can provide a *preDelete* hook when using the [operator-lib prune package][operator-lib-prune].  
This hook function will be called by the library before removing a resource.  This provides a means to examine
the resource logs for example, extracting any valued content, before the resource is removed
from the cluster.

Here is an example of a *preDelete* hook:
```golang
func myhook(ctx context.Context, cfg Config, res ResourceInfo) error {
        log := prune.Logger(ctx, cfg)
        log.V(4).Info("pre-deletion", "GVK", res.GVK, "namespace", res.Namespace, "name", res.Name)
       	if res.GVK.Kind == PodKind {
                req := cfg.Clientset.CoreV1().Pods(res.Namespace).GetLogs(res.Name, &v1.PodLogOptions{})
                podLogs, err := req.Stream(context.Background())
                if err != nil {
                        return err
                }
                defer podLogs.Close()

                buf := new(bytes.Buffer)
                _, err = io.Copy(buf, podLogs)
                if err != nil {
                        return err
                }

                log.V(4).Info("pod log before removing is", "log", buf.String())
        }
	return nil
}
```

*Note* if your custom hook returns an error, then the resource will not be removed by the
prune library.

### Custom Strategy

Library users can also write their own custom pruning strategy function to support advanced
cases. Custom strategy functions are passed in the prune configuration and a list of resources selected by
the library.  The custom strategy builds up a list of resources to be removed, returning the list to the prune library which
performs the actual resource removal. Here is an example custom strategy:
```golang
func myStrategy(ctx context.Context, cfg Config, resources []ResourceInfo) (resourcesToRemove []ResourceInfo, err error) {
        log := Logger(ctx, cfg)
        log.V(4).Info("myStrategy called", "resources", resources, "config", cfg)
	if len(resources) != 3 {
		return resourcesToRemove, fmt.Errorf("count of resources did not equal our expectation")
	}
	return resourcesToRemove, nil
}
```


To have your custom strategy invoked, you will specify your function within the prune configuration
as follows:
```golang
cfg.Strategy.Mode = CustomStrategy
cfg.Strategy.CustomSettings = make(map[string]interface{})
cfg.CustomStrategy = myStrategy
```

Notice that you can optionally pass in settings to your custom function as a map using the `cfg.Strategy.CustomSettings` field.


[operator-lib]: https://github.com/operator-framework/operator-lib
[operator-lib-prune]: https://github.com/operator-framework/operator-lib/tree/main/prune
[jobs]: https://kubernetes.io/docs/concepts/workloads/controllers/job/
[time.Duration formatting]: https://pkg.go.dev/time#Duration
[logr-newcontext]: https://pkg.go.dev/github.com/go-logr/logr#NewContext
