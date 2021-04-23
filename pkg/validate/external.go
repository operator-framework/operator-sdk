// Copyright 2021 The Operator-SDK Authors
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
	"context"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ValidatorEntrypointsEnv should be set to a Unix path list ("/path/to/e1.sh:/path/to/e2")
// containing the list of entrypoints to external (out of code tree) validator scripts
// or binaries to run. Requirements for entrypoints:
// - Entrypoints must be executable by the user running the parent process.
// - The stdout output of an entrypoint *must* conform to the JSON representation
//   of Result so results can be parsed and collated with other internal validators.
// - An entrypoint should exit 1 and print output to stderr only if the entrypoint itself
//   fails for some reason. If the bundle fails to pass validation, that information
//   should be encoded in the Result printed to stdout as a Type="error".
//
// WARNING: the script or binary at the base of this path will be executed arbitrarily,
// so make sure you check the contents of that script or binary prior to running.
const ValidatorEntrypointsEnv = "OPERATOR_SDK_VALIDATOR_ENTRYPOINTS"

// For mocking in tests.
var stderr io.Writer = os.Stderr

// GetExternalValidatorEntrypoints returns a list of external validator entrypoints
// retrieved from ValidatorEntrypointsEnv and true if set. If not set or set to the empty string,
// GetExternalValidatorEntrypoints returns false.
func GetExternalValidatorEntrypoints() ([]string, bool) {
	entrypoints, isSet := os.LookupEnv(ValidatorEntrypointsEnv)
	if !isSet || strings.TrimSpace(entrypoints) == "" {
		return nil, false
	}
	return filepath.SplitList(entrypoints), true
}

// RunExternalValidators runs each entrypoint in entrypoint as a exec.Cmd with the single argument bundleRoot.
// External validators are expected to parse the bundle themselves with library APIs available
// in https://pkg.go.dev/github.com/operator-framework/api/pkg/manifests.
//
// TODO(estroz): what other information should be passed? Output of `docker inspect`?
func RunExternalValidators(ctx context.Context, entrypoints []string, bundleRoot string) ([]Result, error) {
	results := make([]Result, len(entrypoints))
	for i, entrypoint := range entrypoints {
		cmd := exec.CommandContext(ctx, entrypoint, bundleRoot)
		// Let error text go to stderr.
		cmd.Stderr = stderr
		// The validator should only exit non-zero if the entrypoint itself failed to run,
		// not if the bundle failed validation.
		out, err := cmd.Output()
		if err != nil {
			return nil, err
		}
		// Ensure output conforms to the Output spec.
		dec := json.NewDecoder(bytes.NewBuffer(out))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&results[i]); err != nil {
			return nil, err
		}
	}
	return results, nil
}
