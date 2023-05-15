// Copyright 2023 The Operator-SDK Authors
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

package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/containers/image/v5/docker"
)

const (
	help = `digester get the the digest of an image

usage:
	digester <image-name> - returns the image digest for image-name

flags:
	-h, --help - prints this help`

	digestPrefix = "@sha256:"
)

func main() {
	if len(os.Args) != 2 {
		_, _ = fmt.Fprintln(os.Stderr, "Error: must have exactly one parameter of an image name")
		_, _ = fmt.Fprintln(os.Stderr, help)
		os.Exit(1)
	} else if os.Args[1] == "-h" || os.Args[1] == "--help" {
		_, _ = fmt.Println(help)
		os.Exit(0)
	}

	imageToDigest := os.Args[1]

	if strings.Contains(imageToDigest, digestPrefix) {
		_, _ = fmt.Fprintf(os.Stderr, "%q is already in a digest format\n", imageToDigest)
		os.Exit(1)
	}

	imgRef, err := docker.ParseReference("//" + imageToDigest)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to parse image reference; %v\n", err)
		os.Exit(1)
	}

	digest, err := docker.GetDigest(context.Background(), nil, imgRef)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to get digest for %q; %v\n", imageToDigest, err)
		os.Exit(1)
	}

	fmt.Println(digest)
}
