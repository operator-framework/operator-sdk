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
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"

	"sigs.k8s.io/kubebuilder/pkg/plugin"
	kbgov2 "sigs.k8s.io/kubebuilder/pkg/plugin/v2"
)

type pluginKey string

// These plugin keys use the latest imported plugin version.
// NB(estroz): these should probably be versioned since the CLI (flags) used to migrate the project may change,
// or freeze this version and have users upgrade from that version using version migration guides.
var (
	pluginKeyGo = pluginKey(plugin.KeyFor(kbgov2.Plugin{}))
)

func toPluginKey(opt string) (pluginKey, error) {
	switch opt {
	case "go":
		return pluginKeyGo, nil
	}
	return "", fmt.Errorf(`plugin type %s not supported, possible values: ["go"]`, opt)
}

func (c *migrateCmd) runWithPlugin(layout pluginKey) (err error) {
	sdkProject := &project{
		layout:  layout,
		repo:    c.repo,
		license: c.license,
		owner:   c.owner,
	}
	if sdkProject.layout == pluginKeyGo {
		if sdkProject.modFile, err = parseModFile(c.fromDir); err != nil {
			return err
		}
		if sdkProject.repo == "" {
			sdkProject.repo = sdkProject.modFile.Module.Mod.Path
		} else if err = sdkProject.modFile.AddModuleStmt(sdkProject.repo); err != nil {
			return err
		}
	}
	if sdkProject.repo == "" {
		return needsHelpErr{errors.New("--repo must be set")}
	}

	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	if c.toDir == "" {
		c.toDir, err = ioutil.TempDir(wd, path.Base(sdkProject.repo)+"-migrated-")
		if err != nil {
			return err
		}
		if c.toDir, err = filepath.Rel(wd, c.toDir); err != nil {
			return err
		}
	}

	fmt.Printf("Beginning partial migration of project %s in directory %s\n\n", c.fromDir, c.toDir)

	c.fromDir, c.toDir = filepath.Clean(c.fromDir), filepath.Clean(c.toDir)
	if err = sdkProject.parseFromDir(c.fromDir); err != nil {
		return err
	}
	if err = sdkProject.migrateToDir(c.toDir); err != nil {
		return err
	}

	// Print the correct link to a migration guide for an operator type.
	migrationDocLink := ""
	switch layout {
	case pluginKeyGo:
		migrationDocLink = migrationDocLinkGo
	}
	fmt.Printf("\nPartial migration complete. Please read the migration doc for further instructions:\n%s\n",
		migrationDocLink)

	return nil
}

type operatorSDK struct {
	cmd
}

func (c operatorSDK) run(args ...string) (string, error) { //nolint:unparam
	cmd := exec.Command("operator-sdk", args...)
	if c.dir != "" {
		cmd.Dir = c.dir
	}
	cmd.Env = append(c.env, os.Environ()...)
	fmt.Println(cmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%v: %s", err, string(out))
	}
	return string(out), nil
}

type cmd struct {
	dir string
	env []string
}

func (c cmd) run(cmdStr string, args ...string) (string, error) { //nolint:unparam
	cmd := exec.Command(cmdStr, args...)
	if c.dir != "" {
		cmd.Dir = c.dir
	}
	cmd.Env = append(c.env, os.Environ()...)
	fmt.Println(cmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%v: %s", err, string(out))
	}
	return string(out), nil
}
