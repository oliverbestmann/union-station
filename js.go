//go:build js && wasm

package main

import (
	"syscall/js"
)

func ProfileCPU() func() {
	return func() {}
}

type IdleSuspend struct {
	/// the timeRemaining function given by the previous idleCallback
	timeRemaining func() float64
}

func (i *IdleSuspend) MaybeSuspend() {
	if i.timeRemaining != nil && i.timeRemaining() >= 1 {
		// there is still at least another milliseconds of time left
		return
	}

	// suspend and ask for another idle callback
	i.timeRemaining = requestIdleCallback()
}

func requestIdleCallback() func() float64 {
	syncCh := make(chan func() float64)

	handler := js.FuncOf(func(this js.Value, args []js.Value) any {
		deadline := args[0]
		syncCh <- func() float64 {
			return deadline.Call("timeRemaining").Float()
		}

		return nil
	})

	js.Global().Get("window").Call("requestIdleCallback", handler)

	return <-syncCh
}
