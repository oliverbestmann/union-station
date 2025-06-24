//go:build !(js && wasm)

package fetch

import "io"

func Fetch(url string) io.ReadCloser {
	panic("not implemented")
}
