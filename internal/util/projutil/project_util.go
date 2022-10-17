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
	"os"
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"sigs.k8s.io/kubebuilder/v3/pkg/config"
	yamlstore "sigs.k8s.io/kubebuilder/v3/pkg/config/store/yaml"
	_ "sigs.k8s.io/kubebuilder/v3/pkg/config/v2" // Register config/v2 for `config.New`
	_ "sigs.k8s.io/kubebuilder/v3/pkg/config/v3" // Register config/v3 for `config.New`
	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
)

const (
	// Useful file modes.
	DirMode      = 0755
	FileMode     = 0644
	ExecFileMode = 0755
)

const (
	// Go env vars.
	GoFlagsEnv = "GOFLAGS"
)

// Default config file path.
const configFile = "PROJECT"

// OperatorType - the type of operator
type OperatorType = string

const (
	// OperatorTypeGo - golang type of operator.
	OperatorTypeGo OperatorType = "go"
	// OperatorTypeAnsible - ansible type of operator.
	OperatorTypeAnsible OperatorType = "ansible"
	// OperatorTypeHelm - helm type of operator.
	OperatorTypeHelm OperatorType = "helm"
	// OperatorTypeHybrid - hybrid type of operator.
	operatorTypeHybridHelm OperatorType = "hybridHelm"
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

// HasProjectFile returns true if the project is configured as a kubebuilder
// project.
func HasProjectFile() bool {
	_, err := os.Stat(configFile)
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}
		log.Fatalf("Failed to read PROJECT file to detect kubebuilder project: %v", err)
	}
	return true
}

// ReadConfig returns a configuration if a file containing one exists at the
// default path (project root).
func ReadConfig() (config.Config, error) {
	store := yamlstore.New(machinery.Filesystem{FS: afero.NewOsFs()})
	if err := store.Load(); err != nil {
		return nil, err
	}

	return store.Config(), nil
}

// PluginChainToOperatorType converts a plugin chain to an operator project type.
// TODO(estroz): this can probably be made more robust by checking known plugin keys directly.
func PluginChainToOperatorType(pluginKeys []string) OperatorType {
	for _, pluginKey := range pluginKeys {
		switch {
		case strings.HasPrefix(pluginKey, "go"):
			return OperatorTypeGo
		case strings.HasPrefix(pluginKey, "helm"):
			return OperatorTypeHelm
		case strings.HasPrefix(pluginKey, "ansible"):
			return OperatorTypeAnsible
		case strings.HasPrefix(pluginKey, "hybrid"):
			return operatorTypeHybridHelm
		}
	}
	return OperatorTypeUnknown
}

// GetProjectLayout returns the `layout` field as a comma separated list.
func GetProjectLayout(cfg config.Config) string {
	return strings.Join(cfg.GetPluginChain(), ",")
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

// RewriteFileContents adds newContent to the line after the last occurrence of target in filename's contents,
// then writes the updated contents back to disk.
func RewriteFileContents(filename, target, newContent string) error {
	text, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("error in getting contents from the file, %v", err)
	}

	modifiedContent, err := appendContent(string(text), target, newContent)
	if err != nil {
		return err
	}

	err = os.WriteFile(filename, []byte(modifiedContent), FileMode)
	if err != nil {
		return fmt.Errorf("error writing modified contents to file, %v", err)
	}

	return nil
}

func appendContent(fileContents, target, newContent string) (string, error) {
	labelIndex := strings.LastIndex(fileContents, target)
	if labelIndex == -1 {
		return "", fmt.Errorf("no prior string %s in newContent", target)
	}

	separationIndex := strings.Index(fileContents[labelIndex:], "\n")
	if separationIndex == -1 {
		return "", fmt.Errorf("no new line at the end of string %s", fileContents[labelIndex:])
	}

	index := labelIndex + separationIndex + 1
	return fileContents[:index] + newContent + fileContents[index:], nil
}
