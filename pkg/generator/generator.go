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

package generator

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

const (
	defaultDirFileMode  = 0750
	defaultFileMode     = 0644
	defaultExecFileMode = 0744
	// dirs
	cmdDir     = "cmd"
	deployDir  = "deploy"
	configDir  = "config"
	tmpDir     = "tmp"
	buildDir   = tmpDir + "/build"
	codegenDir = tmpDir + "/codegen"
	pkgDir     = "pkg"
	apisDir    = pkgDir + "/apis"
	stubDir    = pkgDir + "/stub"

	// files
	main            = "main.go"
	handler         = "handler.go"
	doc             = "doc.go"
	register        = "register.go"
	types           = "types.go"
	build           = "build.sh"
	dockerBuild     = "docker_build.sh"
	dockerfile      = "Dockerfile"
	boilerplate     = "boilerplate.go.txt"
	updateGenerated = "update-generated.sh"
	gopkgtoml       = "Gopkg.toml"
	gopkglock       = "Gopkg.lock"
	config          = "config.yaml"
	operatorYaml    = deployDir + "/operator.yaml"
	rbacYaml        = "rbac.yaml"
)

type Generator struct {
	// apiVersion is the kubernetes apiVersion that has the format of $GROUP_NAME/$VERSION.
	apiVersion string
	kind       string
	// projectName is name of the new operator application
	// and is also the name of the base directory.
	projectName string
	// repoPath is the project's repository path rooted under $GOPATH.
	repoPath string
}

// NewGenerator creates a new scaffold Generator.
func NewGenerator(apiVersion, kind, projectName, repoPath string) *Generator {
	return &Generator{apiVersion: apiVersion, kind: kind, projectName: projectName, repoPath: repoPath}
}

// Render generates the default project structure:
//
// ├── <projectName>
// │   ├── cmd
// │   │   └── <projectName>
// │   ├── config
// │   ├── deploy
// │   ├── pkg
// │   │   ├── apis
// │   │   │   └── <api-dir-name>  // computed from apiDirName(apiVersion).
// │   │   │       └── <version> // computed from version(apiVersion).
// │   │   └── stub
// │   └── tmp
// │       ├── build
// │       └── codegen
func (g *Generator) Render() error {
	if err := g.renderCmd(); err != nil {
		return err
	}
	if err := g.renderConfig(); err != nil {
		return err
	}
	if err := g.renderDeploy(); err != nil {
		return err
	}
	if err := g.renderPkg(); err != nil {
		return err
	}
	if err := g.renderTmp(); err != nil {
		return err
	}
	return g.renderGoDep()
}

func (g *Generator) renderGoDep() error {
	buf := &bytes.Buffer{}
	if err := renderGopkgTomlFile(buf); err != nil {
		return err
	}
	if err := ioutil.WriteFile(filepath.Join(g.projectName, gopkgtoml), buf.Bytes(), defaultFileMode); err != nil {
		return err
	}

	buf = &bytes.Buffer{}
	if err := renderGopkgLockFile(buf); err != nil {
		return err
	}
	return ioutil.WriteFile(filepath.Join(g.projectName, gopkglock), buf.Bytes(), defaultFileMode)
}

func (g *Generator) renderCmd() error {
	cpDir := filepath.Join(g.projectName, cmdDir, g.projectName)
	if err := os.MkdirAll(cpDir, defaultDirFileMode); err != nil {
		return err
	}
	return renderCmdFiles(cpDir, g.repoPath)
}

func renderCmdFiles(cmdProjectDir, repoPath string) error {
	buf := &bytes.Buffer{}
	if err := renderMainFile(buf, repoPath); err != nil {
		return err
	}
	return ioutil.WriteFile(filepath.Join(cmdProjectDir, main), buf.Bytes(), defaultFileMode)
}

func (g *Generator) renderConfig() error {
	cp := filepath.Join(g.projectName, configDir)
	if err := os.MkdirAll(cp, defaultDirFileMode); err != nil {
		return err
	}
	return renderConfigFiles(cp, g.apiVersion, g.kind, g.projectName)
}

func renderConfigFiles(configDir, apiVersion, kind, projectName string) error {
	buf := &bytes.Buffer{}
	if err := renderConfigFile(buf, apiVersion, kind, projectName); err != nil {
		return err
	}
	return ioutil.WriteFile(filepath.Join(configDir, config), buf.Bytes(), defaultFileMode)
}

func (g *Generator) renderDeploy() error {
	dp := filepath.Join(g.projectName, deployDir)
	if err := os.MkdirAll(dp, defaultDirFileMode); err != nil {
		return err
	}
	return renderRBAC(dp, g.projectName)
}

func renderRBAC(deployDir, projectName string) error {
	buf := &bytes.Buffer{}
	if err := renderRBACYaml(buf, projectName); err != nil {
		return err
	}
	return ioutil.WriteFile(filepath.Join(deployDir, rbacYaml), buf.Bytes(), defaultFileMode)
}

// RenderDeployFiles generates "deploy/operator.yaml"
// TODO: rethink about when/what to generate when invoking build command.
func RenderDeployFiles(c *Config, image string) error {
	buf := &bytes.Buffer{}
	if err := renderOperatorYaml(buf, c.Kind, c.APIVersion, c.ProjectName, image); err != nil {
		return err
	}
	return ioutil.WriteFile(operatorYaml, buf.Bytes(), defaultFileMode)
}

func (g *Generator) renderTmp() error {
	bDir := filepath.Join(g.projectName, buildDir)
	if err := os.MkdirAll(bDir, defaultDirFileMode); err != nil {
		return err
	}
	if err := renderBuildFiles(bDir, g.repoPath, g.projectName); err != nil {
		return err
	}

	cDir := filepath.Join(g.projectName, codegenDir)
	if err := os.MkdirAll(cDir, defaultDirFileMode); err != nil {
		return err
	}
	return renderCodegenFiles(cDir, g.repoPath, apiDirName(g.apiVersion), version(g.apiVersion), g.projectName)
}

func renderBuildFiles(buildDir, repoPath, projectName string) error {
	buf := &bytes.Buffer{}
	if err := renderBuildFile(buf, repoPath, projectName); err != nil {
		return err
	}
	if err := ioutil.WriteFile(filepath.Join(buildDir, build), buf.Bytes(), defaultExecFileMode); err != nil {
		return err
	}

	buf = &bytes.Buffer{}
	if err := renderDockerBuildFile(buf); err != nil {
		return err
	}
	if err := ioutil.WriteFile(filepath.Join(buildDir, dockerBuild), buf.Bytes(), defaultExecFileMode); err != nil {
		return err
	}

	buf = &bytes.Buffer{}
	if err := renderDockerFile(buf, projectName); err != nil {
		return err
	}
	return ioutil.WriteFile(filepath.Join(buildDir, dockerfile), buf.Bytes(), defaultFileMode)
}

func renderCodegenFiles(codegenDir, repoPath, apiDirName, version, projectName string) error {
	buf := &bytes.Buffer{}
	if err := renderBoilerplateFile(buf, projectName); err != nil {
		return err
	}
	if err := ioutil.WriteFile(filepath.Join(codegenDir, boilerplate), buf.Bytes(), defaultFileMode); err != nil {
		return err
	}

	buf = &bytes.Buffer{}
	if err := renderUpdateGeneratedFile(buf, repoPath, apiDirName, version); err != nil {
		return err
	}
	return ioutil.WriteFile(filepath.Join(codegenDir, updateGenerated), buf.Bytes(), defaultExecFileMode)
}

func (g *Generator) renderPkg() error {
	v := version(g.apiVersion)
	apiDir := filepath.Join(g.projectName, apisDir, apiDirName(g.apiVersion), v)
	if err := os.MkdirAll(apiDir, defaultDirFileMode); err != nil {
		return err
	}
	if err := renderAPIFiles(apiDir, groupName(g.apiVersion), v, g.kind); err != nil {
		return err
	}

	sDir := filepath.Join(g.projectName, stubDir)
	if err := os.MkdirAll(sDir, defaultDirFileMode); err != nil {
		return err
	}
	return renderStubFiles(sDir)
}

func renderAPIFiles(apiDir, groupName, version, kind string) error {
	buf := &bytes.Buffer{}
	if err := renderAPIDocFile(buf, groupName, version); err != nil {
		return err
	}
	if err := ioutil.WriteFile(filepath.Join(apiDir, doc), buf.Bytes(), defaultFileMode); err != nil {
		return err
	}

	buf = &bytes.Buffer{}
	if err := renderAPIRegisterFile(buf, kind, groupName, version); err != nil {
		return err
	}
	if err := ioutil.WriteFile(filepath.Join(apiDir, register), buf.Bytes(), defaultFileMode); err != nil {
		return err
	}

	buf = &bytes.Buffer{}
	if err := renderAPITypesFile(buf, kind, version); err != nil {
		return err
	}
	return ioutil.WriteFile(filepath.Join(apiDir, types), buf.Bytes(), defaultFileMode)
}

func renderStubFiles(stubDir string) error {
	buf := &bytes.Buffer{}
	if err := renderHandlerFile(buf); err != nil {
		return err
	}
	return ioutil.WriteFile(filepath.Join(stubDir, handler), buf.Bytes(), defaultFileMode)
}

// version extracts the VERSION from the given apiVersion ($GROUP_NAME/$VERSION).
func version(apiVersion string) string {
	return strings.Split(apiVersion, "/")[1]
}

// groupName extracts the GROUP_NAME from the given apiVersion ($GROUP_NAME/$VERSION).
func groupName(apiVersion string) string {
	return strings.Split(apiVersion, "/")[0]
}

// apiDirName extracts the name of api directory under ../apis/ folder
// from the given apiVersion ($GROUP_NAME/$VERSION).
// the first word separated with "." of the GROUP_NAME is the api directory name.
// for example,
//  apiDirName("app.example.com/v1alpha1") => "app".
func apiDirName(apiVersion string) string {
	return strings.Split(groupName(apiVersion), ".")[0]
}
