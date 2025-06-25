//go:build js && wasm

package fetch

import (
	"io"
	"syscall/js"
	"time"
)

var fn js.Value

func init() {
	fn = js.Global().Call("eval", `
		async (url, method, write) => {
			const resp = await fetch(url, {method: method});
            
            for await (const chunk of resp.body) {
                write(chunk);
            }
            
            write(null);
		}
	`)
}

func Get(url string) io.Reader {
	return fetch(url, "GET")
}

func Post(url string) io.Reader {
	return fetch(url, "POST")
}

func fetch(url, method string) io.Reader {
	read, write := io.Pipe()

	// chunks received and ready for send out
	var chunks [][]byte

	receive := js.FuncOf(func(this js.Value, args []js.Value) any {
		chunk := args[0]
		if chunk.IsNull() {
			chunks = append(chunks, nil)

			return nil
		}

		length := chunk.Get("length").Int()
		buf := make([]byte, length)
		js.CopyBytesToGo(buf, chunk)

		// queue this chunk for writing in another go routine
		chunks = append(chunks, buf)

		return nil
	})

	go fn.Invoke(url, method, receive)

	go func() {
		defer func() { _ = write.Close() }()
		defer func() { receive.Release() }()

		for {
			for len(chunks) > 0 {
				chunk := chunks[0]
				chunks = chunks[1:]

				if chunk == nil {
					break
				}

				_, _ = write.Write(chunk)
			}

			// check again for more data soon
			time.Sleep(100 * time.Millisecond)
		}
	}()

	return read
}
