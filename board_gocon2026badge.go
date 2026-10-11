//go:build waveshare_rp2040_zero && gocon2026badge

package main

// gocon2026badge（RP2040-Zero + ST7789 1.54" 240×240）。
// ビルド: tinygo flash -target=waveshare-rp2040-zero -tags gocon2026badge .

import (
	"machine"

	"tinygo.org/x/drivers/st7789"
)

// 液晶の色反転。色が反転して見えたら false にする。
const displayInversion = true

// BTN_A だけで遊ぶ。押すたびに左のリールから順に止まる。
// 3 ボタンで左・中・右を別々に止めるときは、十字キーの左 / 下 / 右のピンを並べる（右は GP4。
// 回路図と実機でピンが違うことがあるので、tools/pinscan で確かめてから指定する）。
var buttons = []machine.Pin{machine.GP3}

// BTN_UP（実機では GP28）をダブルクリックすると隠しルーレットに入る（もう一度ダブルクリックで戻る）。
var secretButton = machine.GP28

func initBoard() (screen, layout) {
	return st7789Board{
		rst:      machine.GP15,
		dc:       machine.GP14,
		cs:       machine.GP13,
		bl:       machine.GP12,
		rotation: st7789.ROTATION_90,
		unused: []machine.Pin{
			machine.GP0, // PCM510x DIN
			machine.GP1, // PCM510x BCK
			machine.GP2, // PCM510x LRCK
			machine.GP9, // WS2812B × 16
		},
	}.init()
}
