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
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	apierrors "github.com/operator-framework/api/pkg/validation/errors"
)

// For mocking in tests.
var stderr io.Writer = os.Stderr

// GetExternalValidatorEntrypoints returns a list of external validator entrypoints
// retrieved from given entrypoints string. If not set or set to the empty string,
// GetExternalValidatorEntrypoints returns false.
func GetExternalValidatorEntrypoints(entrypoints string) ([]string, bool) {
	if strings.TrimSpace(entrypoints) == "" {
		return nil, false
	}
	return filepath.SplitList(entrypoints), true
}

// RunExternalValidators runs each entrypoint in entrypoint as a exec.Cmd with the
// single argument bundleRoot. External validators are expected to parse the bundle
// themselves with library APIs available in
// https://pkg.go.dev/github.com/operator-framework/api/pkg/manifests.
func RunExternalValidators(ctx context.Context, entrypoints []string, bundleRoot string) ([]apierrors.ManifestResult, error) {
	manifestresults := make([]apierrors.ManifestResult, len(entrypoints))
	for i, entrypoint := range entrypoints {
		cmd := exec.CommandContext(ctx, entrypoint, bundleRoot)
		// Let error text go to stderr.
		cmd.Stderr = stderr

		// The validator should only exit non-zero if the entrypoint itself failed to run,
		// not if the bundle failed validation.
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return nil, err
		}
		if err := cmd.Start(); err != nil {
			return nil, err
		}
		// Ensure output conforms to the Output spec.
		dec := json.NewDecoder(stdout)
		dec.DisallowUnknownFields()

		if err := dec.Decode(&manifestresults[i]); err != nil {
			fmt.Printf("decode failed: %v\n", err)
			return nil, err
		}
		if err := cmd.Wait(); err != nil {
			return nil, err
		}
	}
	return manifestresults, nil
}
