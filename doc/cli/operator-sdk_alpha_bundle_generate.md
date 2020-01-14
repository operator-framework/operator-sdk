## operator-sdk alpha bundle generate

Generate operator bundle metadata and Dockerfile

### Synopsis

The 'operator-sdk bundle generate' command will generate operator
bundle metadata in <directory-arg>/metadata and a Dockerfile to build an
operator bundle image in <directory-arg>.

Unlike 'build', 'generate' does not build the image, permitting use of a
non-default image builder

NOTE: modifying generated metadata is not recommended and may corrupt the
resulting image.


```
operator-sdk alpha bundle generate [flags]
```

### Examples

```
The following command will generate metadata and a Dockerfile defining
a test-operator bundle image containing manifests for package channels
'stable' and 'beta':

$ operator-sdk bundle generate \
    --directory ./deploy/olm-catalog/test-operator \
    --package test-operator \
    --channels stable,beta \
    --default stable

```

### Options

```
  -c, --channels string    The list of channels that bundle image belongs to
  -e, --default string     The default channel for the bundle image
  -d, --directory string   The directory where bundle manifests are located.
  -h, --help               help for generate
  -p, --package string     The name of the package that bundle image belongs to
```

### SEE ALSO

* [operator-sdk alpha bundle](operator-sdk_alpha_bundle.md)	 - Operator bundle commands

