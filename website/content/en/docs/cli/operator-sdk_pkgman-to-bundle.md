---
title: "operator-sdk pkgman-to-bundle"
---
## operator-sdk pkgman-to-bundle

Migrates packagemanifests to bundles

### Synopsis


'pkgman-to-bundle' command helps in migrating OLM packagemanifests to bundles which is the preferred OLM packaging format.
This command takes an input packagemanifest directory and generates bundles for each of the versions of manifests present in
the input directory. Additionally, it also provides the flexibility to build bundle images for each of the generated bundles.

The generated bundles are always written on disk. Location for the generated bundles can be specified using '--output-dir'. If not
specified, the default location would be 'bundle/' directory.

The base container image name for the bundles can be provided using '--image-tag-base' flag. This should be provided without the tag, since the tag
for the images would be the bundle version, (ie) image names will be in the format &lt;base_image&gt;:&lt;bundle_version&gt;.

Specify the build command for building container images using '--build-cmd' flag. The default build command is 'docker build'. The command will
need to be in the 'PATH' or fully qualified path name should be provided.


```
operator-sdk pkgman-to-bundle <packagemanifestdir> [flags]
```

### Examples

```


# Provide the packagemanifests directory as input to the command. Consider the packagemanifests directory to have the following
# structure:

$ tree packagemanifests/
packagemanifests
└── etcd
    ├── 0.0.1
    │   ├── etcdcluster.crd.yaml
    │   └── etcdoperator.clusterserviceversion.yaml
    ├── 0.0.2
    │   ├── etcdbackup.crd.yaml
    │   ├── etcdcluster.crd.yaml
    │   ├── etcdoperator.v0.0.2.clusterserviceversion.yaml
    │   └── etcdrestore.crd.yaml
    └── etcd.package.yaml

# Run the following command to generate bundles in the default 'bundle/' directory with the base-container image name
# to be 'quay.io/example/etcd'
$ operator-sdk pkgman-to-bundle packagemanifests --image-tag-base quay.io/example/etcd
INFO[0000] Packagemanifests will be migrated to bundles in bundle directory
INFO[0000] Creating bundle/bundle-0.0.1/bundle.Dockerfile
INFO[0000] Creating bundle/bundle-0.0.1/metadata/annotations.yaml
...

# After running the above command, the bundles will be generated in 'bundles/' directory.
$ tree bundles/
bundles/
├── bundle-0.0.1
│   ├── bundle
│   │   ├── manifests
│   │   │   ├── etcdcluster.crd.yaml
│   │   │   ├── etcdoperator.clusterserviceversion.yaml
│   │   ├── metadata
│   │   │   └── annotations.yaml
│   │   └── tests
│   │       └── scorecard
│   │           └── config.yaml
│   └── bundle.Dockerfile
└── bundle-0.0.2
    ├── bundle
    │   ├── manifests
    │   │   ├── etcdbackup.crd.yaml
    │   │   ├── etcdcluster.crd.yaml
    │   │   ├── etcdoperator.v0.0.2.clusterserviceversion.yaml
    │   │   ├── etcdrestore.crd.yaml
    │   └── metadata
    │       └── annotations.yaml
    └── bundle.Dockerfile

A custom command to build bundle images can also be specified using the '--build-cmd' flag. For example,

$ operator-sdk pkgman-to-bundle packagemanifests --image-tag-base quay.io/example/etcd --build-cmd "podman build -f bundle.Dockerfile . -t"

Images for the both the bundles will be built with the following names: quay.io/example/etcd:0.0.1 and quay.io/example/etcd:0.0.2.

```

### Options

```
      --build-cmd string        Build command to be run for building images. By default 'docker build' is run.
  -h, --help                    help for pkgman-to-bundle
      --image-tag-base string   Base container image name for bundle image tags, ex. my.reg/foo/bar-operator-bundle will become my.reg/foo/bar-operator-bundle:${package-dir-name} for each child directory name in the packagemanifests directory
      --output-dir string       Directory to write bundle to. (default "bundles")
```

### Options inherited from parent commands

```
      --plugins strings   plugin keys to be used for this subcommand execution
      --verbose           Enable verbose logging
```

### SEE ALSO

* [operator-sdk](../operator-sdk)	 - 

