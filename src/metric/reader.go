package metric

import (
	"io"
	"openwrt-diskio-api/src/model"
	"strings"

	"github.com/spf13/afero"
)

type FsReader struct {
	Fs    afero.Fs
	Paths model.ProcfsPathsInterface
}

func (r FsReader) ReadFile(path string) (string, error) {
	data, err := afero.ReadFile(r.Fs, path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}
func (r FsReader) Exists(path string) bool {
	ok, _ := afero.Exists(r.Fs, path)
	return ok
}
func (r FsReader) Open(path string) (io.ReadCloser, error) {
	return r.Fs.Open(path)
}

