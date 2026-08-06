package fetch

import (
	"io"
	"net/http"
	"os"
	"path"
)

// Fetch downloads the URL and returns the
// name and length of the local file.
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
