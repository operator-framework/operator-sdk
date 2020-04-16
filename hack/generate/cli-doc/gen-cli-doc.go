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
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra/doc"

	"github.com/operator-framework/operator-sdk/cmd/operator-sdk/cli"
)

const fmTemplate = `---
title: "%s"
---
`

func main() {
	// TODO: Generate CLI docs for plugins CLI at doc/cli/kubebuilder
	root := cli.GetCLIRoot()
	root.DisableAutoGenTag = true

	filePrepender := func(filename string) string {
		name := filepath.Base(filename)
		base := strings.TrimSuffix(name, path.Ext(name))
		return fmt.Sprintf(fmTemplate, strings.Replace(base, "_", " ", -1))
	}
	linkHandler := func(name string) string {
		base := strings.TrimSuffix(name, path.Ext(name))
		return "../" + base
	}

	currentDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get current directory: %v", err)
	}

	docPath := filepath.Join(currentDir, "website", "content", "en", "docs", "cli")

	// Remove and recreate the CLI doc directory to ensure that
	// stale files (e.g. from renamed or removed CLI subcommands)
	// are removed.
	if err := os.Rename(docPath+"/_index.md", currentDir+"/tmp_index.md"); err != nil {
		log.Fatalf("Failed to move existing index: %v", err)
	}
	if err := os.RemoveAll(docPath); err != nil {
		log.Fatalf("Failed to remove existing generated docs: %v", err)
	}
	if err := os.MkdirAll(docPath, 0755); err != nil {
		log.Fatalf("Failed to re-create docs directory: %v", err)
	}

	err = doc.GenMarkdownTreeCustom(root, docPath, filePrepender, linkHandler)
	if err != nil {
		log.Fatalf("Failed to generate documentation: %v", err)
	}

	if err := os.Rename(currentDir+"/tmp_index.md", docPath+"/_index.md"); err != nil {
		log.Fatalf("Failed to move existing index: %v", err)
	}
}
