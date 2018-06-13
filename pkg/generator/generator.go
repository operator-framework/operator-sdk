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
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

const (
	defaultDirFileMode  = 0750
	defaultFileMode     = 0644
	defaultExecFileMode = 0744
	// dirs
	cmdDir        = "cmd"
	deployDir     = "deploy"
	olmCatalogDir = deployDir + "/olm-catalog"
	configDir     = "config"
	tmpDir        = "tmp"
	buildDir      = tmpDir + "/build"
	codegenDir    = tmpDir + "/codegen"
	pkgDir        = "pkg"
	apisDir       = pkgDir + "/apis"
	stubDir       = pkgDir + "/stub"
	versionDir    = "version"

	// files
	main               = "main.go"
	handler            = "handler.go"
	doc                = "doc.go"
	register           = "register.go"
	types              = "types.go"
	build              = "build.sh"
	dockerBuild        = "docker_build.sh"
	dockerfile         = "Dockerfile"
	boilerplate        = "boilerplate.go.txt"
	updateGenerated    = "update-generated.sh"
	gopkgtoml          = "Gopkg.toml"
	gopkglock          = "Gopkg.lock"
	config             = "config.yaml"
	operatorYaml       = deployDir + "/operator.yaml"
	rbacYaml           = "rbac.yaml"
	crYaml             = "cr.yaml"
	catalogPackageYaml = "package.yaml"
	catalogCSVYaml     = "csv.yaml"
	crdYaml            = "crd.yaml"
	gitignore          = ".gitignore"
	versionfile        = "version.go"

	// sdkImport is the operator-sdk import path.
	sdkImport          = "github.com/operator-framework/operator-sdk/pkg/sdk"
	k8sutilImport      = "github.com/operator-framework/operator-sdk/pkg/util/k8sutil"
	versionImport      = "github.com/operator-framework/operator-sdk/version"
	packageChannel     = "alpha"
	catalogCRDTmplName = "deploy/olm-catalog/crd.yaml"
	crdTmplName        = "deploy/crd.yaml"
	operatorTmplName   = "deploy/operator.yaml"
	rbacTmplName       = "deploy/rbac.yaml"
	crTmplName         = "deploy/cr.yaml"
	pluralSuffix       = "s"
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
// │   ├── tmp
// │   |   ├── build
// │   |   └── codegen
// │   └── version
func (g *Generator) Render() error {
	if err := g.generateDirStructure(); err != nil {
		return err
	}

	if err := g.renderProject(); err != nil {
		return err
	}

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
	if err := g.renderVersion(); err != nil {
		return err
	}
	return g.renderGoDep()
}

func (g *Generator) renderProject() error {
	return renderProjectGitignore(g.projectName)
}

func renderProjectGitignore(projectName string) error {
	gitignoreFile := filepath.Join(projectName, gitignore)
	buf := &bytes.Buffer{}
	if _, err := buf.Write([]byte(projectGitignoreTmpl)); err != nil {
		return err
	}

	return writeFileAndPrint(gitignoreFile, buf.Bytes(), defaultFileMode)
}

func (g *Generator) renderGoDep() error {
	buf := &bytes.Buffer{}
	if err := renderGopkgTomlFile(buf); err != nil {
		return err
	}
	if err := writeFileAndPrint(filepath.Join(g.projectName, gopkgtoml), buf.Bytes(), defaultFileMode); err != nil {
		return err
	}

	buf = &bytes.Buffer{}
	if err := renderGopkgLockFile(buf); err != nil {
		return err
	}
	return writeFileAndPrint(filepath.Join(g.projectName, gopkglock), buf.Bytes(), defaultFileMode)
}

func (g *Generator) renderCmd() error {
	cpDir := filepath.Join(g.projectName, cmdDir, g.projectName)
	return renderCmdFiles(cpDir, g.repoPath, g.apiVersion, g.kind)
}

func renderCmdFiles(cmdProjectDir, repoPath, apiVersion, kind string) error {
	buf := &bytes.Buffer{}

	td := tmplData{
		OperatorSDKImport: sdkImport,
		StubImport:        filepath.Join(repoPath, stubDir),
		K8sutilImport:     k8sutilImport,
		SDKVersionImport:  versionImport,
		APIVersion:        apiVersion,
		Kind:              kind,
	}

	if err := renderFile(buf, "cmd/<projectName>/main.go", mainTmpl, td); err != nil {
		return err
	}

	return writeFileAndPrint(filepath.Join(cmdProjectDir, main), buf.Bytes(), defaultFileMode)
}

func (g *Generator) renderConfig() error {
	cp := filepath.Join(g.projectName, configDir)
	return renderConfigFiles(cp, g.apiVersion, g.kind, g.projectName)
}

func renderConfigFiles(configDir, apiVersion, kind, projectName string) error {
	buf := &bytes.Buffer{}
	if err := renderConfigFile(buf, apiVersion, kind, projectName); err != nil {
		return err
	}
	return writeFileAndPrint(filepath.Join(configDir, config), buf.Bytes(), defaultFileMode)
}

func (g *Generator) renderDeploy() error {
	dp := filepath.Join(g.projectName, deployDir)
	return renderDeployFiles(dp, g.projectName, g.apiVersion, g.kind)
}

func renderRBAC(deployDir, projectName, groupName string) error {
	buf := &bytes.Buffer{}

	td := tmplData{
		ProjectName: projectName,
		GroupName:   groupName,
	}

	if err := renderFile(buf, rbacTmplName, rbacYamlTmpl, td); err != nil {
		return err
	}

	return writeFileAndPrint(filepath.Join(deployDir, rbacYaml), buf.Bytes(), defaultFileMode)
}

func renderDeployFiles(deployDir, projectName, apiVersion, kind string) error {
	buf := &bytes.Buffer{}
	rbacTd := tmplData{
		ProjectName: projectName,
		GroupName:   groupName(apiVersion),
	}
	if err := renderFile(buf, rbacTmplName, rbacYamlTmpl, rbacTd); err != nil {
		return err
	}
	if err := writeFileAndPrint(filepath.Join(deployDir, rbacYaml), buf.Bytes(), defaultFileMode); err != nil {
		return err
	}

	buf = &bytes.Buffer{}
	crdTd := tmplData{
		Kind:         kind,
		KindSingular: strings.ToLower(kind),
		KindPlural:   toPlural(strings.ToLower(kind)),
		GroupName:    groupName(apiVersion),
		Version:      version(apiVersion),
	}
	if err := renderFile(buf, crdTmplName, crdYamlTmpl, crdTd); err != nil {
		return err
	}
	if err := writeFileAndPrint(filepath.Join(deployDir, crdYaml), buf.Bytes(), defaultFileMode); err != nil {
		return err
	}

	buf = &bytes.Buffer{}
	crTd := tmplData{
		APIVersion: apiVersion,
		Kind:       kind,
	}
	if err := renderFile(buf, crTmplName, crYamlTmpl, crTd); err != nil {
		return err
	}
	return writeFileAndPrint(filepath.Join(deployDir, crYaml), buf.Bytes(), defaultFileMode)
}

// RenderOperatorYaml generates "deploy/operator.yaml"
func RenderOperatorYaml(c *Config, image string) error {
	buf := &bytes.Buffer{}
	td := tmplData{
		ProjectName: c.ProjectName,
		Image:       image,
	}
	if err := renderFile(buf, operatorTmplName, operatorYamlTmpl, td); err != nil {
		return err
	}
	return ioutil.WriteFile(operatorYaml, buf.Bytes(), defaultFileMode)
}

// RenderOlmCatalog generates catalog manifests "deploy/olm-catalog/*"
// The current working directory must be the project repository root
func RenderOlmCatalog(c *Config, image, version string) error {
	// mkdir deploy/olm-catalog
	repoPath, err := os.Getwd()
	if err != nil {
		return err
	}
	olmDir := filepath.Join(repoPath, olmCatalogDir)

	// deploy/olm-catalog/package.yaml
	buf := &bytes.Buffer{}
	cpTd := tmplData{
		PackageName: strings.ToLower(c.Kind),
		ChannelName: packageChannel,
		CurrentCSV:  getCSVName(strings.ToLower(c.Kind), version),
	}
	if err := renderFile(buf, catalogPackageYaml, catalogPackageTmpl, cpTd); err != nil {
		return err
	}
	path := filepath.Join(olmDir, catalogPackageYaml)
	if err := ioutil.WriteFile(path, buf.Bytes(), defaultFileMode); err != nil {
		return err
	}

	// deploy/olm-catalog/crd.yaml
	buf = &bytes.Buffer{}
	ccrdTd := tmplData{
		Kind:         c.Kind,
		KindSingular: strings.ToLower(c.Kind),
		KindPlural:   toPlural(strings.ToLower(c.Kind)),
		GroupName:    groupName(c.APIVersion),
		Version:      version,
	}
	if err := renderFile(buf, catalogCRDTmplName, crdTmpl, ccrdTd); err != nil {
		return err
	}
	path = filepath.Join(olmDir, crdYaml)
	if err := ioutil.WriteFile(path, buf.Bytes(), defaultFileMode); err != nil {
		return err
	}

	// deploy/olm-catalog/csv.yaml
	buf = &bytes.Buffer{}
	ccsvTd := tmplData{
		Kind:           c.Kind,
		KindSingular:   strings.ToLower(c.Kind),
		KindPlural:     toPlural(strings.ToLower(c.Kind)),
		GroupName:      groupName(c.APIVersion),
		CRDVersion:     version,
		CSVName:        getCSVName(strings.ToLower(c.Kind), version),
		Image:          image,
		CatalogVersion: version,
		ProjectName:    c.ProjectName,
	}
	if err := renderFile(buf, catalogCSVYaml, catalogCSVTmpl, ccsvTd); err != nil {
		return err
	}
	path = filepath.Join(olmDir, catalogCSVYaml)
	return ioutil.WriteFile(path, buf.Bytes(), defaultFileMode)
}

func getCSVName(name, version string) string {
	return name + ".v" + version
}

func (g *Generator) renderTmp() error {
	bDir := filepath.Join(g.projectName, buildDir)
	if err := renderBuildFiles(bDir, g.repoPath, g.projectName); err != nil {
		return err
	}

	cDir := filepath.Join(g.projectName, codegenDir)
	return renderCodegenFiles(cDir, g.repoPath, apiDirName(g.apiVersion), version(g.apiVersion), g.projectName)
}

func (g *Generator) renderVersion() error {
	buf := &bytes.Buffer{}

	td := tmplData{
		VersionNumber: "0.0.1",
	}

	if err := renderFile(buf, "version/version.go", versionTmpl, td); err != nil {
		return err
	}

	return writeFileAndPrint(filepath.Join(g.projectName, versionDir, versionfile), buf.Bytes(), defaultFileMode)
}

func renderBuildFiles(buildDir, repoPath, projectName string) error {
	buf := &bytes.Buffer{}

	bTd := tmplData{
		ProjectName: projectName,
		RepoPath:    repoPath,
	}

	if err := renderFile(buf, "tmp/build/build.sh", buildTmpl, bTd); err != nil {
		return err
	}
	if err := writeFileAndPrint(filepath.Join(buildDir, build), buf.Bytes(), defaultExecFileMode); err != nil {
		return err
	}

	buf = &bytes.Buffer{}
	if err := renderDockerBuildFile(buf); err != nil {
		return err
	}
	if err := writeFileAndPrint(filepath.Join(buildDir, dockerBuild), buf.Bytes(), defaultExecFileMode); err != nil {
		return err
	}

	buf = &bytes.Buffer{}
	dTd := tmplData{
		ProjectName: projectName,
	}
	if err := renderFile(buf, "tmp/build/Dockerfile", dockerFileTmpl, dTd); err != nil {
		return err
	}
	return writeFileAndPrint(filepath.Join(buildDir, dockerfile), buf.Bytes(), defaultFileMode)
}

func renderDockerBuildFile(w io.Writer) error {
	_, err := w.Write([]byte(dockerBuildTmpl))
	return err
}

func renderCodegenFiles(codegenDir, repoPath, apiDirName, version, projectName string) error {
	buf := &bytes.Buffer{}
	bTd := tmplData{
		ProjectName: projectName,
	}
	if err := renderFile(buf, "codegen/boilerplate.go.txt", boilerplateTmpl, bTd); err != nil {
		return err
	}
	if err := writeFileAndPrint(filepath.Join(codegenDir, boilerplate), buf.Bytes(), defaultFileMode); err != nil {
		return err
	}

	buf = &bytes.Buffer{}
	ugTd := tmplData{
		RepoPath:   repoPath,
		APIDirName: apiDirName,
		Version:    version,
	}
	if err := renderFile(buf, "codegen/update-generated.sh", updateGeneratedTmpl, ugTd); err != nil {
		return err
	}
	return writeFileAndPrint(filepath.Join(codegenDir, updateGenerated), buf.Bytes(), defaultExecFileMode)
}

func (g *Generator) renderPkg() error {
	v := version(g.apiVersion)
	adn := apiDirName(g.apiVersion)
	apiDir := filepath.Join(g.projectName, apisDir, adn, v)
	if err := renderAPIFiles(apiDir, groupName(g.apiVersion), v, g.kind); err != nil {
		return err
	}

	sDir := filepath.Join(g.projectName, stubDir)
	return renderStubFiles(sDir, g.repoPath, g.kind, adn, v)
}

func renderAPIFiles(apiDir, groupName, version, kind string) error {
	buf := &bytes.Buffer{}
	adTd := tmplData{
		GroupName: groupName,
		Version:   version,
	}
	if err := renderFile(buf, "apis/<apiDirName>/<version>/doc.go", apiDocTmpl, adTd); err != nil {
		return err
	}
	if err := writeFileAndPrint(filepath.Join(apiDir, doc), buf.Bytes(), defaultFileMode); err != nil {
		return err
	}

	buf = &bytes.Buffer{}
	arTd := tmplData{
		Kind:       kind,
		KindPlural: toPlural(strings.ToLower(kind)),
		GroupName:  groupName,
		Version:    version,
	}
	if err := renderFile(buf, "apis/<apiDirName>/<version>/register.go", apiRegisterTmpl, arTd); err != nil {
		return err
	}
	if err := writeFileAndPrint(filepath.Join(apiDir, register), buf.Bytes(), defaultFileMode); err != nil {
		return err
	}

	buf = &bytes.Buffer{}
	atTd := tmplData{
		Kind:    kind,
		Version: version,
	}
	if err := renderFile(buf, "apis/<apiDirName>/<version>/types.go", apiTypesTmpl, atTd); err != nil {
		return err
	}
	return writeFileAndPrint(filepath.Join(apiDir, types), buf.Bytes(), defaultFileMode)
}

func renderStubFiles(stubDir, repoPath, kind, apiDirName, version string) error {
	buf := &bytes.Buffer{}
	td := tmplData{
		OperatorSDKImport: sdkImport,
		RepoPath:          repoPath,
		Kind:              kind,
		APIDirName:        apiDirName,
		Version:           version,
	}
	if err := renderFile(buf, "stub/handler.go", handlerTmpl, td); err != nil {
		return err
	}
	return writeFileAndPrint(filepath.Join(stubDir, handler), buf.Bytes(), defaultFileMode)
}

type tmplData struct {
	VersionNumber string

	OperatorSDKImport string
	StubImport        string
	K8sutilImport     string
	SDKVersionImport  string

	APIVersion string
	Kind       string

	RepoPath   string
	APIDirName string
	Version    string

	ProjectName string
	GroupName   string

	// singular name to be used as an alias on the CLI and for display
	KindSingular string
	// plural name to be used in the URL: /apis/<group>/<version>/<plural>
	KindPlural string

	Image string
	Name  string

	PackageName string
	ChannelName string
	CurrentCSV  string

	CRDVersion     string
	CSVName        string
	CatalogVersion string
}

// Creates all the necesary directories for the generated files
func (g *Generator) generateDirStructure() error {
	dirsToCreate := []string{
		g.projectName,
		filepath.Join(g.projectName, cmdDir, g.projectName),
		filepath.Join(g.projectName, configDir),
		filepath.Join(g.projectName, deployDir),
		filepath.Join(g.projectName, olmCatalogDir),
		filepath.Join(g.projectName, buildDir),
		filepath.Join(g.projectName, codegenDir),
		filepath.Join(g.projectName, versionDir),
		filepath.Join(g.projectName, apisDir, apiDirName(g.apiVersion), version(g.apiVersion)),
		filepath.Join(g.projectName, stubDir),
	}

	for _, dir := range dirsToCreate {
		if err := os.MkdirAll(dir, defaultDirFileMode); err != nil {
			return err
		}
	}

	return nil
}

// Renders a file given a template, and fills in template fields according to values passed in the tmplData struct
func renderFile(w io.Writer, fileLoc string, fileTmpl string, info tmplData) error {
	t := template.New(fileLoc)

	t, err := t.Parse(fileTmpl)
	if err != nil {
		return err
	}

	return t.Execute(w, info)
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

// Writes file to a given path and data buffer, as well as prints out a message confirming creation of a file
func writeFileAndPrint(filePath string, data []byte, fileMode os.FileMode) error {
	if err := ioutil.WriteFile(filePath, data, fileMode); err != nil {
		return err
	}
	fmt.Printf("Create %v \n", filePath)
	return nil
}
