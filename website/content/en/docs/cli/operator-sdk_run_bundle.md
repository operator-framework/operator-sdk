---
title: "operator-sdk run bundle"
---
## operator-sdk run bundle

Deploy an Operator in the bundle format with OLM

### Synopsis

The single argument to this command is a bundle image, with the full registry path specified.
If using a docker.io image, you must specify docker.io(/&lt;namespace&gt;)?/&lt;bundle-image-name&gt;:&lt;tag&gt;.
If the bundle image provided is a SQLite index, it must be pullable by the cluster as SQLite images are pulled from the cluster.
If the bundle image provided is a File-Based Catalog (FBC) index, it will be pulled on the local machine.

The main purpose of this command is to streamline running the bundle without having to provide an index image with the bundle already included.

The `--index-image` flag specifies an index image in which to inject the given bundle. It can be specified to resolve dependencies for a bundle. 
This is an optional flag which will default to `quay.io/operator-framework/opm:latest`.
The index image provided should **NOT** already have the bundle. A limitation of the index image flag is that it does not check the upgrade graph
as the annotations for channels are ignored but it is still a useful flag to have to validate the dependencies. 
For example: It does not fail fast when the bundle version provided is &lt;= ChannelHead.


```
operator-sdk run bundle <bundle-image> [flags]
```

### Options

```
      --ca-secret-name string                     Name of a generic secret containing a PEM root certificate file required to pull bundle images. This secret *must* be in the namespace that this command is configured to run in, and the file *must* be encoded under the key "cert.pem"
      --decompression-image string                image used in an init container in the registry pod to decompress the compressed catalog contents. cat and gzip binaries are expected to exist in the PATH (default "docker.io/library/busybox:1.36.0")
  -h, --help                                      help for bundle
      --index-image string                        index image in which to inject bundle (default "quay.io/operator-framework/opm:latest")
      --install-mode InstallModeValue             install mode
      --kubeconfig string                         Path to the kubeconfig file to use for CLI requests.
  -n, --namespace string                          If present, namespace scope for this CLI request
      --pull-secret-name string                   Name of image pull secret ("type: kubernetes.io/dockerconfigjson") required to pull bundle images. This secret *must* be both in the namespace and an imagePullSecret of the service account that this command is configured to run in
      --security-context-config SecurityContext   specifies the security context to use for the catalog pod. allowed: 'restricted', 'legacy'. (default legacy)
      --service-account string                    Service account name to bind registry objects to. If unset, the default service account is used. This value does not override the operator's service account
      --skip-tls                                  skip authentication of image registry TLS certificate when pulling a bundle image in-cluster
      --skip-tls-verify                           skip TLS certificate verification for container image registries while pulling bundles
      --timeout duration                          Duration to wait for the command to complete before failing (default 2m0s)
      --use-http                                  use plain HTTP for container image registries while pulling bundles
```

### Options inherited from parent commands

```
      --plugins strings   plugin keys to be used for this subcommand execution
      --verbose           Enable verbose logging
```

### SEE ALSO

* [operator-sdk run](../operator-sdk_run)	 - Run an Operator in a variety of environments

