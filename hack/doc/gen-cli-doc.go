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
	"time"

	"github.com/operator-framework/operator-sdk/cmd/operator-sdk/cli"

	"github.com/spf13/cobra/doc"

	log "github.com/sirupsen/logrus"
)

func main() {
	root := cli.GetCLIRoot()
	currentDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get current directory, (%v)", err)
	}
	const fmTemplate = `---
date: %s
title: "%s"
---
`

	filePrepender := func(filename string) string {
		now := time.Now().Format(time.RFC3339)
		name := filepath.Base(filename)
		base := strings.TrimSuffix(name, path.Ext(name))
		return fmt.Sprintf(fmTemplate, now, strings.Replace(base, "_", " ", -1))
	}

	linkHandler := func(name string) string {
		base := strings.TrimSuffix(name, path.Ext(name))
		return base
	}

	err = doc.GenMarkdownTreeCustom(root, currentDir+"/doc/cli", filePrepender, linkHandler)
	if err != nil {
		log.Fatalf("Failed to generate documenation, (%v)", err)
	}
}
