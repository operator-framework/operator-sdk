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

package ready

import (
	"os"
	"testing"
)

func TestFileReady(t *testing.T) {
	r := NewFileReady()
	err := r.Set()
	if err != nil {
		t.Errorf("could not set ready file: %v", err)
	}

	_, err = os.Stat(FileName)
	if err != nil {
		t.Errorf("did not find expected file at %s: %v", FileName, err)
	}

	err = r.Unset()
	if err != nil {
		t.Errorf("could not unset ready file: %v", err)
	}

	_, err = os.Stat(FileName)
	if err == nil {
		t.Errorf("file still exists at %s", FileName)
	}
	if !os.IsNotExist(err) {
		t.Errorf("error determining if file still exists at %s: %v", FileName, err)
	}
}
