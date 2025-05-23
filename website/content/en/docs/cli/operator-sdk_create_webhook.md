---
title: "operator-sdk create webhook"
---
## operator-sdk create webhook

Scaffold a webhook for an API resource

### Synopsis

Scaffold a webhook for an API resource. You can choose to scaffold defaulting,
validating and/or conversion webhooks.


```
operator-sdk create webhook [flags]
```

### Examples

```
  # Create defaulting and validating webhooks for Group: ship, Version: v1beta1
  # and Kind: Frigate
  operator-sdk create webhook --group ship --version v1beta1 --kind Frigate --defaulting --programmatic-validation

  # Create conversion webhook for Group: ship, Version: v1beta1
  # and Kind: Frigate
  operator-sdk create webhook --group ship --version v1beta1 --kind Frigate --conversion --spoke v1

```

### Options

```
      --conversion                   if set, scaffold the conversion webhook
      --defaulting                   if set, scaffold the defaulting webhook
      --external-api-domain string   Specify the domain name for the external API. This domain is used to generate accurate RBAC markers and permissions for the external resources (e.g., cert-manager.io).
      --external-api-path string     Specify the Go package import path for the external API. This is used to scaffold controllers for resources defined outside this project (e.g., github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1).
      --force                        attempt to create resource even if it already exists
      --group string                 resource Group
  -h, --help                         help for webhook
      --kind string                  resource Kind
      --legacy                       [DEPRECATED] Attempts to create resource under the API directory (legacy path). This option will be removed in future versions.
      --make make generate           if true, run make generate after generating files (default true)
      --plural string                resource irregular plural form
      --programmatic-validation      if set, scaffold the validating webhook
      --spoke strings                Comma-separated list of spoke versions to be added to the conversion webhook (e.g., --spoke v1,v2)
      --version string               resource Version
```

### Options inherited from parent commands

```
      --plugins strings   plugin keys to be used for this subcommand execution
      --verbose           Enable verbose logging
```

### SEE ALSO

* [operator-sdk create](../operator-sdk_create)	 - Scaffold a Kubernetes API or webhook

