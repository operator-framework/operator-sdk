package generator

import (
	"io"
	"text/template"
)

// Build contains all the customized data needed to generate tmp/build.sh
// for a new operator when pairing with buildTmpl template.
type Build struct {
	RepoPath    string
	ProjectName string
}

// renderBuildFile generates the tmp/build.sh file given a repo path ("github.com/coreos/app-operator")
// and projectName ("app-operator").
func renderBuildFile(w io.Writer, repo, projectName string) error {
	t := template.New("tmp/build.sh")
	t, err := t.Parse(buildTmpl)
	if err != nil {
		return err
	}

	m := Build{
		RepoPath:    repo,
		ProjectName: projectName,
	}
	return t.Execute(w, m)
}
