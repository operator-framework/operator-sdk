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

package pkg

import (
	"os"

	log "github.com/sirupsen/logrus"
)

// CheckError will check the error and exit with 1 when as errors
func CheckError(msg string, err error) {
	if err != nil {
		log.Errorf("Error %s: %s", msg, err)
		os.Exit(1)
	}
}

// RunOlmIntegration runs all commands to integrate the project with OLM
func RunOlmIntegration(ctx *SampleContext) {
	log.Infof("Integrating project with OLM")
	err := ctx.DisableOLMBundleInteractiveMode()
	CheckError("disabling the OLM bundle", err)

	err = ctx.Make("bundle", "IMG="+ctx.ImageName)
	CheckError("running make bundle", err)

	err = ctx.Make("bundle-build", "BUNDLE_IMG="+ctx.BundleImageName)
	CheckError("running make bundle-build", err)
}
