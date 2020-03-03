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

package run

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

	"github.com/operator-framework/operator-sdk/internal/scaffold"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"
	"github.com/operator-framework/operator-sdk/pkg/ansible"
	aoflags "github.com/operator-framework/operator-sdk/pkg/ansible/flags"
	"github.com/operator-framework/operator-sdk/pkg/helm"
	hoflags "github.com/operator-framework/operator-sdk/pkg/helm/flags"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"github.com/operator-framework/operator-sdk/pkg/log/zap"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

type runLocalArgs struct {
	kubeconfig           string
	watchNamespace       string
	operatorFlags        string
	ldFlags              string
	enableDelve          bool
	ansibleOperatorFlags *aoflags.AnsibleOperatorFlags
	helmOperatorFlags    *hoflags.HelmOperatorFlags
}

func (c *runLocalArgs) addToFlags(fs *pflag.FlagSet) {
	prefix := "[local only] "
	fs.StringVar(&c.watchNamespace, "watch-namespace", "",
		prefix+"The namespace where the operator watches for changes.")
	fs.StringVar(&c.operatorFlags, "operator-flags", "",
		prefix+"The flags that the operator needs. Example: \"--flag1 value1 --flag2=value2\"")
	fs.StringVar(&c.ldFlags, "go-ldflags", "", prefix+"Set Go linker options")
	fs.BoolVar(&c.enableDelve, "enable-delve", false,
		prefix+"Start the operator using the delve debugger")
}

func (c runLocalArgs) run() error {
	log.Infof("Running the operator locally in namespace %s.", c.watchNamespace)

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

func (c runLocalArgs) runGo() error {
	projutil.MustInProjectRoot()
	absProjectPath := projutil.MustGetwd()
	projectName := filepath.Base(absProjectPath)
	outputBinName := filepath.Join(scaffold.BuildBinDir, projectName+"-local")
	if runtime.GOOS == "windows" {
		outputBinName += ".exe"
	}
	if err := c.buildLocal(outputBinName); err != nil {
		return fmt.Errorf("failed to build operator to run locally: %v", err)
	}

	args := []string{}
	if c.operatorFlags != "" {
		extraArgs := strings.Split(c.operatorFlags, " ")
		args = append(args, extraArgs...)
	}

	var dc *exec.Cmd

	if c.enableDelve {
		delveArgs := []string{"--listen=:2345", "--headless=true", "--api-version=2", "exec", outputBinName, "--"}
		delveArgs = append(delveArgs, args...)

		dc = exec.Command("dlv", delveArgs...)
		log.Infof("Delve debugger enabled with args %s", delveArgs)
	} else {
		dc = exec.Command(outputBinName, args...)
	}

	ch := make(chan os.Signal)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-ch
		err := dc.Process.Kill()
		if err != nil {
			log.Fatalf("Failed to terminate the operator: (%v)", err)
		}
		os.Exit(0)
	}()
	dc.Env = os.Environ()
	dc.Env = append(dc.Env, fmt.Sprintf("%s=%s", k8sutil.ForceRunModeEnv, k8sutil.LocalRunMode))
	// only set env var if user explicitly specified a kubeconfig path
	if c.kubeconfig != "" {
		dc.Env = append(dc.Env, fmt.Sprintf("%v=%v", k8sutil.KubeConfigEnvVar, c.kubeconfig))
	}
	dc.Env = append(dc.Env, fmt.Sprintf("%v=%v", k8sutil.WatchNamespaceEnvVar, c.watchNamespace))

	// Set the ANSIBLE_ROLES_PATH
	if c.ansibleOperatorFlags != nil && len(c.ansibleOperatorFlags.AnsibleRolesPath) > 0 {
		log.Info(fmt.Sprintf("set the value %v for environment variable %v.", c.ansibleOperatorFlags.AnsibleRolesPath,
			aoflags.AnsibleRolesPathEnvVar))
		dc.Env = append(dc.Env, fmt.Sprintf("%v=%v", aoflags.AnsibleRolesPathEnvVar, c.ansibleOperatorFlags.AnsibleRolesPath))
	}

	if err := projutil.ExecCmd(dc); err != nil {
		return fmt.Errorf("failed to run operator locally: %v", err)
	}

	return nil
}

func (c runLocalArgs) runAnsible() error {
	logf.SetLogger(zap.Logger())
	if err := setupOperatorEnv(c.kubeconfig, c.watchNamespace); err != nil {
		return err
	}
	return ansible.Run(c.ansibleOperatorFlags)
}

func (c runLocalArgs) runHelm() error {
	logf.SetLogger(zap.Logger())
	if err := setupOperatorEnv(c.kubeconfig, c.watchNamespace); err != nil {
		return err
	}
	return helm.Run(c.helmOperatorFlags)
}

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
	// Set the operator name, if not already set
	projutil.MustInProjectRoot()
	if _, err := k8sutil.GetOperatorName(); err != nil {
		operatorName := filepath.Base(projutil.MustGetwd())
		if err := os.Setenv(k8sutil.OperatorNameEnvVar, operatorName); err != nil {
			return fmt.Errorf("failed to set %s environment variable: %v", k8sutil.OperatorNameEnvVar, err)
		}
	}
	return nil
}

func (c runLocalArgs) buildLocal(outputBinName string) error {
	var args []string
	if c.ldFlags != "" {
		args = []string{"-ldflags", c.ldFlags}
	}
	if c.enableDelve {
		args = append(args, "-gcflags=\"all=-N -l\"")
	}
	opts := projutil.GoCmdOptions{
		BinName:     outputBinName,
		PackagePath: path.Join(projutil.GetGoPkg(), filepath.ToSlash(scaffold.ManagerDir)),
		Args:        args,
	}
	return projutil.GoBuild(opts)
}
