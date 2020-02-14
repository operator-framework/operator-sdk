## operator-sdk build

Compiles code and builds artifacts

### Synopsis

The operator-sdk build command compiles the Operator code into an executable binary
and generates the Dockerfile manifest.

<image> is the container image to be built, e.g. "quay.io/example/operator:v0.0.1".
By default, this image will be automatically set in the deployment manifests. Note that you can use
the flag --skip-image to skip building the container image and only build the operator binary.

After build completes, the image would be built locally in docker. Then it needs to
be pushed to remote registry.
For example:

	$ operator-sdk build quay.io/example/operator:v0.0.1
	$ docker push quay.io/example/operator:v0.0.1


```
operator-sdk build [<image>] [flags]
```

### Options

```
      --go-build-args string      Extra Go build arguments as one string such as "-ldflags -X=main.xyz=abc"
  -h, --help                      help for build
      --image-build-args string   Extra image build arguments as one string such as "--build-arg https_proxy=$https_proxy"
      --image-builder string      Tool to build OCI images. One of: [docker, podman, buildah] (default "docker")
      --skip-image                If set, only the operator binary is built and the container image build is skipped.
```

### SEE ALSO

* [operator-sdk](operator-sdk.md)	 - An SDK for building operators with ease

