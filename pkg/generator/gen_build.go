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

// renderBuildFile generates the tmp/build/build.sh file given a repo path ("github.com/coreos/app-operator")
// and projectName ("app-operator").
func renderBuildFile(w io.Writer, repo, projectName string) error {
	t := template.New("tmp/build/build.sh")
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

// renderDockerBuildFile generates the docker_build.sh script which builds the docker image for this operator.
func renderDockerBuildFile(w io.Writer) error {
	_, err := w.Write([]byte(dockerBuildTmpl))
	return err
}

// DockerFile contains all the customized data needed to generate tmp/build/Dockerfie
// for a new operator when pairing with dockerFileTmpl template.
type DockerFile struct {
	ProjectName string
}

// renderDockerFile generates the tmp/build/Dockerfile file given the projectName ("app-operator").
func renderDockerFile(w io.Writer, projectName string) error {
	t := template.New("tmp/build/Dockerfile")
	t, err := t.Parse(dockerFileTmpl)
	if err != nil {
		return err
	}

	df := DockerFile{
		ProjectName: projectName,
	}
	return t.Execute(w, df)
}
