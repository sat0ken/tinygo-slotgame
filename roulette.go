package main

// 隠しルーレット。停止中に隠しボタン（gocon2026badge の Up）を 2 回続けて押すと、
// 図柄がすべてお酒のリールに切り替わる。もう一度 2 回押すと通常のスロットに戻る。
// 中段に同じお酒が 3 つ揃えばそのお酒が当たり、それ以外はハズレ（水）。

import (
	"image/color"
	"strconv"
	"time"

	"tinygo.org/x/tinyfont/proggy"
)

// 2 回目の押下をダブルクリックとみなす間隔。
const doubleClickTime = time.Second

// お酒の図柄。drinkWater はハズレの図柄。
const (
	drinkBeer uint8 = iota
	drinkSake
	drinkShochu
	drinkWhisky
	drinkWine
	drinkCocktail
	drinkHoppy
	drinkWater
	numDrinks
)

// drinks は各図柄の名前（コマの下に出す）と、3 つ揃ったときの配当。
var drinks = [numDrinks]struct {
	name string
	pay  int
}{
	drinkBeer:     {"BEER", 10},
	drinkSake:     {"NIHONSHU", 20},
	drinkShochu:   {"SHOCHU", 20},
	drinkWhisky:   {"WHISKY", 30},
	drinkWine:     {"WINE", 50},
	drinkCocktail: {"COCKTAIL", 15},
	drinkHoppy:    {"HOPPY", 100},
	drinkWater:    {"WATER", 0},
}

// 各リール 12 コマ。お酒は 1 つずつ、残りの 5 コマが水。
var drinkStrips = [3][]uint8{
	{drinkBeer, drinkWater, drinkSake, drinkShochu, drinkWater, drinkWhisky, drinkWine, drinkWater, drinkCocktail, drinkWater, drinkHoppy, drinkWater},
	{drinkBeer, drinkWater, drinkWine, drinkHoppy, drinkWater, drinkSake, drinkCocktail, drinkWater, drinkShochu, drinkWhisky, drinkWater, drinkWater},
	{drinkBeer, drinkWater, drinkCocktail, drinkWater, drinkWhisky, drinkSake, drinkWater, drinkHoppy, drinkShochu, drinkWater, drinkWine, drinkWater},
}

var rouletteSet = symbolSet{
	title:  "BAR",
	ready:  "SAKE ROULETTE!",
	count:  int(numDrinks),
	strips: drinkStrips,
	draw:   drawDrink,
	judge:  judgeDrinks,
}

// judgeDrinks は中段に同じお酒が 3 つ揃ったときだけ当たりにする。
func judgeDrinks(l, c, r uint8) (int, string) {
	if l == c && c == r && l != drinkWater {
		d := drinks[l]
		return d.pay, d.name + "! +" + strconv.Itoa(d.pay)
	}
	return 0, "Lose...You can't drink"
}

// secretPressed は隠しボタンが押された瞬間に呼ぶ。停止中に doubleClickTime 以内に
// 2 回押されたら、通常のスロットと隠しルーレットを切り替える。
func (g *game) secretPressed(now time.Time) {
	if g.state != stateIdle {
		g.lastSecret = time.Time{}
		return
	}
	if !g.lastSecret.IsZero() && now.Sub(g.lastSecret) <= doubleClickTime {
		g.lastSecret = time.Time{}
		if g.set == &rouletteSet {
			g.useSet(&slotSet)
		} else {
			g.useSet(&rouletteSet)
		}
		return
	}
	g.lastSecret = now
}

var (
	colGlass   = color.RGBA{165, 170, 180, 255}
	colGlassIn = color.RGBA{236, 240, 245, 255}
	colFoam    = color.RGBA{255, 255, 255, 255}
	colFoamEdg = color.RGBA{200, 195, 185, 255}
	colBeer    = color.RGBA{245, 175, 30, 255}
	colBeerHi  = color.RGBA{255, 220, 120, 255}
	colIndigo  = color.RGBA{40, 60, 140, 255}
	colPorcel  = color.RGBA{242, 246, 252, 255}
	colBottleG = color.RGBA{40, 110, 60, 255}
	colBottleB = color.RGBA{115, 60, 20, 255}
	colCap     = color.RGBA{200, 170, 60, 255}
	colWhisky  = color.RGBA{200, 115, 25, 255}
	colIce     = color.RGBA{250, 252, 255, 255}
	colIceEdge = color.RGBA{180, 195, 210, 255}
	colWine    = color.RGBA{140, 20, 55, 255}
	colPink    = color.RGBA{240, 90, 150, 255}
	colOlive   = color.RGBA{110, 150, 40, 255}
	colWater   = color.RGBA{60, 150, 230, 255}
	colWaterHi = color.RGBA{200, 230, 255, 255}
	colHoppyBl = color.RGBA{30, 80, 170, 255}
	colLabel   = color.RGBA{90, 70, 50, 255}
)

// drawDrink はお酒の図柄を描く（幅 84 のコマ基準の座標）。上に絵、下に名前。
func drawDrink(c canvas, sym uint8) {
	switch sym {
	case drinkBeer: // ジョッキ
		c.fillRect(51, 15, 7, 3, colGlass)
		c.fillRect(55, 15, 3, 14, colGlass)
		c.fillRect(51, 26, 7, 3, colGlass)
		c.fillRect(29, 12, 22, 25, colBeer)
		c.fillRect(32, 16, 3, 18, colBeerHi)
		for _, p := range [3][2]int{{40, 22}, {45, 28}, {41, 32}} {
			c.fillCircle(p[0], p[1], 1, colBeerHi)
		}
		c.fillRect(27, 8, 26, 8, colFoamEdg)
		for _, p := range [4][3]int{{31, 8, 5}, {38, 6, 5}, {45, 7, 5}, {50, 9, 4}} {
			c.fillCircle(p[0], p[1], p[2], colFoamEdg)
		}
		c.fillRect(28, 9, 24, 6, colFoam)
		for _, p := range [4][3]int{{31, 8, 4}, {38, 6, 4}, {45, 7, 4}, {50, 9, 3}} {
			c.fillCircle(p[0], p[1], p[2], colFoam)
		}
	case drinkSake: // 徳利とお猪口
		c.fillRect(31, 4, 11, 3, colIndigo)
		c.fillRect(33, 6, 7, 10, colIndigo)
		c.line(36, 14, 36, 22, 5, colIndigo)
		c.fillCircle(36, 28, 10, colIndigo)
		c.fillCircle(32, 26, 2, colPorcel)
		c.fillRect(28, 31, 17, 2, colPorcel)
		c.fillRect(49, 28, 14, 10, colIndigo)
		c.fillRect(50, 29, 12, 8, colPorcel)
		c.fillCircle(56, 33, 2, colIndigo)
	case drinkShochu: // 一升瓶
		c.fillRect(38, 2, 9, 3, colCap)
		c.fillRect(39, 5, 7, 8, colBottleG)
		c.fillCircle(42, 18, 8, colBottleG)
		c.fillRect(34, 18, 17, 20, colBottleG)
		c.fillRect(36, 21, 13, 13, colPorcel)
		c.fillRect(41, 23, 3, 9, colRed)
	case drinkWhisky: // ロックグラス
		c.fillRect(28, 13, 28, 26, colGlass)
		c.fillRect(30, 13, 24, 23, colGlassIn)
		c.fillRect(30, 24, 24, 12, colWhisky)
		c.fillRect(31, 18, 11, 11, colIceEdge)
		c.fillRect(32, 19, 9, 9, colIce)
		c.fillRect(42, 21, 11, 11, colIceEdge)
		c.fillRect(43, 22, 9, 9, colIce)
	case drinkWine: // ワイングラス
		c.fillCircle(42, 14, 11, colGlass)
		c.fillCircle(42, 14, 10, colGlassIn)
		for dy := 0; dy <= 10; dy++ {
			for dx := -10; dx <= 10; dx++ {
				if dx*dx+dy*dy <= 100 {
					c.SetPixel(int16(42+dx), int16(16+dy-2), colWine)
				}
			}
		}
		c.fillRect(31, 2, 23, 4, colCell) // 飲み口を平らにする
		c.fillRect(41, 25, 3, 10, colGlass)
		c.fillRect(34, 35, 17, 3, colGlass)
	case drinkCocktail: // カクテルグラス
		c.line(53, 3, 45, 15, 0, colLabel)
		c.fillCircle(48, 10, 3, colOlive)
		for y := 8; y <= 24; y++ {
			hw := 15 * (24 - y) / 16
			c.fillRect(42-hw-1, y, 2*hw+3, 1, colGlass)
			if y >= 11 {
				c.fillRect(42-hw+1, y, 2*hw-1, 1, colPink)
			}
		}
		c.fillRect(41, 24, 3, 11, colGlass)
		c.fillRect(34, 35, 17, 3, colGlass)
	case drinkHoppy: // ホッピーの瓶
		c.fillRect(38, 4, 9, 3, colCap)
		c.fillRect(39, 7, 7, 7, colBottleB)
		c.fillCircle(42, 19, 8, colBottleB)
		c.fillRect(34, 19, 17, 19, colBottleB)
		c.fillRect(34, 22, 17, 10, colPorcel)
		c.fillRect(34, 25, 17, 4, colHoppyBl)
	case drinkWater: // 水滴
		for y := 6; y <= 22; y++ {
			hw := 10 * (y - 6) / 16
			c.fillRect(42-hw, y, 2*hw+1, 1, colWater)
		}
		c.fillCircle(42, 27, 10, colWater)
		c.fillCircle(38, 28, 3, colWaterHi)
	}
	c.textCentered(&proggy.TinySZ8pt7b, 42, 47, drinks[sym].name, colLabel)
}
