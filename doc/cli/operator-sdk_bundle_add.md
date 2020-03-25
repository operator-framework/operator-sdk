## operator-sdk bundle add

Add an operator bundle image to an operator index image

### Synopsis

The 'operator-sdk bundle add' command will add an operator bundle image to an
existing operator index image, or create an index image.

This command downloads and shells out to the 'opm' binary under the hood. The
version downloaded by 'bundle add' is: v1.6.0

Bundle images being passed to 'bundle add' must be present remotely, and access
to the remote repository should be enabled in the command line environment.

More information on operator index images:
https://github.com/openshift/enhancements/blob/master/enhancements/olm/operator-registry.md
More information on 'opm':
https://github.com/operator-framework/operator-registry/blob/master/docs/design/opm-tooling.md


```
operator-sdk bundle add [flags]
```

### Examples

```
The following invocation will create a new test-operator bundle index image:

  $ operator-sdk bundle add quay.io/example/test-operator:v0.1.0 \
      --to-index quay.io/example/test-operator-index:v0.1.0

The following invocation will add a test-operator bundle image to an existing
index image at version v0.1.0, creating a new index image at version v0.2.0:

  $ operator-sdk bundle add quay.io/example/test-operator:v0.2.0 \
      --from-index quay.io/example/test-operator-index:v0.1.0 \
      --to-index quay.io/example/test-operator-index:v0.2.0

```

### Options

```
      --dockerfile-name string   Name of the Dockerfile to generate if --generate-only is set. Default is 'Dockerfile'
  -f, --from-index string        Previous index to build new index image from
  -g, --generate-only            Generate the underlying database and a Dockerfile without building the index container image
  -h, --help                     help for add
      --image-builder string     Tool to build container images. One of: [docker, podman] (default "docker")
      --permissive               Allow registry load errors without exiting the build
  -t, --to-index string          Tag for new index image being built
```

### SEE ALSO

* [operator-sdk bundle](operator-sdk_bundle.md)	 - Work with operator bundle metadata and operator bundle and index images

