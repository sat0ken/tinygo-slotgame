//go:build wioterminal

package main

// Wio Terminal のハードウェア初期化。

import (
	"machine"

	"tinygo.org/x/drivers/ili9341"
)

// 液晶の色反転。色が反転して見えたら true にする。
const displayInversion = false

// 左・中・右リールのストップボタン。本体上面のボタンは、画面側から見て
// 左から KEY_C / KEY_B / KEY_A の順に並ぶ。
var buttons = [3]machine.Pin{machine.WIO_KEY_C, machine.WIO_KEY_B, machine.WIO_KEY_A}

func initBoard() *ili9341.Device {
	// ブザーは使わないが、浮いているとノイズが出るので Low に固定する。
	bz := machine.WIO_BUZZER
	bz.Configure(machine.PinConfig{Mode: machine.PinOutput})
	bz.Low()

	// 押すと Low。内部プルアップを有効にする。
	for _, b := range buttons {
		b.Configure(machine.PinConfig{Mode: machine.PinInputPullup})
	}

	machine.SPI3.Configure(machine.SPIConfig{
		SCK:       machine.LCD_SCK_PIN,
		SDO:       machine.LCD_SDO_PIN,
		SDI:       machine.LCD_SDI_PIN,
		Frequency: 40e6,
	})
	bl := machine.LCD_BACKLIGHT
	bl.Configure(machine.PinConfig{Mode: machine.PinOutput})

	// 液晶はネイティブ 240×320 の縦長。Rotation270 で 320×240 の横向きにする。
	display := ili9341.NewSPI(machine.SPI3, machine.LCD_DC, machine.LCD_SS_PIN, machine.LCD_RESET)
	display.Configure(ili9341.Config{
		DisplayInversion: displayInversion,
	})
	display.SetRotation(ili9341.Rotation270)
	bl.High()
	return display
}
