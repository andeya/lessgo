package fileservice

import (
	"io"
	"os"
	"path/filepath"

	"github.com/lessgo/lessgo"
	"github.com/lessgo/lessgo/utils"
)

func SaveFile(ctx lessgo.Context, pname string, cover bool, newname ...string) (err error) {
	fh, err := ctx.FormFile(pname)
	if err != nil {
		return
	}
	f, err := fh.Open()
	if err != nil {
		return
	}
	defer func() {
		err2 := f.Close()
		if err2 != nil && err == nil {
			err = err2
		}
	}()
	var filename = fh.Filename
	if len(newname) > 0 {
		filename = newname[0]
	}
	if !cover && utils.FileExists(filename) {
		return
	}
	f2, _ := os.OpenFile(filepath.Join(UPLOAD_DIR, filename), os.O_CREATE|os.O_WRONLY, 0644)
	_, err = io.Copy(f2, f)
	defer func() {
		err3 := f2.Close()
		if err3 != nil && err == nil {
			err = err3
		}
	}()
	return
}
