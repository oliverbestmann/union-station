//go:build !(js && wasm)

package main

import (
	"github.com/pkg/profile"
)

func ProfileCPU() func() {
	return profile.Start(profile.CPUProfile).Stop
}

type IdleSuspend struct {
}

func (i *IdleSuspend) MaybeSuspend() {
	// do nothing, we don't need to suspend
}
