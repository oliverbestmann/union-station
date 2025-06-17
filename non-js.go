//go:build !(js && wasm)

package main

type IdleSuspend struct {
}

func (i *IdleSuspend) MaybeSuspend() {
	// do nothing, we don't need to suspend
}
