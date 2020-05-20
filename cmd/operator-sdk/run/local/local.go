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

package local

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/operator-framework/operator-sdk/internal/scaffold"
	kbutil "github.com/operator-framework/operator-sdk/internal/util/kubebuilder"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"
	"github.com/operator-framework/operator-sdk/pkg/ansible"
	aoflags "github.com/operator-framework/operator-sdk/pkg/ansible/flags"
	"github.com/operator-framework/operator-sdk/pkg/helm"
	hoflags "github.com/operator-framework/operator-sdk/pkg/helm/flags"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"github.com/operator-framework/operator-sdk/pkg/log/zap"
)

type RunLocalCmd struct {
	Kubeconfig           string
	WatchNamespace       string
	OperatorFlags        string
	LDFlags              string
	EnableDelve          bool
	AnsibleOperatorFlags *aoflags.AnsibleOperatorFlags
	HelmOperatorFlags    *hoflags.HelmOperatorFlags
}

func (c *RunLocalCmd) AddToFlags(fs *pflag.FlagSet) {
	// The main.go and manager.yaml scaffolds in the new layout do not support the WATCH_NAMESPACE
	// env var to configure the namespace that the operator watches. The default is all namespaces.
	// So this flag is unsupported for the new layout.
	if !kbutil.HasProjectFile() {
		fs.StringVar(&c.WatchNamespace, "watch-namespace", "",
			"The namespace where the operator watches for changes. Set \"\" for AllNamespaces, "+
				"set \"ns1,ns2\" for MultiNamespace")
	}

	fs.StringVar(&c.Kubeconfig, "kubeconfig", "",
		"The file path to kubernetes configuration file. Defaults to location "+
			"specified by $KUBECONFIG, or to default file rules if not set")
	fs.StringVar(&c.OperatorFlags, "operator-flags", "",
		"The flags that the operator needs. Example: \"--flag1 value1 --flag2=value2\"")
	fs.StringVar(&c.LDFlags, "go-ldflags", "", "Set Go linker options")
	fs.BoolVar(&c.EnableDelve, "enable-delve", false,
		"Start the operator using the delve debugger")
}

func (c RunLocalCmd) Run() error {
	// The new layout will not have c.WatchNamespace
	if kbutil.HasProjectFile() {
		log.Infof("Running the operator locally ...")
	} else {
		log.Infof("Running the operator locally; watching namespace %q", c.WatchNamespace)
	}
	switch t := projutil.GetOperatorType(); t {
	case projutil.OperatorTypeGo:
		return c.runGo()
	case projutil.OperatorTypeAnsible:
		return c.runAnsible()
	case projutil.OperatorTypeHelm:
		return c.runHelm()
	}
	return projutil.ErrUnknownOperatorType{}
}

// runGo will run the project locally for the Go Type projects
func (c RunLocalCmd) runGo() error {
	// Build the project and generate binary that will be executed
	binName, err := c.generateBinary()
	if err != nil {
		return err
	}
	// Get the args that will be used to exec the binary.
	// Users are allowed to use the flag operator-flags to pass any value that they may wish
	args := c.argsFromOperatorFlags()
	// Build the command
	var cmd *exec.Cmd
	if c.EnableDelve {
		cmd = getExecCmdWithDebugger(binName, args)
	} else {
		cmd = exec.Command(binName, args...)
	}
	// Kill the command if an exit signal is received.
	ch := make(chan os.Signal)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-ch
		err := cmd.Process.Kill()
		if err != nil {
			log.Fatalf("Failed to terminate the operator: (%v)", err)
		}
		os.Exit(0)
	}()
	// Add default env vars and values informed via flags
	c.addEnvVars(cmd)
	if err := projutil.ExecCmd(cmd); err != nil {
		return fmt.Errorf("failed to run operator locally: %v", err)
	}
	return nil
}

func (c RunLocalCmd) runAnsible() error {
	logf.SetLogger(zap.Logger())
	if err := setupOperatorEnv(c.Kubeconfig, c.WatchNamespace); err != nil {
		return err
	}
	return ansible.Run(c.AnsibleOperatorFlags)
}

func (c RunLocalCmd) runHelm() error {
	logf.SetLogger(zap.Logger())
	if err := setupOperatorEnv(c.Kubeconfig, c.WatchNamespace); err != nil {
		return err
	}
	return helm.Run(c.HelmOperatorFlags)
}

// setupOperatorEnv will add envvar for the kubeconfig and namespace informed
func setupOperatorEnv(kubeconfig, namespace string) error {
	// Set the kubeconfig that the manager will be able to grab
	// only set env var if user explicitly specified a kubeconfig path
	if kubeconfig != "" {
		if err := os.Setenv(k8sutil.KubeConfigEnvVar, kubeconfig); err != nil {
			return fmt.Errorf("failed to set %s environment variable: %v", k8sutil.KubeConfigEnvVar, err)
		}
	}
	// Set the namespace that the manager will be able to grab
	if namespace != "" {
		if err := os.Setenv(k8sutil.WatchNamespaceEnvVar, namespace); err != nil {
			return fmt.Errorf("failed to set %s environment variable: %v", k8sutil.WatchNamespaceEnvVar, err)
		}
	}
	if _, err := k8sutil.GetOperatorName(); err != nil {
		operatorName := filepath.Base(projutil.MustGetwd())
		if err := os.Setenv(k8sutil.OperatorNameEnvVar, operatorName); err != nil {
			return fmt.Errorf("failed to set %s environment variable: %v", k8sutil.OperatorNameEnvVar, err)
		}
	}
	return nil
}

// getBuildRunLocalArgs returns go build args for -ldflags and -gcflags
func (c RunLocalCmd) getBuildRunLocalArgs() []string {
	var args []string
	if c.LDFlags != "" {
		args = []string{"-ldflags", c.LDFlags}
	}
	if c.EnableDelve {
		args = append(args, "-gcflags=\"all=-N -l\"")
	}
	return args
}

// generateBinary will build the Go project by running the command `go build -o bin/manager main.go`
func (c RunLocalCmd) generateBinary() (string, error) {
	// Define name of the bin and where is the main.go pkg for each layout
	var outputBinName, packagePath string
	if kbutil.HasProjectFile() {
		outputBinName = filepath.Join(kbutil.BinBuildDir, getProjectName()+"-local")
		packagePath = projutil.GetGoPkg()
	} else {
		// todo: remove the if, else when the legacy code is no longer supported
		packagePath = path.Join(projutil.GetGoPkg(), filepath.ToSlash(scaffold.ManagerDir))
		outputBinName = filepath.Join(scaffold.BuildBinDir, getProjectName()+"-local")
	}
	// allow the command works in windows SO
	if runtime.GOOS == "windows" {
		outputBinName += ".exe"
	}
	opts := projutil.GoCmdOptions{
		BinName:     outputBinName,
		PackagePath: packagePath,
		Args:        c.getBuildRunLocalArgs(),
	}
	if err := projutil.GoBuild(opts); err != nil {
		return "", err
	}
	return outputBinName, nil
}

// execCmdForBinWithDebugger will exec the command with the delve required args
// Note that delve is a debugger for the Go programming language.
// More info: https://github.com/go-delve/delve
func getExecCmdWithDebugger(binName string, args []string) *exec.Cmd {
	delveArgs := []string{"--listen=:2345", "--headless=true", "--api-version=2", "exec", binName, "--"}
	delveArgs = append(delveArgs, args...)
	log.Infof("Delve debugger enabled with args %s", delveArgs)
	return exec.Command("dlv", delveArgs...)
}

// addEnvVars will add the EnvVars to the command informed
func (c RunLocalCmd) addEnvVars(cmd *exec.Cmd) {
	cmd.Env = os.Environ()

	// Set EnvVar to let the project knows that it is running outside of the cluster
	cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k8sutil.ForceRunModeEnv, k8sutil.LocalRunMode))

	// Only set env var if user explicitly specified a kubeconfig path
	if c.Kubeconfig != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%v=%v", k8sutil.KubeConfigEnvVar, c.Kubeconfig))
	}

	// Set WATCH_NAMESPACE with the value informed via flag
	cmd.Env = append(cmd.Env, fmt.Sprintf("%v=%v", k8sutil.WatchNamespaceEnvVar, c.WatchNamespace))

	// todo: check if it should really be here. Shows that it should be part of AnsibleRun only.
	// Set the ANSIBLE_ROLES_PATH
	if c.AnsibleOperatorFlags != nil && len(c.AnsibleOperatorFlags.AnsibleRolesPath) > 0 {
		log.Info(fmt.Sprintf("set the value %v for environment variable %v.",
			c.AnsibleOperatorFlags.AnsibleRolesPath, aoflags.AnsibleRolesPathEnvVar))
		cmd.Env = append(cmd.Env, fmt.Sprintf("%v=%v", aoflags.AnsibleRolesPathEnvVar,
			c.AnsibleOperatorFlags.AnsibleRolesPath))
	}
}

// argsFromOperatorFlags will return an array with all args used in the flags
func (c RunLocalCmd) argsFromOperatorFlags() []string {
	args := []string{}
	if c.OperatorFlags != "" {
		extraArgs := strings.Split(c.OperatorFlags, " ")
		args = append(args, extraArgs...)
	}
	return args
}

// getProjectName will return the name of the project. This function only works if the current working directory
// is the project root.
func getProjectName() string {
	absProjectPath := projutil.MustGetwd()
	return filepath.Base(absProjectPath)
}
