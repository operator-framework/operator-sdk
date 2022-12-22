---
title: Custom Bundle Validation
weight: 80
---

## Summary

Operator authors can now use "external" validators with the
`operator-sdk bundle validate` command by using the
`--alpha-select-external` flag. This feature enables Operator authors,
users, and registry pipelines to use custom validators. These custom
validators can be written in any language.

## Usage

External validators can be used by specifying a list of local filepaths to
executables using colons as path separators:

```sh
$ operator-sdk bundle validate \
--alpha-select-external path/validator1:path/validator2
```

## Writing a Custom Validator

For a validator to work with `operator-sdk bundle validate` each of the files must:
1. Be executable with appropriate permissions
1. Return JSON to STDOUT in the [`ManifestResult`][manifest_result] format.

### Custom Validator from Scratch

Using the `errors` package from [`operator-framework/api`][of-api], we
can start by validating the correct number of arguments and marshaling a
[`ManifestResult`][manifest_result] into STDOUT.

`myvalidator/main.go`

```go
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/operator-framework/api/pkg/validation/errors"
)

func main() {

	// we expect a single argument which is the bundle root.
	// usage: validator-poc <bundle root>
	if len(os.Args) < 2 {
		fmt.Printf("usage: %s <bundle root>\n", os.Args[0])
		os.Exit(1)
	}

	var validatorErrors []errors.Error
	var validatorWarnings []errors.Error
	result := errors.ManifestResult{
		Name:     "Always Green Example",
		Errors:   validatorErrors,
		Warnings: validatorWarnings,
	}
	prettyJSON, err := json.MarshalIndent(result, "", "    ")
	if err != nil {
		fmt.Println("Invalid json")
		os.Exit(1)
	}
	fmt.Printf("%s\n", string(prettyJSON))
}
```

When executed on its own, this validator prints a JSON representation of
[`ManifestResult`][manifest_result].

```sh
go build -o myvalidator/main myvalidator/main.go && ./myvalidator/main

{
    "Name": "Always Green Example",
    "Errors": null,
    "Warnings": null
}
```

```sh
$ go build -o myvalidator/main myvalidator/main.go
$ operator-sdk bundle validate ./bundle --alpha-select-external ./myvalidator/main
```
```
INFO[0000] All validation tests have completed successfully
```

From here, custom validator authors can read in the bundle and make any
assertions necessary.

Errors and Warnings are both implementations of the `error` interface
and need `ErrorType`, `Level`, `Field`, `BadValue`, and `Detail`, which
are all initialized by arbitrary strings. When using Golang, validator
authors can use the [operator-framework/api][of-api] impementation of
[errors and warnings][errors-pkg]

```go
validatorErrors = []errors.Error{errors.Error{"someErrorType", "somelevel", "somefield", "somebadvalue", "somedetail"}}
validatorWarnings = []errors.Error{errors.Error{"someWarningType", "somelevel", "somefield", "somebadvalue", "somedetail"}}
```

We can now rebuild and run the validator, which now shows errors.

```sh
$ go build -o myvalidator/main myvalidator/main.go
$ operator-sdk bundle validate ./bundle --alpha-select-external ./myvalidator/main
```
```
WARN[0000] somelevel: Field somefield, Value somebadvalue: somedetail
ERRO[0000] somelevel: Field somefield, Value somebadvalue: somedetail
```

### Composing Validators

For users wishing to use validators from
[`operator-framework/api`][of-api] without being restricted to the
version that is built into the `operator-sdk` binary, it is possible to
create a `main.go` that makes use of the [`validation`
package][of-validation] at an arbitrary version.

Currently, some of the code necessary requires copying code from
internal packages, which may someday become a library.

`myvalidator/main.go`

```go
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	apimanifests "github.com/operator-framework/api/pkg/manifests"
	apivalidation "github.com/operator-framework/api/pkg/validation"
	registrybundle "github.com/operator-framework/operator-registry/pkg/lib/bundle"
	log "github.com/sirupsen/logrus"

	"github.com/spf13/afero"
	"sigs.k8s.io/yaml"
)

func main() {

	// we expect a single argument which is the bundle root.
	// usage: validator-poc <bundle root>
	if len(os.Args) < 2 {
		fmt.Printf("usage: %s <bundle root>\n", os.Args[0])
		os.Exit(1)
	}

	// Read the bundle object and metadata from the passed in directory.
	bundle, _, err := getBundleDataFromDir(os.Args[1])
	if err != nil {
		fmt.Printf("problem getting bundle [%s] data, %v\n", os.Args[1], err)
		os.Exit(1)
	}

	// pass the objects to the validator
	objs := bundle.ObjectsToValidate()
	for _, obj := range bundle.Objects {
		objs = append(objs, obj)
	}
	results := apivalidation.GoodPracticesValidator.Validate(objs...)

	// take each of the ManifestResults and print to STDOUT
	for _, result := range results {
		prettyJSON, err := json.MarshalIndent(result, "", "    ")
		if err != nil {
			// should output JSON so that the call knows how to parse it
			fmt.Printf("XXX ERROR: %v\n", err)
		}
		fmt.Printf("%s\n", string(prettyJSON))
	}
}

// getBundleDataFromDir returns the bundle object and associated metadata from dir, if any.
func getBundleDataFromDir(dir string) (*apimanifests.Bundle, string, error) {
	// Gather bundle metadata.
	metadata, _, err := FindBundleMetadata(dir)
	if err != nil {
		return nil, "", err
	}
	manifestsDirName, hasLabel := metadata.GetManifestsDir()
	if !hasLabel {
		manifestsDirName = registrybundle.ManifestsDir
	}
	manifestsDir := filepath.Join(dir, manifestsDirName)
	// Detect mediaType.
	mediaType, err := registrybundle.GetMediaType(manifestsDir)
	if err != nil {
		return nil, "", err
	}
	// Read the bundle.
	bundle, err := apimanifests.GetBundleFromDir(manifestsDir)
	if err != nil {
		return nil, "", err
	}
	return bundle, mediaType, nil
}

// -------------------------------------------------------
// Everything below this line was copied code from the internal Operator SDK
// registry package operator-sdk/internal/registry/labels.go. If this is
// generally useful please file an issue to move this to a reuable library.
// to make this a library or other reusable code.
// -------------------------------------------------------

type MetadataNotFoundError string

func (e MetadataNotFoundError) Error() string {
	return fmt.Sprintf("metadata not found in %s", string(e))
}

// Labels is a set of key:value labels from an operator-registry object.
type Labels map[string]string

// GetManifestsDir returns the manifests directory name in ls using
// a predefined key, or false if it does not exist.
func (ls Labels) GetManifestsDir() (string, bool) {
	value, hasKey := ls[registrybundle.ManifestsLabel]
	return filepath.Clean(value), hasKey
}

// FindBundleMetadata walks bundleRoot searching for metadata (ex. annotations.yaml),
// and returns metadata and its path if found. If one is not found, an error is returned.
func FindBundleMetadata(bundleRoot string) (Labels, string, error) {
	return findBundleMetadata(afero.NewOsFs(), bundleRoot)
}

func findBundleMetadata(fs afero.Fs, bundleRoot string) (Labels, string, error) {
	// Check the default path first, and return annotations if they were found or an error if that error
	// is not because the path does not exist (it exists or there was an unmarshalling error).
	annotationsPath := filepath.Join(bundleRoot, registrybundle.MetadataDir, registrybundle.AnnotationsFile)
	annotations, err := readAnnotations(fs, annotationsPath)
	if (err == nil && len(annotations) != 0) || (err != nil && !errors.Is(err, os.ErrNotExist)) {
		return annotations, annotationsPath, err
	}

	// Annotations are not at the default path, so search recursively.
	annotations = make(Labels)
	annotationsPath = ""
	err = afero.Walk(fs, bundleRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// Skip directories and hidden files, or if annotations were already found.
		if len(annotations) != 0 || info.IsDir() || strings.HasPrefix(path, ".") {
			return nil
		}

		annotationsPath = path
		// Ignore this error, since we only care if any annotations are returned.
		if annotations, err = readAnnotations(fs, path); err != nil {
			log.Debug(err)
		}
		return nil
	})
	if err != nil {
		return nil, "", err
	}

	if len(annotations) == 0 {
		return nil, "", MetadataNotFoundError(bundleRoot)
	}

	return annotations, annotationsPath, nil
}

// readAnnotations reads annotations from file(s) in bundleRoot and returns them as Labels.
func readAnnotations(fs afero.Fs, annotationsPath string) (Labels, error) {
	// The annotations file is well-defined.
	b, err := afero.ReadFile(fs, annotationsPath)
	if err != nil {
		return nil, err
	}

	// Use the arbitrarily-labelled bundle representation of the annotations file
	// for forwards and backwards compatibility.
	annotations := registrybundle.AnnotationMetadata{
		Annotations: make(Labels),
	}
	if err = yaml.Unmarshal(b, &annotations); err != nil {
		return nil, fmt.Errorf("error unmarshalling potential bundle metadata %s: %v", annotationsPath, err)
	}

	return annotations.Annotations, nil
}
```

The `main.go` is then built into a binary and used with `operator-sdk bundle
validate`

```sh
$ go build -o myvalidator/main myvalidator/main.go
$ operator-sdk bundle validate ./bundle --alpha-select-external ./myvalidator/main
```
```
WARN[0000] Warning: Value sandbox-op.v0.0.1: owned CRD "sandboxes.sandbox.example.come" has an empty description
INFO[0000] All validation tests have completed successfully
```
[errors-pkg]: https://github.com/operator-framework/api/tree/master/pkg/validation/errors
[manifest_result]: https://github.com/operator-framework/api/blob/master/pkg/validation/errors/error.go#L9-L16
[of-api]: https://github.com/operator-framework/api
[of-validation]: https://github.com/operator-framework/api/tree/master/pkg/validation
