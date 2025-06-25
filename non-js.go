//go:build !(js && wasm)

package main

import (
	"github.com/pkg/profile"
)

var Debug = true

func ProfileStart() func() {
	return profile.Start(profile.CPUProfile).Stop
}

type IdleSuspend struct {
}

func (i *IdleSuspend) MaybeSuspend() {
	// do nothing, we don't need to suspend
}

func PlayerName() string {
	return "Hopfenherrscher"
}
