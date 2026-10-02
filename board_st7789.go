//go:build waveshare_rp2040_zero

package main

// RP2040-Zero + ST7789（240×240）の基板に共通の初期化。
// 液晶は SPI1（SCK=GP10, SDO=GP11）につなぐ。ピンの違いは各基板のファイルで指定する。

import (
	"machine"

	"tinygo.org/x/drivers/st7789"
)

type st7789Board struct {
	rst, dc, cs, bl machine.Pin
	rotation        st7789.Rotation
	unused          []machine.Pin // 使わない出力（スピーカー、LED など）。Low に固定する
}

func (b st7789Board) init() (screen, layout) {
	for _, p := range b.unused {
		p.Configure(machine.PinConfig{Mode: machine.PinOutput})
		p.Low()
	}

	// ボタンは押すと GND に落ちる。外部プルアップが無いので内部プルアップを使う。
	for _, p := range buttons {
		p.Configure(machine.PinConfig{Mode: machine.PinInputPullup})
	}

	// SPI1 の SDI（GP12）は液晶の制御線に使うので割り当てない。
	machine.SPI1.Configure(machine.SPIConfig{
		SCK:       machine.GP10,
		SDO:       machine.GP11,
		SDI:       machine.NoPin,
		Frequency: 40e6,
	})
	display := st7789.New(machine.SPI1, b.rst, b.dc, b.cs, b.bl) // BL は Configure の最後に点灯する
	display.Configure(st7789.Config{
		Width:    240,
		Height:   240,
		Rotation: b.rotation,
	})
	display.InvertColors(displayInversion)
	return &display, layoutSquare
}
