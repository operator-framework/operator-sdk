---
title: Setting Override Values in Helm-based Operators
linkTitle: Override Values
weight: 100
description: Learn how to set override values and pass environment variables to your Helm chart.
---

Sometimes it is useful to pass down environment variables from the Operators `Deployment`
all the way to the helm charts templates. This allows the Operator to be configured at a global
level at runtime. This is new compared to dealing with the helm CLI
as they usually don't have access to any environment variables in the context of Tiller (helm v2)
or the helm binary (helm v3) for security reasons.

With the helm Operator this becomes possible by override values. This enforces that certain
template values provided by the chart's default `values.yaml` or by a CR spec are always set
when rendering the chart. If the value is set by a CR it gets overridden by the global override value.
The override value can be static but can also refer to an environment variable. To pass down environment
variables to the chart override values is currently the only way.

An example use case of this is when your helm chart references container images by chart variables,
which is a good practice.
If your Operator is deployed in a disconnected environment (no network access to the default images
location) you can use this mechanism to set them globally at the Operator level using environment variables
versus individually per CR / chart release.

> Note that it is strongly recommended to reference container images in your chart by helm variables
> and then also associate these with an environment variable of your Operator like shown below.
> This allows your Operator to be mirrored for offline usage when packaged for OLM.

To configure your operator with override values, add an `overrideValues` map to your
`watches.yaml` file for the GVK and chart you need to override. For example, to change
the repository used by the nginx chart, you would update your `watches.yaml` to the
following:

```yaml
# Use the 'create api' subcommand to add watches to this file.
- group: example.com
  version: v1alpha1
  kind: Nginx
  chart: helm-charts/nginx
  overrideValues:
    image.repository: quay.io/mycustomrepo
```

By setting `image.repository` to `quay.io/mycustomrepo` you are ensuring that
`quay.io/mycustomrepo` will always be used instead of the chart's default repository
(`nginx`). If the CR attempts to set this value, it will be ignored.

It is now possible to reference environment variables in the `overrideValues` section:

```yaml
  overrideValues:
    image.repository: $IMAGE_REPOSITORY # or ${IMAGE_REPOSITORY}
```

By using an environment variable reference in `overrideValues` you enable these override
values to be set at runtime by configuring the environment variable on the
operator deployment. For example, in `config/manager/manager.yaml` you could add the
following snippet to the container spec:

```yaml
env:
  - name: IMAGE_REPOSITORY
    value: quay.io/mycustomrepo
```

If an environment variable reference is listed in `overrideValues`, but is not present
in the environment when the operator runs, it will resolve to an empty string and
override all other values. Therefore, these environment variables should _always_ be
set. It is suggested to update the Dockerfile to set these environment variables to
the same defaults that are defined by the chart.

To warn users that their CR settings may be ignored, the Helm operator creates events on
the CR that include the name and value of each overridden value. For example:

```
$ kubectl describe nginxes.example.com
...
Events:
  Type     Reason               Age   From              Message
  ----     ------               ----  ----              -------
  Warning  OverrideValuesInUse  1m    nginx-controller  Chart value "image.repository" overridden to "quay.io/mycustomrepo" by operator's watches.yaml
```
