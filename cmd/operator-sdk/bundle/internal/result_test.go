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
	"io/ioutil"
	"os"
	"testing"

	. "github.com/onsi/ginkgo" //nolint:golint
	. "github.com/onsi/gomega" //nolint:golint
	log "github.com/sirupsen/logrus"
)

func TestResult(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Output Result Tests")
}

var _ = Describe("Output Result", func() {
	var result Result

	BeforeEach(func() {
		result = NewResult()
	})

	Describe("Test Result Model Manipulation", func() {
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

	Describe("Test PrintText", func() {
		It("should work successfully with valid log levels", func() {
			logger := log.NewEntry(NewLoggerTo(os.Stderr))
			result.AddError(errors.New("example of an error"))
			result.AddWarn(errors.New("example of a warn"))
			result.AddInfo("Example of an info")

			err := result.printText(logger)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should fail when an invalid log level is found", func() {
			// This scenario can just occurs if the setters are not used
			logger := log.NewEntry(NewLoggerTo(os.Stderr))
			result.Outputs = append(result.Outputs, output{
				Type:    log.TraceLevel.String(),
				Message: "invalid",
			})

			err := result.printText(logger)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("unknown output level \"trace\""))
		})

		It("should fail when is not possible parse the log level", func() {
			// This scenario can just occurs if the setters are not used
			logger := log.NewEntry(NewLoggerTo(os.Stderr))
			result.Outputs = append(result.Outputs, output{
				Type:    "invalid",
				Message: "invalid",
			})

			err := result.printText(logger)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("not a valid logrus Level: \"invalid\""))
		})
	})

	Describe("Test prepare()", func() {
		It("should finished with passed flagged with false when has an output with the ErrorLevel", func() {
			// This scenario can just occurs if the setters are not used
			result.Outputs = append(result.Outputs, output{
				Type:    log.ErrorLevel.String(),
				Message: "error",
			})

			result.Outputs = append(result.Outputs, output{
				Type:    log.InfoLevel.String(),
				Message: "info",
			})

			result.Outputs = append(result.Outputs, output{
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
			result.Outputs = append(result.Outputs, output{
				Type:    "invalid",
				Message: "invalid",
			})

			err := result.prepare()
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Test getPrintFuncFormat()", func() {
		It("should return the printJSON func which works successfully", func() {
			By("passing the format`json-alpha1`")
			printf := result.getPrintFuncFormat(JSONAlpha1)
			Expect(printf).ToNot(BeNil())

			r, w, _ := os.Pipe()
			tmp := os.Stdout
			defer func() {
				os.Stdout = tmp
			}()
			os.Stdout = w
			go func() {
				err := printf(result)
				Expect(err).NotTo(HaveOccurred())
				w.Close()
			}()
			stdout, _ := ioutil.ReadAll(r)
			res := map[string]interface{}{}

			Expect(json.Unmarshal(stdout, &res)).To(Succeed())
			Expect(res).To(HaveKeyWithValue("passed", true))
		})

		It("should return the printText func which works successfully", func() {
			By("passing ANY value as format")
			printf := result.getPrintFuncFormat("ANY")
			Expect(printf).ToNot(BeNil())

			r, w, _ := os.Pipe()
			tmp := os.Stdout
			defer func() {
				os.Stdout = tmp
			}()
			os.Stdout = w
			go func() {
				err := printf(result)
				Expect(err).NotTo(HaveOccurred())
				w.Close()
			}()
			stdout, _ := ioutil.ReadAll(r)
			res := map[string]interface{}{}

			Expect(json.Unmarshal(stdout, &res)).NotTo(Succeed())
		})
	})

	Describe("Test printJSON()", func() {
		It("should return a pretty JSON", func() {
			By("adding an error")
			result.AddError(errors.New("example of an error"))
			r, w, _ := os.Pipe()
			tmp := os.Stdout
			defer func() {
				os.Stdout = tmp
			}()
			os.Stdout = w
			go func() {
				Expect(result.printJSON()).To(Succeed())
				w.Close()
			}()
			stdout, _ := ioutil.ReadAll(r)
			res := map[string]interface{}{}

			By("checking if stdout is an JSON")
			Expect(json.Unmarshal(stdout, &res)).To(Succeed())

			By("checking if the stdout has the expected values")
			Expect(res).To(HaveKeyWithValue("passed", false))
			Expect(res).To(HaveKey("outputs"))
			Expect(string(stdout)).To(ContainSubstring("example of an error"))
		})
	})

	Describe("Test PrintWithFormat()", func() {
		It("should print a JSON", func() {
			By("passing the format`json-alpha1`")
			result.AddWarn(errors.New("example of an warn"))
			r, w, _ := os.Pipe()
			tmp := os.Stdout
			defer func() {
				os.Stdout = tmp
			}()
			os.Stdout = w
			go func() {
				Expect(result.PrintWithFormat(JSONAlpha1)).To(Succeed())
				w.Close()
			}()
			stdout, _ := ioutil.ReadAll(r)
			res := map[string]interface{}{}
			By("checking if stdout is an JSON")
			Expect(json.Unmarshal(stdout, &res)).To(Succeed())

			By("checking if the stdout has the expected values")
			Expect(res).To(HaveKeyWithValue("passed", true))
			Expect(res).To(HaveKey("outputs"))
			Expect(string(stdout)).To(ContainSubstring("example of an warn"))
		})

		It("should NOT print a JSON", func() {
			By("passing the format`text`")
			result.AddWarn(errors.New("example of an warn"))
			r, w, _ := os.Pipe()
			tmp := os.Stdout
			defer func() {
				os.Stdout = tmp
			}()
			os.Stdout = w
			go func() {
				Expect(result.PrintWithFormat(Text)).To(Succeed())
				w.Close()
			}()
			stdout, _ := ioutil.ReadAll(r)
			res := map[string]interface{}{}

			By("checking that the output is NOT a JSON")
			Expect(json.Unmarshal(stdout, &res)).NotTo(Succeed())

			By("checking if the stdout has the expected values")
			Expect(string(stdout)).To(ContainSubstring("example of an warn"))
		})

		It("should return an error to prepare the output", func() {
			By("adding an invalid log level`")
			result.Outputs = append(result.Outputs, output{
				Type:    "invalid",
				Message: "invalid",
			})

			Expect(result.PrintWithFormat(Text)).NotTo(Succeed())
		})
	})
})
