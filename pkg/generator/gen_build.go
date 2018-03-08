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
