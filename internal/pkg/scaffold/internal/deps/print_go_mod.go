// Copyright 2019 The Operator-SDK Authors
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

package deps

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"text/tabwriter"

	"github.com/operator-framework/operator-sdk/internal/util/projutil"

	"github.com/rogpeppe/go-internal/modfile"
)

func ExecGoModTmpl(tmpl string) ([]byte, error) {
	projutil.MustInProjectRoot()
	repo := projutil.GetGoPkg()
	t, err := template.New("").Parse(tmpl)
	if err != nil {
		return nil, fmt.Errorf("failed to parse go mod template: (%v)", err)
	}
	buf := &bytes.Buffer{}
	if err := t.Execute(buf, struct{ Repo string }{Repo: repo}); err != nil {
		return nil, fmt.Errorf("failed to execute go mod template: (%v)", err)
	}
	return buf.Bytes(), nil
}

func PrintGoMod(b []byte) error {
	modFile, err := modfile.Parse("go.mod", b, nil)
	if err != nil {
		return err
	}

	mods := &GoModFile{modFile}
	buf := &bytes.Buffer{}
	w := tabwriter.NewWriter(buf, 16, 8, 0, '\t', 0)
	if err = mods.writeRequireSection(w); err != nil {
		return err
	}
	if len(mods.Replace) > 0 {
		if err = mods.writeReplaceSection(w); err != nil {
			return err
		}
	}
	if len(mods.Exclude) > 0 {
		if err = mods.writeExcludeSection(w); err != nil {
			return err
		}
	}
	if err := w.Flush(); err != nil {
		return err
	}

	fmt.Print(buf.String())
	return nil
}

type GoModFile struct {
	*modfile.File
}

func (g *GoModFile) writeRequireSection(w io.Writer) error {
	_, err := w.Write([]byte("REQUIRE\t\n"))
	if err != nil {
		return err
	}
	_, err = w.Write([]byte("Name\tVersion\tIndirect\t\n"))
	if err != nil {
		return err
	}
	for _, r := range g.Require {
		if err = writeRowRequire(w, r); err != nil {
			return err
		}
	}
	return nil
}

func (g *GoModFile) writeReplaceSection(w io.Writer) error {
	_, err := w.Write([]byte("\nREPLACE\t\n"))
	if err != nil {
		return err
	}
	_, err = w.Write([]byte("Old Name\tOld Version\tNew Name\tNew Version\t\n"))
	if err != nil {
		return err
	}
	for _, r := range g.Replace {
		if err = writeRowReplace(w, r); err != nil {
			return err
		}
	}
	return nil
}

func (g *GoModFile) writeExcludeSection(w io.Writer) error {
	_, err := w.Write([]byte("\nEXCLUDE\t\n"))
	if err != nil {
		return err
	}
	_, err = w.Write([]byte("Name\tVersion\t\n"))
	if err != nil {
		return err
	}
	for _, e := range g.Exclude {
		if err = writeRowExclude(w, e); err != nil {
			return err
		}
	}
	return nil
}

func writeRowRequire(w io.Writer, r *modfile.Require) error {
	row := fmt.Sprintf("%v\t%v\t", r.Mod.Path, r.Mod.Version)
	if r.Indirect {
		row += "true\t\n"
	} else {
		row += "\n"
	}
	_, err := w.Write([]byte(row))
	return err
}

func writeRowReplace(w io.Writer, r *modfile.Replace) error {
	row := fmt.Sprintf("%v\t%v\t%v\t%v\t\n", r.Old.Path, r.Old.Version, r.New.Path, r.New.Version)
	_, err := w.Write([]byte(row))
	return err
}

func writeRowExclude(w io.Writer, e *modfile.Exclude) error {
	row := fmt.Sprintf("%v\t%v\t\n", e.Mod.Path, e.Mod.Version)
	_, err := w.Write([]byte(row))
	return err
}
