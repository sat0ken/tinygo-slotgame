//go:build tinygo

package main

// メインループ。ボードごとの初期化は board_*.go、ゲーム本体は slot.go にある。

import (
	"time"
)

func main() {
	display, lay := initBoard()
	g := newGame(display, lay)
	// ボタンが 1 つだけのボードは、押すたびに左のリールから順に止める。
	g.oneButton = len(buttons) == 1

	var prev [3]bool
	for {
		start := time.Now()

		var down [3]bool // 1 ボタンのときは down[0] だけを使う
		for i, b := range buttons {
			down[i] = !b.Get() // どのボードも押すと Low
		}
		g.step(edges(down, &prev), start)

		if d := frameTime - time.Since(start); d > 0 {
			time.Sleep(d)
		}
	}
}
