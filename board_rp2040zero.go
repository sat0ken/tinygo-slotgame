//go:build waveshare_rp2040_zero && !gocon2026badge

package main

// Waveshare RP2040-Zero + ST7789（240×240）の自作基板。

import (
	"machine"

	"tinygo.org/x/drivers/st7789"
)

// 液晶の色反転。ST7789 の IPS 液晶は反転 ON で正しい色になるものが多い。
// 色が反転して見えたら false にする。
const displayInversion = true

// A ボタン（SW1）だけで遊ぶ。押すたびに左のリールから順に止まる。
// 3 ボタンで左・中・右を別々に止めるときは {machine.GP3, machine.GP4, machine.GP5}（SW1 / SW2 / SW3）にする。
var buttons = []machine.Pin{machine.GP3}

func initBoard() (screen, layout) {
	return st7789Board{
		rst:      machine.GP9,
		dc:       machine.GP12,
		cs:       machine.GP13,
		bl:       machine.GP14,
		rotation: st7789.ROTATION_90,
		unused: []machine.Pin{
			machine.GP2,  // スピーカー
			machine.GP29, // WS2812B
		},
	}.init()
}
