package fetch

import (
	"io"
	"net/http"
	"os"
	"path"
)

// 不修改fetch的行为，重写fetch函数，要求使用defer机制关闭文件。
func Fetch(url string) (filename string, n int64, err error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()

	filename = path.Base(resp.Request.URL.Path)
	out, err := os.Create(filename)
	if err != nil {
		return "", 0, err
	}

	n, err = io.Copy(out, resp.Body)
	defer func() {
		if closeErr := out.Close(); err == nil {
			err = closeErr
		}
	}()
	return filename, n, nil
}
