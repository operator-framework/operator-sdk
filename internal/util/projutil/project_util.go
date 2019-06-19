// Copyright 2018 The Operator-SDK Authors
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

package projutil

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"github.com/rogpeppe/go-internal/modfile"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const (
	GoPathEnv  = "GOPATH"
	GoFlagsEnv = "GOFLAGS"
	GoModEnv   = "GO111MODULE"
	SrcDir     = "src"

	fsep            = string(filepath.Separator)
	mainFile        = "cmd" + fsep + "manager" + fsep + "main.go"
	buildDockerfile = "build" + fsep + "Dockerfile"
	rolesDir        = "roles"
	helmChartsDir   = "helm-charts"
	goModFile       = "go.mod"
	gopkgTOMLFile   = "Gopkg.toml"
)

// OperatorType - the type of operator
type OperatorType = string

const (
	// OperatorTypeGo - golang type of operator.
	OperatorTypeGo OperatorType = "go"
	// OperatorTypeAnsible - ansible type of operator.
	OperatorTypeAnsible OperatorType = "ansible"
	// OperatorTypeHelm - helm type of operator.
	OperatorTypeHelm OperatorType = "helm"
	// OperatorTypeUnknown - unknown type of operator.
	OperatorTypeUnknown OperatorType = "unknown"
)

type ErrUnknownOperatorType struct {
	Type string
}

func (e ErrUnknownOperatorType) Error() string {
	if e.Type == "" {
		return "unknown operator type"
	}
	return fmt.Sprintf(`unknown operator type "%v"`, e.Type)
}

type DepManagerType string

const (
	DepManagerGoMod DepManagerType = "modules"
	DepManagerDep   DepManagerType = "dep"
)

type ErrInvalidDepManager string

func (e ErrInvalidDepManager) Error() string {
	return fmt.Sprintf(`"%s" is not a valid dep manager; dep manager must be one of ["%v", "%v"]`, string(e), DepManagerDep, DepManagerGoMod)
}

var ErrNoDepManager = fmt.Errorf(`no valid dependency manager file found; dep manager must be one of ["%v", "%v"]`, DepManagerDep, DepManagerGoMod)

func GetDepManagerType() (DepManagerType, error) {
	if IsDepManagerDep() {
		return DepManagerDep, nil
	} else if IsDepManagerGoMod() {
		return DepManagerGoMod, nil
	}
	return "", ErrNoDepManager
}

func IsDepManagerDep() bool {
	_, err := os.Stat(gopkgTOMLFile)
	return err == nil || os.IsExist(err)
}

func IsDepManagerGoMod() bool {
	_, err := os.Stat(goModFile)
	return err == nil || os.IsExist(err)
}

// MustInProjectRoot checks if the current dir is the project root, and exits
// if not.
func MustInProjectRoot() {
	if err := CheckProjectRoot(); err != nil {
		log.Fatal(err)
	}
}

// CheckProjectRoot checks if the current dir is the project root, and returns
// an error if not.
func CheckProjectRoot() error {
	// If the current directory has a "build/Dockerfile", then it is safe to say
	// we are at the project root.
	if _, err := os.Stat(buildDockerfile); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("must run command in project root dir: project structure requires %s", buildDockerfile)
		}
		return errors.Wrap(err, "error while checking if current directory is the project root")
	}
	return nil
}

func CheckGoProjectCmd(cmd *cobra.Command) error {
	if IsOperatorGo() {
		return nil
	}
	return fmt.Errorf("'%s' can only be run for Go operators; %s does not exist", cmd.CommandPath(), mainFile)
}

func MustGetwd() string {
	wd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get working directory: (%v)", err)
	}
	return wd
}

func getHomeDir() (string, error) {
	hd, err := homedir.Dir()
	if err != nil {
		return "", err
	}
	return homedir.Expand(hd)
}

// GetGoPkg returns the current directory's import path by parsing it from
// wd if this project's repository path is rooted under $GOPATH/src, or
// from go.mod the project uses Go modules to manage dependencies.
//
// Example: "github.com/example-inc/app-operator"
func GetGoPkg() string {
	// Default to reading from go.mod, as it should usually have the (correct)
	// package path, and no further processing need be done on it if so.
	if _, err := os.Stat(goModFile); err != nil && !os.IsNotExist(err) {
		log.Fatalf("Failed to read go.mod: %v", err)
	} else if err == nil {
		b, err := ioutil.ReadFile(goModFile)
		if err != nil {
			log.Fatalf("Read go.mod: %v", err)
		}
		mf, err := modfile.Parse(goModFile, b, nil)
		if err != nil {
			log.Fatalf("Parse go.mod: %v", err)
		}
		if mf.Module != nil && mf.Module.Mod.Path != "" {
			return mf.Module.Mod.Path
		}
	}

	// Then try parsing package path from $GOPATH (set env or default).
	goPath, ok := os.LookupEnv(GoPathEnv)
	if !ok || goPath == "" {
		hd, err := getHomeDir()
		if err != nil {
			log.Fatal(err)
		}
		goPath = filepath.Join(hd, "go", "src")
	} else {
		// MustSetWdGopath is necessary here because the user has set GOPATH,
		// which could be a path list.
		goPath = MustSetWdGopath(goPath)
	}
	if !strings.HasPrefix(MustGetwd(), goPath) {
		log.Fatal("Could not determine project repository path: $GOPATH not set, wd in default $HOME/go/src, or wd does not contain a go.mod")
	}
	return parseGoPkg(goPath)
}

func parseGoPkg(gopath string) string {
	goSrc := filepath.Join(gopath, SrcDir)
	wd := MustGetwd()
	pathedPkg := strings.Replace(wd, goSrc, "", 1)
	// Make sure package only contains the "/" separator and no others, and
	// trim any leading/trailing "/".
	return strings.Trim(filepath.ToSlash(pathedPkg), "/")
}

// GetOperatorType returns type of operator is in cwd.
// This function should be called after verifying the user is in project root.
func GetOperatorType() OperatorType {
	switch {
	case IsOperatorGo():
		return OperatorTypeGo
	case IsOperatorAnsible():
		return OperatorTypeAnsible
	case IsOperatorHelm():
		return OperatorTypeHelm
	}
	return OperatorTypeUnknown
}

func IsOperatorGo() bool {
	_, err := os.Stat(mainFile)
	return err == nil
}

func IsOperatorAnsible() bool {
	stat, err := os.Stat(rolesDir)
	return err == nil && stat.IsDir()
}

func IsOperatorHelm() bool {
	stat, err := os.Stat(helmChartsDir)
	return err == nil && stat.IsDir()
}

// MustGetGopath gets GOPATH and ensures it is set and non-empty. If GOPATH
// is not set or empty, MustGetGopath exits.
func MustGetGopath() string {
	gopath, ok := os.LookupEnv(GoPathEnv)
	if !ok || len(gopath) == 0 {
		log.Fatal("GOPATH env not set")
	}
	return gopath
}

// MustSetWdGopath sets GOPATH to the first element of the path list in
// currentGopath that prefixes the wd, then returns the set path.
// If GOPATH cannot be set, MustSetWdGopath exits.
func MustSetWdGopath(currentGopath string) string {
	var (
		newGopath   string
		cwdInGopath bool
		wd          = MustGetwd()
	)
	for _, newGopath = range filepath.SplitList(currentGopath) {
		if strings.HasPrefix(filepath.Dir(wd), newGopath) {
			cwdInGopath = true
			break
		}
	}
	if !cwdInGopath {
		log.Fatalf("Project not in $GOPATH")
	}
	if err := os.Setenv(GoPathEnv, newGopath); err != nil {
		log.Fatal(err)
	}
	return newGopath
}

var flagRe = regexp.MustCompile("(.* )?-v(.* )?")

// SetGoVerbose sets GOFLAGS="${GOFLAGS} -v" if GOFLAGS does not
// already contain "-v" to make "go" command output verbose.
func SetGoVerbose() error {
	gf, ok := os.LookupEnv(GoFlagsEnv)
	if !ok || len(gf) == 0 {
		return os.Setenv(GoFlagsEnv, "-v")
	}
	if !flagRe.MatchString(gf) {
		return os.Setenv(GoFlagsEnv, gf+" -v")
	}
	return nil
}

// CheckDepManagerWithRepo ensures dependency manager type and repo are being used in combination
// correctly, as different dependency managers have different Go environment
// requirements.
func CheckDepManagerWithRepo(dm DepManagerType, repo string) error {
	inGopathSrc, err := WdInGoPathSrc()
	if err != nil {
		return err
	}
	switch dm {
	case DepManagerDep:
		// dep assumes the project's path under $GOPATH/src is the project's
		// repo path.
		if repo != "" {
			return fmt.Errorf(`The flag --repo cannot be set with dependency manager "dep", as dep always infers the repo path`)
		}
		if !inGopathSrc {
			return fmt.Errorf(`dependency manager "dep" requires working directory to be in $GOPATH/src`)
		}
	case DepManagerGoMod:
		if !inGopathSrc && repo == "" {
			return fmt.Errorf(`dependency manager "modules" requires the flag --repo to be set if the working directory is not in $GOPATH/src. See "operator-sdk new -h"`)
		}
	default:
		return ErrInvalidDepManager(dm)
	}

	return nil
}
