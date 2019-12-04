// Copyright 2019 The Operator-SDK Authors
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
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra/doc"

	"github.com/operator-framework/operator-sdk/cmd/operator-sdk/cli"
)

func main() {
	root := cli.GetCLIRoot()
	root.DisableAutoGenTag = true

	currentDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get current directory: %v", err)
	}

	docPath := filepath.Join(currentDir, "doc", "cli")

	// Remove and recreate the CLI doc directory to ensure that
	// stale files (e.g. from renamed or removed CLI subcommands)
	// are removed.
	if err := os.RemoveAll(docPath); err != nil {
		log.Fatalf("Failed to remove existing generated docs: %v", err)
	}
	if err := os.MkdirAll(docPath, 0755); err != nil {
		log.Fatalf("Failed to re-create docs directory: %v", err)
	}

	err = doc.GenMarkdownTree(root, docPath)
	if err != nil {
		log.Fatalf("Failed to generate documentation: %v", err)
	}
}
