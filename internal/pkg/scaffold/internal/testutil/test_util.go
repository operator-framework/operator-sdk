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
		if !info.IsDir() {
			b, err := afero.ReadFile(fromFS, path)
			if err != nil {
				return err
			}
			return afero.WriteFile(toFS, path, b, fileutil.DefaultFileMode)
		}
		return nil
	})
}
