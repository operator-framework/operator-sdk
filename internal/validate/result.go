// Copyright 2020 The Operator-SDK Authors
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

package validate

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	apierrors "github.com/operator-framework/api/pkg/validation/errors"
	registrybundle "github.com/operator-framework/operator-registry/pkg/lib/bundle"
	"github.com/sirupsen/logrus"
)

// TODO: might be nice to formalize this with apiVersion=sdk.operatorframework.io/v1alpha1, kind=ValidationResult.

const (
	JSONAlpha1Output = "json-alpha1"
	TextOutput       = "text"
)

// Result represents the final result
type Result struct {
	Passed  bool     `json:"passed"`
	Outputs []Output `json:"outputs"`
}

// Output represents the logs which are used to return the final result in the JSON format
type Output struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// NewResult return a new result object which starts with passed == true since has no errors
func NewResult() *Result {
	return &Result{Passed: true}
}

// AddManifestResults adds warnings and errors in results to Results.
func (r *Result) AddManifestResults(results ...apierrors.ManifestResult) {
	for _, mr := range results {
		for _, w := range mr.Warnings {
			r.AddWarn(w)
		}
		for _, e := range mr.Errors {
			r.AddError(e)
		}
	}
}

// AddInfo will add a log to the result with the Info Level
func (r *Result) AddInfo(msg string) {
	r.Outputs = append(r.Outputs, Output{
		Type:    logrus.InfoLevel.String(),
		Message: msg,
	})
}

// AddError will add a log to the result with the Error Level
func (r *Result) AddError(err error) {
	verr := registrybundle.ValidationError{}
	if errors.As(err, &verr) {
		for _, valErr := range verr.Errors {
			r.Outputs = append(r.Outputs, Output{
				Type:    logrus.ErrorLevel.String(),
				Message: valErr.Error(),
			})
		}
	} else {
		r.Outputs = append(r.Outputs, Output{
			Type:    logrus.ErrorLevel.String(),
			Message: err.Error(),
		})
	}
	r.Passed = false
}

// AddWarn will add a log to the result with the Warn Level
func (r *Result) AddWarn(err error) {
	r.Outputs = append(r.Outputs, Output{
		Type:    logrus.WarnLevel.String(),
		Message: err.Error(),
	})
}

// Combine creates a new Result and calls Result.Combine(results).
func Combine(results ...Result) (r Result, err error) {
	err = r.Combine(results...)
	return r, err
}

// Combine combines results into r, setting r.Passed = false if
// any Result in results has r.Passed == false.
func (r *Result) Combine(results ...Result) error {
	for _, result := range results {
		r.Outputs = append(r.Outputs, result.Outputs...)
	}
	return r.prepare()
}

// prepare should be used when writing an Result to a non-log writer.
// it will ensure that the passed boolean will properly set in the case of the setters were not properly used
func (r *Result) prepare() error {
	r.Passed = true
	for i, obj := range r.Outputs {
		lvl, err := logrus.ParseLevel(obj.Type)
		if err != nil {
			return err
		}
		if r.Passed && lvl == logrus.ErrorLevel {
			r.Passed = false
		}
		lvlBytes, _ := lvl.MarshalText()
		r.Outputs[i].Type = string(lvlBytes)
	}
	return nil
}

// PrintWithFormat prints output to w in format, and exits if some object in output
// is not in a passing state.
func (r *Result) PrintWithFormat(format string) (failed bool, err error) {
	return r.printWithFormat(os.Stdout, format)
}

func (r *Result) printWithFormat(w io.Writer, format string) (failed bool, err error) {
	// the prepare will ensure the result data if the setters were not used
	if err = r.prepare(); err != nil {
		return failed, fmt.Errorf("error preparing output: %v", err)
	}

	switch format {
	case JSONAlpha1Output:
		err = r.printJSON(w)
	case TextOutput: // Text
		// Address all to the Stdout when the type is not JSON
		entry := logrus.NewEntry(newLoggerTo(w))
		err = r.printText(entry)
	default:
		return failed, fmt.Errorf("invalid result format type: %s", format)
	}
	if err == nil && !r.Passed {
		failed = true
	}

	return failed, err
}

// printText will print the Output in human readable format
func (r *Result) printText(logger *logrus.Entry) error {
	for _, obj := range r.Outputs {
		lvl, err := logrus.ParseLevel(obj.Type)
		if err != nil {
			return err
		}
		switch lvl {
		case logrus.InfoLevel:
			logger.Info(obj.Message)
		case logrus.WarnLevel:
			logger.Warn(obj.Message)
		case logrus.ErrorLevel:
			logger.Error(obj.Message)
		default:
			return fmt.Errorf("unknown log output level %q", obj.Type)
		}
	}

	return nil
}

// printJSON will print the Output in JSON format
func (r *Result) printJSON(w io.Writer) error {
	prettyJSON, err := json.MarshalIndent(r, "", "    ")
	if err != nil {
		return err
	}
	fmt.Fprintf(w, "%s\n", string(prettyJSON))
	return nil
}

func newLoggerTo(w io.Writer) *logrus.Logger {
	logger := logrus.New()
	logger.SetOutput(w)
	return logger
}
