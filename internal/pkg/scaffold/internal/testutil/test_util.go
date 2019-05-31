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

package test

import (
	"os"

	"github.com/operator-framework/operator-sdk/internal/util/fileutil"

	"github.com/spf13/afero"
)

func WriteOSPathToFS(fromFS, toFS afero.Fs, root string) error {
	if _, err := fromFS.Stat(root); err != nil {
		return err
	}
	return afero.Walk(fromFS, root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil {
			return err
		}
		// only copy non-dir and non-symlink files
		if !info.IsDir() && info.Mode()&os.ModeSymlink == 0 {
			b, err := afero.ReadFile(fromFS, path)
			if err != nil {
				return err
			}
			return afero.WriteFile(toFS, path, b, fileutil.DefaultFileMode)
		}
		return nil
	})
}
