//go:build js && wasm

package main

import (
	"math/rand/v2"
	"strings"
	"syscall/js"
)

func ProfileStart() func() {
	return func() {}
}

var Debug = false

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

func PlayerName() (name string) {
	defer func() { _ = recover() }()

	prefix := iff(rand.IntN(2) == 0, "Mr.", "Mrs.")
	candidate := prefix + " " + surnames[rand.IntN(len(surnames))]

	playerVar := js.Global().Call("GetPlayer", candidate)
	if !playerVar.IsNull() && !playerVar.IsUndefined() {
		player := strings.TrimSpace(playerVar.String())
		if player != "" {
			return player
		}
	}

	return candidate
}

var surnames = []string{
	"Bennett",
	"Pembroke",
	"Ashworth",
	"Hargreaves",
	"Middleton",
	"Montgomery",
	"Fairfax",
	"Blakemore",
	"Thorne",
	"Everly",
	"Clarke",
	"Wainwright",
	"Pritchard",
	"Ellis",
	"Chadwick",
	"Gresham",
	"Foster",
	"Holloway",
	"Templeton",
	"Redgrave",
	"Winthrop",
	"Hawthorne",
	"Linton",
	"Farrow",
	"Golding",
	"Fairclough",
	"Blackwood",
	"Stratton",
	"Dunmore",
	"Prescott",
	"Carrington",
	"Trevelyan",
	"Marlowe",
	"Bramhall",
	"Huxley",
	"Greaves",
	"Alderton",
	"Kingsley",
	"Hargrave",
	"Rowntree",
	"Whittington",
	"Amesbury",
	"Beckford",
	"Cavendish",
}
