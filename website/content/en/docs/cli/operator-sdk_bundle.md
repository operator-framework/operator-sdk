---
title: "operator-sdk bundle"
---
## operator-sdk bundle

Manage operator bundle metadata

### Synopsis


Manage bundle builds, bundle metadata generation, and bundle validation.
An operator bundle is a portable operator packaging format understood by Kubernetes
native software, like the Operator Lifecycle Manager.

The bundle generate and in this command follow the Operator Registry Manifests format.
Note that, the bundle metadata and bundle images will be validated following the Operator Registry rules.

And then, for further information over the integration with OLM via SDK see its docs:
https://sdk.operatorframework.io/docs/olm-integration/

Notes:
* More info about OLM: https://github.com/operator-framework/operator-lifecycle-manager.
* More info about the bundle format see: https://github.com/operator-framework/operator-registry#manifest-format.


### Options

```
  -h, --help   help for bundle
```

### SEE ALSO

* [operator-sdk](../operator-sdk)	 - An SDK for building operators with ease
* [operator-sdk bundle create](../operator-sdk_bundle_create)	 - Create an operator bundle image
* [operator-sdk bundle validate](../operator-sdk_bundle_validate)	 - Validate an operator bundle image

