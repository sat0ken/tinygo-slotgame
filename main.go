//go:build tinygo

package main

// メインループ。ボードごとの初期化は board_*.go、ゲーム本体は slot.go にある。

import (
	"time"
)

func main() {
	display := initBoard()
	g := newGame(display)

	var prev [3]bool
	for {
		start := time.Now()

		var down [3]bool
		for i, b := range buttons {
			down[i] = !b.Get() // どのボードも押すと Low
		}
		g.step(edges(down, &prev), start)

		if d := frameTime - time.Since(start); d > 0 {
			time.Sleep(d)
		}
	}
}
