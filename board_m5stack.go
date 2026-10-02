//go:build m5stack

package main

// M5Stack Basic / Gray のハードウェア初期化。

import (
	"machine"

	"tinygo.org/x/drivers/ili9341"
)

// 液晶の色反転。色が反転して見えたら true にする（個体によって異なる）。
const displayInversion = false

// 左・中・右リールのストップボタン。本体前面の A / B / C がリールの真下に並ぶ。
var buttons = [3]machine.Pin{machine.BUTTON_A, machine.BUTTON_B, machine.BUTTON_C}

func initBoard() *ili9341.Device {
	// スピーカーは使わないが、浮いているとノイズが出るので Low に固定する。
	spk := machine.SPEAKER_PIN
	spk.Configure(machine.PinConfig{Mode: machine.PinOutput})
	spk.Low()

	// ボタンは基板側でプルアップ済み。押すと Low。
	for _, b := range buttons {
		b.Configure(machine.PinConfig{Mode: machine.PinInput})
	}

	machine.SPI2.Configure(machine.SPIConfig{
		SCK:       machine.LCD_SCK_PIN,
		SDO:       machine.LCD_SDO_PIN,
		SDI:       machine.LCD_SDI_PIN,
		Frequency: 40e6,
	})
	bl := machine.LCD_BL_PIN
	bl.Configure(machine.PinConfig{Mode: machine.PinOutput})

	display := ili9341.NewSPI(machine.SPI2, machine.LCD_DC_PIN, machine.LCD_SS_PIN, machine.LCD_RST_PIN)
	display.Configure(ili9341.Config{
		Width:            screenW,
		Height:           screenH,
		DisplayInversion: displayInversion,
	})
	display.SetRotation(ili9341.Rotation0Mirror) // M5Stack の横向き
	bl.High()
	return display
}
