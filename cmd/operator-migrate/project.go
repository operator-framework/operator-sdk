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
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/operator-framework/operator-sdk/internal/util/k8sutil"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"golang.org/x/mod/modfile"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"sigs.k8s.io/kubebuilder/pkg/model/config"
	"sigs.k8s.io/yaml"
)

// File path separator.
const sep = string(filepath.Separator)

// project contains all information required to define an operator project.
type project struct {
	layout     pluginKey
	license    string
	owner      string
	repo       string
	domain     string
	gvks       gvks
	multiGroup bool

	// fs holds parsed project information.
	fs afero.Fs
	// modFile is only used for Go projects.
	modFile *modfile.File
}

// gvk wraps a config resource and adds other useful fields.
type gvk struct {
	config.GVK
	shortGroup string
	crdVersion string
}

type gvks []gvk

// isMultiGroup returns true if more than one group name exists in gvks.
func (gs gvks) isMultiGroup() bool {
	if len(gs) == 0 {
		return false
	}

	firstGroup := gs[0].Group
	for _, gvk := range gs[1:] {
		if firstGroup != gvk.Group {
			return true
		}
	}
	return false
}

// pathChecker is a function that returns true if a file should be modified somehow
// before being copied to the new project.
type pathChecker func(string) (bool, error)

// pathFixer returns some modified string that is based on the input file path.
// That modified string will be used as the new path for the input file path.
type pathFixer func(string) string

// parseFromDir parses a full project from dir.
func (p *project) parseFromDir(dir string) error {
	p.fs = afero.NewMemMapFs()

	seenGVKs := make(map[config.GVK]struct{})
	err := filepath.Walk(filepath.Join(dir, "deploy"), p.getGVKsFromCRDs(seenGVKs))
	if err != nil {
		return err
	}

	p.multiGroup = p.gvks.isMultiGroup()
	if len(p.gvks) != 0 {
		gvk := p.gvks[0]
		p.domain = strings.TrimPrefix(gvk.Group, gvk.shortGroup+".")
	}

	var isPathOfType pathChecker
	var fixPath pathFixer
	switch p.layout {
	case pluginKeyGo:
		isPathOfType = isPathTypeGo
		fixPath = p.fixPathGo
	}

	pathChanges := make(map[string]string)
	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || isFileIgnore(filepath.Base(path)) {
			return err
		}

		oldPath := strings.TrimPrefix(path, dir+sep)
		newPath := fixPath(oldPath)
		log.Debugf("Fix path:\n\tfrom: %s\n\tto:   %s", oldPath, newPath)

		if pathOfType, err := isPathOfType(path); err != nil {
			return err
		} else if pathOfType {
			pathChanges[filepath.Dir(newPath)] = filepath.Dir(oldPath)
		}

		if err = p.fs.MkdirAll(filepath.Dir(newPath), 0777); err != nil {
			return err
		}
		b, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}
		return afero.WriteFile(p.fs, newPath, b, info.Mode())
	})
	if err != nil {
		return err
	}

	var walkFunc filepath.WalkFunc
	switch p.layout {
	case pluginKeyGo:
		walkFunc = p.importsWalkFuncGo(pathChanges)
	}

	return afero.Walk(p.fs, ".", walkFunc)
}

func (p *project) getGVKsFromCRDs(seenGVKs map[config.GVK]struct{}) filepath.WalkFunc {
	return func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || filepath.Ext(path) != ".yaml" {
			return err
		}

		b, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}
		typeMeta, err := k8sutil.GetTypeMetaFromBytes(b)
		if err == nil && typeMeta.Kind == "CustomResourceDefinition" {
			gvks, err := getCRGVKs(b, seenGVKs)
			if err != nil {
				return err
			}
			p.gvks = append(p.gvks, gvks...)
		}
		return nil
	}
}

func isFileIgnore(path string) bool {
	return path == "go.mod" || path == "go.sum" || strings.HasPrefix(path, "zz_generated")
}

func addOldExt(filePath string) string {
	ext := filepath.Ext(filePath)
	return strings.TrimSuffix(filePath, ext) + ".old" + ext
}

func getCRGVKs(crdBytes []byte, seenGVKs map[config.GVK]struct{}) (gvks []gvk, err error) {
	crd := v1beta1.CustomResourceDefinition{}
	if err := yaml.Unmarshal(crdBytes, &crd); err != nil {
		return nil, err
	}

	for _, version := range crd.Spec.Versions {
		groupSplit := strings.SplitN(crd.Spec.Group, ".", 2)

		gvk := gvk{}
		gvk.crdVersion = crd.GetObjectKind().GroupVersionKind().Version
		gvk.shortGroup = groupSplit[0]
		gvk.Group = crd.Spec.Group
		gvk.Version = version.Name
		gvk.Kind = crd.Spec.Names.Kind

		if _, seenGVK := seenGVKs[gvk.GVK]; !seenGVK {
			gvks = append(gvks, gvk)
			seenGVKs[gvk.GVK] = struct{}{}
		}
	}
	return
}

// migrateToDir rewrites a parsed project from the prior format to the new format in dir.
func (p *project) migrateToDir(dir string) error {
	if err := os.MkdirAll(dir, 0777); err != nil {
		return err
	}
	sdk := operatorSDK{}
	sdk.dir = dir

	args := []string{
		"init",
		"--plugins", string(p.layout),
		"--project-version", "3-alpha",
		"--repo", p.repo,
	}
	if p.domain != "" {
		args = append(args, "--domain", p.domain)
	}
	switch p.layout {
	case pluginKeyGo:
		sdk.env = append(sdk.env, "GO111MODULE=on")
		args = append(args, "--fetch-deps=false")
		if p.license != "" {
			args = append(args, "--license", p.license)
		}
		if p.owner != "" {
			args = append(args, "--owner", p.owner)
		}
	}
	if _, err := sdk.run(args...); err != nil {
		return err
	}

	if p.multiGroup {
		args = []string{"edit", "--multigroup"}
		if _, err := sdk.cmd.run("kubebuilder", args...); err != nil {
			return err
		}
	}

	for _, gvk := range p.gvks {
		args = []string{
			"create", "api",
			"--group", gvk.shortGroup,
			"--version", gvk.Version,
			"--kind", gvk.Kind,
		}
		switch p.layout {
		case pluginKeyGo:
			args = append(args, "--resource", "--controller")
		}
		if _, err := sdk.run(args...); err != nil {
			return err
		}
	}

	// Merge mod files.
	if p.layout == pluginKeyGo {
		newModFile, err := parseModFile(dir)
		if err != nil {
			return err
		}
		if err = mergeModFiles(newModFile, p.modFile); err != nil {
			return err
		}
		newModFile.Cleanup()
		b, err := newModFile.Format()
		if err != nil {
			return fmt.Errorf("error merging modules: %v", err)
		}
		if err := ioutil.WriteFile(filepath.Join(dir, "go.mod"), b, 0755); err != nil {
			return fmt.Errorf("error writing go.mod: %v", err)
		}
	}

	return afero.Walk(p.fs, ".", func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		b, err := afero.ReadFile(p.fs, path)
		if err != nil {
			return fmt.Errorf("error reading morphed file: %v", err)
		}
		if err = os.MkdirAll(filepath.Join(dir, filepath.Dir(path)), 0777); err != nil {
			return err
		}
		if err = ioutil.WriteFile(filepath.Join(dir, path), b, info.Mode()); err != nil {
			return fmt.Errorf("error writing migrated file: %v", err)
		}
		return nil
	})
}
