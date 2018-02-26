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
	main     = "main.go"
	handler  = "handler.go"
	doc      = "doc.go"
	register = "register.go"
	types    = "types.go"
	build    = "build.sh"
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
	return g.pullDep()
}

func (g *Generator) pullDep() error {
	// TODO: After we have setup scalffolding, pull dependencies: render Gopkg.toml, then `dep ensure`
	return nil
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
	if err := os.MkdirAll(filepath.Join(g.projectName, configDir), defaultDirFileMode); err != nil {
		return err
	}
	// TODO render files.
	return nil
}

func (g *Generator) renderDeploy() error {
	if err := os.MkdirAll(filepath.Join(g.projectName, deployDir), defaultDirFileMode); err != nil {
		return err
	}
	// TODO render files.
	return nil
}

func (g *Generator) renderTmp() error {
	bDir := filepath.Join(g.projectName, buildDir)
	if err := os.MkdirAll(bDir, defaultDirFileMode); err != nil {
		return err
	}
	if err := renderBuildFiles(bDir, g.repoPath, g.projectName); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Join(g.projectName, codegenDir), defaultDirFileMode); err != nil {
		return err
	}
	// TODO render files.
	return nil
}

func renderBuildFiles(buildDir, repoPath, projectName string) error {
	buf := &bytes.Buffer{}
	if err := renderBuildFile(buf, repoPath, projectName); err != nil {
		return err
	}
	return ioutil.WriteFile(filepath.Join(buildDir, build), buf.Bytes(), defaultExecFileMode)
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
