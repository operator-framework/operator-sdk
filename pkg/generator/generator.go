package generator

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

const (
	defaultFileMode = 0750
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
	main = "main.go"
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
	return g.renderTmp()
}

func (g *Generator) renderCmd() error {
	cpDir := filepath.Join(g.projectName, cmdDir, g.projectName)
	if err := os.MkdirAll(cpDir, defaultFileMode); err != nil {
		return err
	}

	buf := &bytes.Buffer{}
	if err := renderMain(buf, g.repoPath, version(g.apiVersion), apiDirName(g.apiVersion), g.kind, toPlural(g.kind)); err != nil {
		return err
	}
	return ioutil.WriteFile(filepath.Join(cpDir, main), buf.Bytes(), 0644)
}

func toPlural(kind string) string {
	return kind + "Plural"
}

func (g *Generator) renderConfig() error {
	if err := os.MkdirAll(filepath.Join(g.projectName, configDir), defaultFileMode); err != nil {
		return err
	}
	// TODO render files.
	return nil
}

func (g *Generator) renderDeploy() error {
	if err := os.MkdirAll(filepath.Join(g.projectName, deployDir), defaultFileMode); err != nil {
		return err
	}
	// TODO render files.
	return nil
}

func (g *Generator) renderTmp() error {
	if err := os.MkdirAll(filepath.Join(g.projectName, buildDir), defaultFileMode); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(g.projectName, codegenDir), defaultFileMode); err != nil {
		return err
	}
	// TODO render files.
	return nil
}

func (g *Generator) renderPkg() error {
	if err := os.MkdirAll(filepath.Join(g.projectName, apisDir, apiDirName(g.apiVersion), version(g.apiVersion)), defaultFileMode); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(g.projectName, stubDir), defaultFileMode); err != nil {
		return err
	}
	// TODO render files.
	return nil
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
func apiDirName(apiGroup string) string {
	return strings.Split(groupName(apiGroup), ".")[0]
}
