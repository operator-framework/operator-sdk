package scaffold

import "io"

type gopkg struct {
}

func NewGopkgCodegen() Codegen {
	return &gopkg{}
}

func (g *gopkg) Render(w io.Writer) error {
	_, err := w.Write([]byte(gopkgTomlTemplate))
	return err
}

const gopkgTomlTemplate = `# Force dep to vendor the code generators, which aren't imported just used at dev time.
required = [
  "k8s.io/code-generator/cmd/defaulter-gen",
  "k8s.io/code-generator/cmd/deepcopy-gen",
  "k8s.io/code-generator/cmd/conversion-gen",
  "k8s.io/code-generator/cmd/client-gen",
  "k8s.io/code-generator/cmd/lister-gen",
  "k8s.io/code-generator/cmd/informer-gen",
  "k8s.io/code-generator/cmd/openapi-gen",
  "k8s.io/gengo/args",
]

[[override]]
  name = "k8s.io/code-generator"
  # revision for tag "kubernetes-1.11.2"
  revision = "6702109cc68eb6fe6350b83e14407c8d7309fd1a"

[[override]]
  name = "k8s.io/api"
  # revision for tag "kubernetes-1.11.2"
  revision = "2d6f90ab1293a1fb871cf149423ebb72aa7423aa"

[[override]]
  name = "k8s.io/apiextensions-apiserver"
  # revision for tag "kubernetes-1.11.2"
  revision = "408db4a50408e2149acbd657bceb2480c13cb0a4"

[[override]]
  name = "k8s.io/apimachinery"
  # revision for tag "kubernetes-1.11.2"
  revision = "103fd098999dc9c0c88536f5c9ad2e5da39373ae"

[[override]]
  name = "k8s.io/client-go"
  # revision for tag "kubernetes-1.11.2"
  revision = "1f13a808da65775f22cbf47862c4e5898d8f4ca1"

[[override]]
  name = "sigs.k8s.io/controller-runtime"
  revision = "53fc44b56078cd095b11bd44cfa0288ee4cf718f"

[[constraint]]
  name = "github.com/operator-framework/operator-sdk"
  # The version rule is used for a specific release and the master branch for in between releases.
  branch = "master"
  # version = "=v0.0.6"

[prune]
  go-tests = true
  non-go = true
  unused-packages = true
  
  [[prune.project]]
    name = "k8s.io/code-generator"
    non-go = false
    unused-packages = false
`
