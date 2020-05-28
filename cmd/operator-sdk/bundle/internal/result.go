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

package internal

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	registrybundle "github.com/operator-framework/operator-registry/pkg/lib/bundle"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
)

const (
	JSONAlpha1 = "json-alpha1"
	Text       = "text"
)

// Result represents the final result
type Result struct {
	Passed  bool     `json:"passed"`
	Outputs []output `json:"outputs"`
}

// output represents the logs which are used to return the final result in the JSON format
type output struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// NewResult return a new result object which starts with passed == true since has no errors
func NewResult() Result {
	return Result{Passed: true}
}

// AddInfo will add a log to the result with the Info Level
func (o *Result) AddInfo(msg string) {
	o.Outputs = append(o.Outputs, output{
		Type:    logrus.InfoLevel.String(),
		Message: msg,
	})
}

// AddError will add a log to the result with the Error Level
func (o *Result) AddError(err error) {
	verr := registrybundle.ValidationError{}
	if errors.As(err, &verr) {
		for _, valErr := range verr.Errors {
			o.Outputs = append(o.Outputs, output{
				Type:    logrus.ErrorLevel.String(),
				Message: valErr.Error(),
			})
		}
	} else {
		o.Outputs = append(o.Outputs, output{
			Type:    logrus.ErrorLevel.String(),
			Message: err.Error(),
		})
	}
	o.Passed = false
}

// AddWarn will add a log to the result with the Warn Level
func (o *Result) AddWarn(err error) {
	o.Outputs = append(o.Outputs, output{
		Type:    logrus.WarnLevel.String(),
		Message: err.Error(),
	})
}

// printText will print the output in human readable format
func (o *Result) printText(logger *logrus.Entry) error {
	for _, obj := range o.Outputs {
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
			return fmt.Errorf("unknown output level %q", obj.Type)
		}
	}

	return nil
}

// printJSON will print the output in JSON format
func (o *Result) printJSON() error {
	prettyJSON, err := json.MarshalIndent(o, "", "    ")
	if err != nil {
		return fmt.Errorf("error marshaling JSON output: %v", err)
	}
	fmt.Printf("%s\n", string(prettyJSON))
	return nil
}

// prepare should be used when writing an Result to a non-log writer.
// it will ensure that the passed boolean will properly set in the case of the setters were not properly used
func (o *Result) prepare() error {
	o.Passed = true
	for i, obj := range o.Outputs {
		lvl, err := logrus.ParseLevel(obj.Type)
		if err != nil {
			return err
		}
		if o.Passed && lvl == logrus.ErrorLevel {
			o.Passed = false
		}
		lvlBytes, _ := lvl.MarshalText()
		o.Outputs[i].Type = string(lvlBytes)
	}
	return nil
}

// PrintWithFormat prints output to w in format, and exits if some object in output
// is not in a passing state.
func (o *Result) PrintWithFormat(format string) (err error) {
	// the prepare will ensure the result data if the setters were not used
	if err = o.prepare(); err != nil {
		return fmt.Errorf("error to prepare output: %v", err)
	}

	printf := o.getPrintFuncFormat(format)
	if err = printf(*o); err == nil && !o.Passed {
		os.Exit(1) // Exit with error when any Error type was added
	}
	return err
}

// getPrintFuncFormat returns a function that writes an Result to w in a given
// format, defaulting to "text" if format is not recognized.
func (o *Result) getPrintFuncFormat(format string) func(Result) error {
	// PrintWithFormat output in desired format.
	switch format {
	case JSONAlpha1:
		return func(o Result) error {
			return o.printJSON()
		}
	}

	// Address all to the Stdout when the type is not JSON
	logger := log.NewEntry(NewLoggerTo(os.Stdout))
	return func(o Result) error {
		return o.printText(logger)
	}
}
