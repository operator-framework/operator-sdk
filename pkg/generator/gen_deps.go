package generator

import "io"

func renderGopkgTomlFile(w io.Writer) error {
	_, err := w.Write([]byte(gopkgTomlTmpl))
	return err
}

func renderGopkgLockFile(w io.Writer) error {
	_, err := w.Write([]byte(gopkgLockTmpl))
	return err
}
