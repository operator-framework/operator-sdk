// Copyright 2020 The Operator-SDK Authors
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
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"golang.org/x/mod/modfile"
	"golang.org/x/tools/imports"
)

func isPathTypeGo(path string) (bool, error) {
	return filepath.Ext(path) == ".go", nil
}

func (p project) fixPathGo(filePath string) string {

	pathSplit := strings.Split(filepath.Clean(filePath), sep)
	if len(pathSplit) < 2 || filePath == "go.mod" || filePath == "go.sum" {
		return filePath
	}

	switch {
	case pathSplit[0] == "deploy":
		filePath = filepath.Join("config", filepath.Join(pathSplit[1:]...))
	case pathSplit[0] == "pkg" && pathSplit[1] == "apis":
		if p.multiGroup {
			filePath = filepath.Join(pathSplit[1:]...)
		} else {
			// Subdirectory of pkg/apis
			if len(pathSplit) > 3 {
				pathSplit = pathSplit[1:]
			}
			filePath = filepath.Join("api", filepath.Join(pathSplit[2:]...))
		}
	case pathSplit[0] == "pkg" && pathSplit[1] == "controller":
		// Subdirectory of pkg/controller
		if !p.multiGroup && len(pathSplit) > 3 {
			pathSplit = pathSplit[1:]
		}
		filePath = filepath.Join("controllers", filepath.Join(pathSplit[2:]...))
	case pathSplit[0] == "cmd" && pathSplit[1] == "manager":
		filePath = filepath.Join(pathSplit[2:]...)
	case pathSplit[0] == "build" && pathSplit[1] == "Dockerfile":
		filePath = pathSplit[1]
	case pathSplit[0] == "build" && pathSplit[1] == "bin":
		filePath = filepath.Join(pathSplit[1:]...)
	}

	return addOldExt(filePath)
}

func (p project) importsWalkFuncGo(pathChanges map[string]string) filepath.WalkFunc {
	return func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		b, err := afero.ReadFile(p.fs, path)
		if err != nil {
			return err
		}
		if filepath.Ext(path) == ".go" {
			for newPath, oldPath := range pathChanges {
				b = fixImportsGo(b, p.repo, newPath, oldPath)
			}
			if b, err = imports.Process("", b, nil); err != nil {
				return fmt.Errorf("error processing %s: %v", path, err)
			}
		}
		return afero.WriteFile(p.fs, path, b, info.Mode())
	}
}

func fixImportsGo(b []byte, repo, newPath, oldPath string) []byte {
	if oldPath == "." || oldPath == "" {
		return b
	}
	if newPath == "." {
		newPath = ""
	}
	newPkg := path.Join(repo, filepath.ToSlash(newPath))
	oldPkg := path.Join(repo, filepath.ToSlash(oldPath))
	if newPkg == oldPkg || newPkg == repo {
		return b
	}
	log.Debugf("Replacing strings in file %s:\n\tfrom: %q\n\tto:   %q", oldPath, oldPkg, newPkg)
	b = bytes.ReplaceAll(b, []byte(`"`+oldPkg+`"`), []byte(`"`+newPkg+`"`))
	// Also replace package name.
	b = bytes.ReplaceAll(b, []byte(" "+path.Base(oldPkg)+"."), []byte(" "+path.Base(newPkg)+"."))
	return b
}

var (
	modFileGoRegexp     = regexp.MustCompile(`\bgo [1-2]\.[1-9]([1-9])?\b`)
	modFileModuleRegexp = regexp.MustCompile(`\bmodule [^\n]+\b`)
)

func parseModFile(dir string) (*modfile.File, error) {
	path := filepath.Join(dir, "go.mod")
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return modfile.ParseLax(path, b, nil)
}

// mergeModFiles merges modA and modB into modA.
func mergeModFiles(modA, modB *modfile.File) error {
	ab, err := modA.Format()
	if err != nil {
		return err
	}
	bb, err := modB.Format()
	if err != nil {
		return err
	}
	bb = modFileGoRegexp.ReplaceAll(bb, []byte{})
	bb = modFileModuleRegexp.ReplaceAll(bb, []byte{})
	newMod, err := modfile.ParseLax("", append(ab, bb...), nil)
	if err != nil {
		return err
	}
	*modA = *newMod
	return nil
}
