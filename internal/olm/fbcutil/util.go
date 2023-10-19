// Copyright 2022 The Operator-SDK Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package fbcutil

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/operator-framework/operator-registry/alpha/action"
	"github.com/operator-framework/operator-registry/alpha/declcfg"
	declarativeconfig "github.com/operator-framework/operator-registry/alpha/declcfg"
	"github.com/operator-framework/operator-registry/pkg/containertools"
	"github.com/operator-framework/operator-registry/pkg/image/containerdregistry"
	registryutil "github.com/operator-framework/operator-sdk/internal/registry"
	log "github.com/sirupsen/logrus"
)

const (
	SchemaChannel   = "olm.channel"
	SchemaPackage   = "olm.package"
	DefaultChannel  = "operator-sdk-run-bundle"
	DefaultCacheDir = "operator-sdk-run-bundle-cache"
)

const (
	// defaultIndexImageBase is the base for defaultIndexImage. It is necessary to separate
	// them for string comparison when defaulting bundle add mode.
	DefaultIndexImageBase = "quay.io/operator-framework/opm:"
	// DefaultIndexImage is the index base image used if none is specified. It contains no bundles.
	// TODO(v2.0.0): pin this image tag to a specific version.
	DefaultIndexImage = DefaultIndexImageBase + "latest"
	// DefaultInitImage is the default image to be used in the registry init container
	DefaultInitImage = "docker.io/library/busybox:1.36.0"
)

// BundleDeclcfg represents a minimal File-Based Catalog.
// This struct only consists of one Package, Bundle, and Channel blob. It is used to
// represent the bundle image in the File-Based Catalog format.
type BundleDeclcfg struct {
	Package declcfg.Package
	Channel declcfg.Channel
	Bundle  declcfg.Bundle
}

// FBCContext is a struct that stores all the required information while constructing
// a new File-Based Catalog on the fly. The fields from this struct are passed as
// parameters to Operator Registry API calls to generate declarative config objects.
type FBCContext struct {
	Package       string
	ChannelName   string
	Refs          []string
	ChannelEntry  declarativeconfig.ChannelEntry
	SkipTLSVerify bool
	UseHTTP       bool
}

// CreateFBC generates an FBC by creating bundle, package and channel blobs.
func (f *FBCContext) CreateFBC(ctx context.Context) (BundleDeclcfg, error) {
	var bundleDC BundleDeclcfg
	// Rendering the bundle image into a declarative config format.
	declcfg, err := RenderRefs(ctx, f.Refs, f.SkipTLSVerify, f.UseHTTP)
	if err != nil {
		return BundleDeclcfg{}, err
	}

	// Ensuring a valid bundle size.
	if len(declcfg.Bundles) != 1 {
		return BundleDeclcfg{}, fmt.Errorf("bundle image should contain exactly one bundle blob")
	}

	bundleDC.Bundle = declcfg.Bundles[0]

	// generate package.
	bundleDC.Package = declarativeconfig.Package{
		Schema:         SchemaPackage,
		Name:           f.Package,
		DefaultChannel: f.ChannelName,
	}

	// generate channel.
	bundleDC.Channel = declarativeconfig.Channel{
		Schema:  SchemaChannel,
		Name:    f.ChannelName,
		Package: f.Package,
		Entries: []declarativeconfig.ChannelEntry{f.ChannelEntry},
	}

	return bundleDC, nil
}

// ValidateAndStringify first converts the generated declarative config to a model and validates it.
// If the declarative config model is valid, it will convert the declarative config to a YAML string and return it.
func ValidateAndStringify(declcfg *declarativeconfig.DeclarativeConfig) (string, error) {
	// validates and converts declarative config to model
	_, err := declarativeconfig.ConvertToModel(*declcfg)
	if err != nil {
		return "", fmt.Errorf("error converting the declarative config to model: %v", err)
	}

	var buf bytes.Buffer
	err = declarativeconfig.WriteYAML(*declcfg, &buf)
	if err != nil {
		return "", fmt.Errorf("error writing generated declarative config to JSON encoder: %v", err)
	}

	if buf.String() == "" {
		return "", errors.New("file-based catalog contents cannot be empty")
	}

	return buf.String(), nil
}

func NullLogger() *log.Entry {
	logger := log.New()
	logger.SetOutput(io.Discard)
	return log.NewEntry(logger)
}

// RenderRefs will invoke Operator Registry APIs and return a declarative config object representation
// of the references that are passed in as a string array.
func RenderRefs(ctx context.Context, refs []string, skipTLSVerify bool, useHTTP bool) (*declarativeconfig.DeclarativeConfig, error) {
	cacheDir := dirNameFromRefs(refs)

	if cacheDir == "" {
		cacheDir = DefaultCacheDir
	}
	reg, err := containerdregistry.NewRegistry(
		containerdregistry.WithLog(NullLogger()),
		containerdregistry.SkipTLSVerify(skipTLSVerify),
		containerdregistry.WithPlainHTTP(useHTTP),
		containerdregistry.WithCacheDir(cacheDir))
	if err != nil {
		return nil, fmt.Errorf("error creating new image registry: %v", err)
	}

	defer func() {
		err = reg.Destroy()
		if err != nil {
			log.Warn(fmt.Sprintf("Unable to cleanup registry. You may have to manually cleanup by removing the %q directory", cacheDir))
		}
	}()

	render := action.Render{
		Refs:     refs,
		Registry: reg,
	}

	log.SetOutput(io.Discard)
	declcfg, err := render.Run(ctx)
	log.SetOutput(os.Stdout)
	if err != nil {
		return nil, fmt.Errorf("error in rendering the bundle and index image: %v", err)
	}

	return declcfg, nil
}

func dirNameFromRefs(refs []string) string {
	dirNameBytes := []byte(strings.ReplaceAll(strings.Join(refs, "_"), "/", "-"))
	hash := sha256.New()
	hash.Write(dirNameBytes)
	hashBytes := hash.Sum(nil)
	return fmt.Sprintf("%x", hashBytes)
}

// IsFBC will determine if an index image uses the File-Based Catalog or SQLite index image format.
// The default index image will adopt the File-Based Catalog format.
func IsFBC(ctx context.Context, indexImage string) (bool, error) {
	// adding updates to the IndexImageCatalogCreator if it is an FBC image
	catalogLabels, err := registryutil.GetImageLabels(ctx, nil, indexImage, false)
	if err != nil {
		return false, fmt.Errorf("get index image labels: %v", err)
	}
	_, hasFBCLabel := catalogLabels[containertools.ConfigsLocationLabel]

	return hasFBCLabel || indexImage == DefaultIndexImage, nil
}
