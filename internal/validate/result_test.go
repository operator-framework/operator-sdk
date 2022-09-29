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
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"
)

var _ = Describe("Output Result", func() {
	var result *Result

	BeforeEach(func() {
		result = NewResult()
	})

	Describe("Result Model Manipulation", func() {
		It("should add the error with ErrorLevel and passed should be flagged with false", func() {
			result.AddError(errors.New("example of an error"))

			Expect(result).NotTo(BeNil())
			Expect(result.Passed).To(BeFalse())
			Expect(result.Outputs[0].Type).To(Equal(log.ErrorLevel.String()))
			Expect(result.Outputs[0].Message).To(Equal("example of an error"))
		})

		It("should add the error with WarnLevel and passed should be flagged with true", func() {
			result.AddWarn(errors.New("example of a warn"))

			Expect(result).NotTo(BeNil())
			Expect(result.Passed).To(BeTrue())
			Expect(result.Outputs[0].Type).To(Equal(log.WarnLevel.String()))
			Expect(result.Outputs[0].Message).To(Equal("example of a warn"))
		})

		It("should add msg with InfoLevel and passed should be flagged with true", func() {
			result.AddInfo("Example of an info")

			Expect(result).NotTo(BeNil())
			Expect(result.Passed).To(BeTrue())
			Expect(result.Outputs[0].Type).To(Equal(log.InfoLevel.String()))
			Expect(result.Outputs[0].Message).To(Equal("Example of an info"))
		})

		It("should passed be flagged with false when has many outputs and an error", func() {
			result.AddError(errors.New("example of an error"))
			result.AddWarn(errors.New("example of a warn"))
			result.AddInfo("Example of an info")

			Expect(result).NotTo(BeNil())
			Expect(result.Passed).To(BeFalse())
			Expect(result.Outputs).To(HaveLen(3))
		})

	})

	Describe("PrintText", func() {
		It("should work successfully with valid log levels", func() {
			logger := log.NewEntry(newLoggerTo(os.Stderr))
			result.AddError(errors.New("example of an error"))
			result.AddWarn(errors.New("example of a warn"))
			result.AddInfo("Example of an info")

			err := result.printText(logger)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should fail when an invalid log level is found", func() {
			// This scenario can just occurs if the setters are not used
			logger := log.NewEntry(newLoggerTo(os.Stderr))
			result.Outputs = append(result.Outputs, Output{
				Type:    log.TraceLevel.String(),
				Message: "invalid",
			})

			err := result.printText(logger)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("unknown log output level \"trace\""))
		})

		It("should fail when is not possible parse the log level", func() {
			// This scenario can just occurs if the setters are not used
			logger := log.NewEntry(newLoggerTo(os.Stderr))
			result.Outputs = append(result.Outputs, Output{
				Type:    "invalid",
				Message: "invalid",
			})

			err := result.printText(logger)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("not a valid logrus Level: \"invalid\""))
		})
	})

	Describe("prepare", func() {
		It("should finished with passed flagged with false when has an output with the ErrorLevel", func() {
			// This scenario can just occurs if the setters are not used
			result.Outputs = append(result.Outputs, Output{
				Type:    log.ErrorLevel.String(),
				Message: "error",
			})

			result.Outputs = append(result.Outputs, Output{
				Type:    log.InfoLevel.String(),
				Message: "info",
			})

			result.Outputs = append(result.Outputs, Output{
				Type:    log.WarnLevel.String(),
				Message: "warn",
			})

			err := result.prepare()
			Expect(err).NotTo(HaveOccurred())
			Expect(result).NotTo(BeNil())
			Expect(result.Passed).To(BeFalse())
			Expect(result.Outputs).To(HaveLen(3))
		})

		It("should fail when an invalid log level is found", func() {
			// This scenario can just occurs if the setters are not used
			result.Outputs = append(result.Outputs, Output{
				Type:    "invalid",
				Message: "invalid",
			})

			err := result.prepare()
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("PrintWithFormat", func() {
		var w *bytes.Buffer
		var output []byte
		var resJSON Result
		const warnText = "example of a warning"
		const errorText = "example of an error"

		BeforeEach(func() {
			w = &bytes.Buffer{}
			resJSON = Result{}
		})

		Context("json-alpha1 formatting", func() {
			It("prints a warning", func() {
				result.AddWarn(errors.New(warnText))
				failed, err := result.printWithFormat(w, JSONAlpha1Output)
				Expect(failed).To(BeFalse())
				Expect(err).To(Succeed())
				output = w.Bytes()
				Expect(json.Unmarshal(output, &resJSON)).To(Succeed(), string(output))
				Expect(resJSON.Passed).To(BeTrue())
				Expect(resJSON.Outputs).To(HaveLen(1))
				Expect(resJSON.Outputs[0].Type).To(Equal("warning"))
				Expect(resJSON.Outputs[0].Message).To(Equal(warnText))
			})

			It("prints an error", func() {
				result.AddError(errors.New(errorText))
				failed, err := result.printWithFormat(w, JSONAlpha1Output)
				fmt.Println(failed)
				fmt.Println(err)
				Expect(failed).To(BeTrue())
				Expect(err).To(Succeed())
				output = w.Bytes()
				Expect(json.Unmarshal(output, &resJSON)).To(Succeed(), string(output))
				Expect(resJSON.Passed).To(BeFalse())
				Expect(resJSON.Outputs).To(HaveLen(1))
				Expect(resJSON.Outputs[0].Type).To(Equal("error"))
				Expect(resJSON.Outputs[0].Message).To(Equal(errorText))
			})
		})

		Context("text formatting", func() {
			It("prints a warning", func() {
				result.AddWarn(errors.New(warnText))
				failed, err := result.printWithFormat(w, TextOutput)
				Expect(failed).To(BeFalse())
				Expect(err).To(Succeed())
				output = w.Bytes()
				lines := bytes.Split(bytes.TrimSpace(output), []byte("\n"))
				Expect(lines).To(HaveLen(1))
				line := string(lines[0])
				Expect(line).To(ContainSubstring("level=warning"), line)
				Expect(line).To(ContainSubstring(`msg="example of a warning"`), line)
			})

			It("prints an error", func() {
				result.AddError(errors.New(errorText))
				failed, err := result.printWithFormat(w, TextOutput)
				Expect(failed).To(BeTrue())
				Expect(err).To(Succeed())
				output = w.Bytes()
				lines := bytes.Split(bytes.TrimSpace(output), []byte("\n"))
				Expect(lines).To(HaveLen(1))
				line := string(lines[0])
				fmt.Println(line)
				Expect(line).To(ContainSubstring("level=error"), line)
				Expect(line).To(ContainSubstring(`msg="example of an error"`), line)
			})
		})

		Context("returns an error", func() {
			It("gets an invalid log level", func() {
				result.Outputs = append(result.Outputs, Output{
					Type:    "invalid",
					Message: "invalid",
				})
				failed, err := result.printWithFormat(w, TextOutput)
				Expect(failed).To(BeFalse())
				Expect(err).NotTo(Succeed())
			})
		})
	})
})
