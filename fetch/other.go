//go:build !(js && wasm)

package fetch

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
)

func Get(url string) io.Reader {
	panic("not implemented")
}

func Post(url string) io.Reader {
	req, _ := http.NewRequest("POST", url, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("[err] post failed: %s", err)
		return bytes.NewReader(nil)
	}

	body, _ := io.ReadAll(resp.Body)
	return bytes.NewReader(body)
}
