---
title: "Designing Lean Operators"
linkTitle: "Designing Lean Operators"
weight: 5
description: This guide describes good practices concepts to designing lean Operators.
---

## Overview

One of the pitfalls that many operators are failing into is that they watch resources with high cardinality like secrets possibly in all namespaces. This has a massive impact on the memory used by the controller on big clusters. Such resources can be filtered by label or fields. The original doc design for `Filter cache ListWatch using selectors` can be accessed from [here][Filter cache ListWatch using selectors]

**IMPORTANT NOTE**
Requests to a client backed by a filtered cache for objects that do not match the filter will never return anything. In other words, filtered caches make the filtered-out objects invisible to the client. 

## How is this done ?

- When creating the manager, you can override the default NewCache function
- Each client.Object can be filtered with labels and fields

## Examples

In this scenario, the user will override the NewCache function to filter the secret object by it's label. This will return a filtered cache for objects that match the filter.

```yaml
mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
   NewCache: cache.BuilderWithOptions(cache.Options{
	  SelectorsByObject: cache.SelectorsByObject{
	    &corev1.Secret{}: {
	 	  Label: labels.SelectorFromSet(labels.Set{"app": "app-name"}),
	    },
      },
	}),
})
```

In this scenario, the user will override the NewCache function to filter the node object by it's field name. This will return a filtered cache for objects that match the filter.

```yaml
mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
   NewCache: cache.BuilderWithOptions(cache.Options{
	  SelectorsByObject: cache.SelectorsByObject{
	    &corev1.Node{}: {
            Field: fields.SelectorFromSet(fields.Set{"metadata.name": "node01"}),
        },
      },
	}),
})
```

[Filter cache ListWatch using selectors]: https://github.com/kubernetes-sigs/controller-runtime/blob/master/designs/use-selectors-at-cache.md