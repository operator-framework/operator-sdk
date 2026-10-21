// Copyright 2026 The Operator-SDK Authors
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

package bundle

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/operator-framework/operator-sdk/internal/scorecard"
)

func TestHasScorecardConfig(t *testing.T) {
	inputDir := t.TempDir()
	outputDir := t.TempDir()

	if hasScorecardConfig(inputDir, outputDir) {
		t.Fatal("expected no scorecard config in empty directories")
	}

	// The config is written to the output directory when manifests come from --input-dir.
	cfgDir := filepath.Join(outputDir, filepath.FromSlash(scorecard.DefaultConfigDir))
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, scorecard.ConfigFileName), []byte("kind: Configuration\n"), 0666); err != nil {
		t.Fatal(err)
	}

	if !hasScorecardConfig(inputDir, outputDir) {
		t.Fatal("expected scorecard config to be found in the output directory")
	}
	if hasScorecardConfig(inputDir, "") {
		t.Fatal("expected no scorecard config when only the empty input directory is checked")
	}
}
