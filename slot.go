package main

// ハードウェアに依存しない部分（図柄の描画、リール、役判定、ゲーム進行）。
// machine / ili9341 を import しないので、ホストの Go でもテスト・画像出力できる。

import (
	_ "embed"
	"image/color"
	"strconv"
	"time"

	"tinygo.org/x/drivers"
	"tinygo.org/x/drivers/pixel"
	"tinygo.org/x/tinyfont"
	"tinygo.org/x/tinyfont/freesans"
)

const (
	screenW = 320
	screenH = 240

	symW        = 84
	symH        = 50
	visibleRows = 3
	reelH       = symH * visibleRows
	reelY       = 42

	spinSpeed   = 12 // 1フレームで進むピクセル数
	frameTime   = 30 * time.Millisecond
	stopLockout = 300 * time.Millisecond

	bet         = 3
	startCredit = 50

	msgY     = 196
	msgH     = 23
	labelY   = 221
	labelH   = 19
	markerCY = 117
)

var reelX = [3]int16{26, 118, 210}

// 図柄
const (
	symSeven uint8 = iota
	symBar
	symBell  // 見た目は茶色の Gopher（img/Gogophercolor.png）
	symGrape // 見た目は青い Gopher（img/gopher.svg）
	symCherry
	numSymbols
)

var strips = [3][]uint8{
	{symSeven, symGrape, symCherry, symBell, symGrape, symBar, symCherry, symGrape, symBell, symGrape, symCherry, symBell},
	{symSeven, symBell, symGrape, symBar, symGrape, symBell, symCherry, symGrape, symBell, symGrape, symBar, symGrape},
	{symSeven, symGrape, symBell, symGrape, symBar, symBell, symGrape, symCherry, symBell, symGrape, symBar, symGrape},
}

// 色
var (
	colBG      = color.RGBA{18, 18, 40, 255}
	colFrame   = color.RGBA{200, 170, 60, 255}
	colCell    = color.RGBA{250, 246, 232, 255}
	colDivider = color.RGBA{215, 210, 195, 255}
	colLine    = color.RGBA{230, 30, 30, 255}
	colYellow  = color.RGBA{255, 220, 40, 255}
	colRed     = color.RGBA{220, 30, 30, 255}
	colGray    = color.RGBA{90, 90, 110, 255}
	colWhite   = color.RGBA{255, 255, 255, 255}
	colBlack   = color.RGBA{0, 0, 0, 255}

	colDarkRed     = color.RGBA{120, 0, 0, 255}
	colGold        = color.RGBA{255, 190, 0, 255}
	colOrange      = color.RGBA{230, 110, 0, 255}
	colPurple      = color.RGBA{120, 40, 170, 255}
	colLightPurple = color.RGBA{200, 150, 230, 255}
	colGreen       = color.RGBA{40, 150, 50, 255}
)

// screen は ili9341.Device のうち、このゲームが使うメソッドだけを抜き出したもの。
type screen interface {
	drivers.Displayer
	FillRectangle(x, y, width, height int16, c color.RGBA) error
	DrawRectangle(x, y, w, h int16, c color.RGBA) error
	DrawFastVLine(x, y0, y1 int16, c color.RGBA) error
	DrawBitmap(x, y int16, bitmap pixel.Image[pixel.RGB565BE]) error
}

// ---- 図柄の事前描画 ----

// canvas は pixel.Image に描くための drivers.Displayer。tinyfont からも使える。
type canvas struct {
	img pixel.Image[pixel.RGB565BE]
}

func (c canvas) Size() (int16, int16) {
	w, h := c.img.Size()
	return int16(w), int16(h)
}

func (c canvas) SetPixel(x, y int16, col color.RGBA) {
	w, h := c.img.Size()
	if x < 0 || y < 0 || int(x) >= w || int(y) >= h {
		return
	}
	c.img.Set(int(x), int(y), pixel.NewColor[pixel.RGB565BE](col.R, col.G, col.B))
}

func (c canvas) Display() error { return nil }

func (c canvas) fillRect(x, y, w, h int, col color.RGBA) {
	for yy := y; yy < y+h; yy++ {
		for xx := x; xx < x+w; xx++ {
			c.SetPixel(int16(xx), int16(yy), col)
		}
	}
}

func (c canvas) fillCircle(cx, cy, r int, col color.RGBA) {
	for dy := -r; dy <= r; dy++ {
		for dx := -r; dx <= r; dx++ {
			if dx*dx+dy*dy <= r*r {
				c.SetPixel(int16(cx+dx), int16(cy+dy), col)
			}
		}
	}
}

// line は半径 r の円を線上に並べて太線を描く。
func (c canvas) line(x0, y0, x1, y1, r int, col color.RGBA) {
	dx, dy := x1-x0, y1-y0
	steps := max(abs(dx), abs(dy))
	if steps == 0 {
		c.fillCircle(x0, y0, r, col)
		return
	}
	for i := 0; i <= steps; i++ {
		c.fillCircle(x0+dx*i/steps, y0+dy*i/steps, r, col)
	}
}

func (c canvas) textCentered(font tinyfont.Fonter, cx, baseline int, s string, col color.RGBA) {
	_, w := tinyfont.LineWidth(font, s)
	tinyfont.WriteLine(c, font, int16(cx-int(w)/2), int16(baseline), s, col)
}

// Gopher のスプライト。tools/gensprite で作る（形式は [幅, 高さ] + RGBA）。
// string で埋め込むと TinyGo では flash に置かれ、RAM を使わない。
//
//go:generate rsvg-convert -h 880 img/gopher.svg -o img/gopher_svg.png
//go:generate go run ./tools/gensprite -h 44 -o img/gopher_blue.rgba img/gopher_svg.png
//go:generate go run ./tools/gensprite -h 44 -o img/gopher_brown.rgba img/Gogophercolor.png
var (
	//go:embed img/gopher_blue.rgba
	gopherBlue string
	//go:embed img/gopher_brown.rgba
	gopherBrown string
)

// sprite はスプライトを中心 x=cx、上端 y=top に、コマ地の色と合成して描く。
func (c canvas) sprite(cx, top int, data string) {
	w, h := int(data[0]), int(data[1])
	px := data[2:]
	x0 := cx - w/2
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := (y*w + x) * 4
			a := int(px[i+3])
			if a == 0 {
				continue
			}
			c.SetPixel(int16(x0+x), int16(top+y), color.RGBA{
				blend(px[i], colCell.R, a),
				blend(px[i+1], colCell.G, a),
				blend(px[i+2], colCell.B, a),
				255,
			})
		}
	}
}

func blend(fg, bg uint8, a int) uint8 {
	return uint8((int(fg)*a + int(bg)*(255-a) + 127) / 255)
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func makeSymbols() [numSymbols]pixel.Image[pixel.RGB565BE] {
	var imgs [numSymbols]pixel.Image[pixel.RGB565BE]
	for i := range imgs {
		imgs[i] = pixel.NewImage[pixel.RGB565BE](symW, symH)
		c := canvas{imgs[i]}
		c.fillRect(0, 0, symW, symH, colCell)
		c.fillRect(0, symH-1, symW, 1, colDivider)
		drawSymbol(c, uint8(i))
	}
	return imgs
}

func drawSymbol(c canvas, sym uint8) {
	switch sym {
	case symSeven:
		c.textCentered(&freesans.Bold24pt7b, 44, 44, "7", colDarkRed)
		c.textCentered(&freesans.Bold24pt7b, 42, 42, "7", colRed)
	case symBar:
		c.fillRect(8, 11, 68, 28, colBlack)
		c.fillRect(10, 13, 64, 2, colWhite)
		c.fillRect(10, 35, 64, 2, colWhite)
		c.textCentered(&freesans.Bold12pt7b, 42, 33, "BAR", colWhite)
	case symBell:
		c.sprite(42, 3, gopherBrown)
	case symGrape:
		c.sprite(42, 3, gopherBlue)
	case symCherry:
		c.line(30, 28, 46, 7, 1, colGreen)
		c.line(54, 26, 46, 7, 1, colGreen)
		c.fillCircle(53, 8, 4, colGreen)
		c.fillCircle(30, 36, 9, colRed)
		c.fillCircle(54, 34, 9, colRed)
		c.fillCircle(27, 33, 2, colWhite)
		c.fillCircle(51, 31, 2, colWhite)
	}
}

// ---- リール ----

type reel struct {
	strip    []uint8 // リール配列
	offset   int     // 0 .. len(strip)*symH-1。増えると図柄が下に流れる
	spinning bool
	stopping bool
	remain   int // 停止までに進む残りピクセル
}

func (r *reel) total() int { return len(r.strip) * symH }

func (r *reel) start() {
	r.spinning = true
	r.stopping = false
	r.remain = 0
}

// requestStop は次のコマ境界で止まるよう停止要求を出す（最大1コマ滑り）。
func (r *reel) requestStop() {
	if !r.spinning || r.stopping {
		return
	}
	r.stopping = true
	r.remain = (symH - r.offset%symH) % symH
}

// update は1フレーム分進め、描き直しが必要なら true を返す。
func (r *reel) update() bool {
	if !r.spinning {
		return false
	}
	step := spinSpeed
	if r.stopping {
		step = min(spinSpeed, r.remain)
		r.remain -= step
	}
	r.offset = (r.offset + step) % r.total()
	if r.stopping && r.remain == 0 {
		r.spinning = false
		r.stopping = false
	}
	return true
}

// center は中段の図柄を返す（停止中のみ意味がある）。
func (r *reel) center() uint8 {
	t := r.total()
	return r.strip[((symH-r.offset)%t+t)%t/symH]
}

// render はリールの見えている部分を buf (84×150) に描く。
func (r *reel) render(buf pixel.Image[pixel.RGB565BE], syms *[numSymbols]pixel.Image[pixel.RGB565BE]) {
	const rowBytes = symW * 2
	dst := buf.RawBuffer()
	t := r.total()
	for y := 0; y < reelH; y++ {
		s := ((y-r.offset)%t + t) % t
		src := syms[r.strip[s/symH]].RawBuffer()
		row := s % symH
		copy(dst[y*rowBytes:(y+1)*rowBytes], src[row*rowBytes:(row+1)*rowBytes])
	}
	v := pixel.NewColor[pixel.RGB565BE](colLine.R, colLine.G, colLine.B)
	for _, y := range [2]int{symH - 1, symH * 2} {
		line := dst[y*rowBytes : (y+1)*rowBytes]
		for x := 0; x < rowBytes; x += 2 {
			line[x] = byte(v)
			line[x+1] = byte(v >> 8)
		}
	}
}

// ---- 役判定 ----

// judge は中段の3図柄から配当とメッセージを返す。上から順に最初に当てはまった役のみ。
func judge(l, c, r uint8) (int, string) {
	switch {
	case l == symSeven && c == symSeven && r == symSeven:
		return 100, "BIG BONUS!! +100"
	case l == symBar && c == symBar && r == symBar:
		return 50, "BAR BAR BAR! +50"
	case l == symBell && c == symBell && r == symBell:
		return 15, "BROWN GOPHER +15"
	case l == symCherry && c == symCherry && r == symCherry:
		return 10, "CHERRY x3 +10"
	case l == symGrape && c == symGrape && r == symGrape:
		return 8, "BLUE GOPHER +8"
	case l == symCherry:
		return 2, "CHERRY +2"
	}
	return 0, "PRESS ANY BUTTON"
}

// ---- ゲーム ----

const (
	stateIdle = iota
	stateSpinning
	stateGameOver
)

type game struct {
	scr       screen
	syms      [numSymbols]pixel.Image[pixel.RGB565BE]
	reelImg   pixel.Image[pixel.RGB565BE]
	reels     [3]reel
	state     int
	credit    int
	spinStart time.Time
}

func newGame(scr screen) *game {
	g := &game{scr: scr, credit: startCredit}
	g.syms = makeSymbols()
	g.reelImg = pixel.NewImage[pixel.RGB565BE](symW, reelH)
	for i := range g.reels {
		g.reels[i].strip = strips[i]
	}
	g.drawStatic()
	g.drawCredit()
	for i := range g.reels {
		g.drawReel(i)
		g.drawLabel(i)
	}
	g.drawMessage("PRESS ANY BUTTON", colYellow)
	return g
}

// edges は押された瞬間（前フレームで離されていて、今フレームで押されている）だけを true にする。
// フレーム周期でサンプリングするので、チャタリング対策も兼ねる。
func edges(down [3]bool, prev *[3]bool) [3]bool {
	var pressed [3]bool
	for i := range down {
		pressed[i] = down[i] && !prev[i]
	}
	*prev = down
	return pressed
}

// step は1フレーム分の処理。pressed は各ボタンがこのフレームで押された瞬間かどうか。
func (g *game) step(pressed [3]bool, now time.Time) {
	anyPressed := pressed[0] || pressed[1] || pressed[2]
	switch g.state {
	case stateIdle:
		if !anyPressed {
			return
		}
		g.credit -= bet
		g.drawCredit()
		g.clearMessage()
		g.spinStart = now
		for i := range g.reels {
			g.reels[i].start()
			g.drawLabel(i)
		}
		g.state = stateSpinning

	case stateSpinning:
		if now.Sub(g.spinStart) >= stopLockout {
			for i, p := range pressed {
				if p {
					g.reels[i].requestStop()
				}
			}
		}
		allStopped := true
		for i := range g.reels {
			r := &g.reels[i]
			wasSpinning := r.spinning
			if r.update() {
				g.drawReel(i)
			}
			if wasSpinning && !r.spinning {
				g.drawLabel(i)
			}
			if r.spinning {
				allStopped = false
			}
		}
		if allStopped {
			g.finishSpin()
		}

	case stateGameOver:
		if !anyPressed {
			return
		}
		g.credit = startCredit
		g.drawCredit()
		g.drawMessage("PRESS ANY BUTTON", colYellow)
		g.state = stateIdle
	}
}

func (g *game) finishSpin() {
	win, msg := judge(g.reels[0].center(), g.reels[1].center(), g.reels[2].center())
	g.credit += win
	g.drawCredit()
	switch {
	case g.credit < bet:
		g.drawMessage("GAME OVER - PRESS ANY BUTTON", colRed)
		g.state = stateGameOver
	case win > 0:
		g.drawMessage(msg, colYellow)
		g.state = stateIdle
	default:
		g.drawMessage(msg, colWhite)
		g.state = stateIdle
	}
}

// ---- 画面描画 ----

func (g *game) drawStatic() {
	s := g.scr
	s.FillRectangle(0, 0, screenW, screenH, colBG)
	tinyfont.WriteLine(s, &freesans.Bold12pt7b, 8, 26, "SLOT", colYellow)
	for _, x := range reelX {
		s.DrawRectangle(x-1, reelY-1, symW+2, reelH+2, colFrame)
		s.DrawRectangle(x-2, reelY-2, symW+4, reelH+4, colFrame)
	}
	// 有効ラインを指す内向きの三角マーカー
	for i := int16(0); i < 10; i++ {
		h := 9 - i
		s.DrawFastVLine(8+i, markerCY-h, markerCY+h, colLine)
		s.DrawFastVLine(311-i, markerCY-h, markerCY+h, colLine)
	}
}

func (g *game) drawCredit() {
	s := g.scr
	s.FillRectangle(150, 4, 170, 30, colBG)
	str := "CREDIT " + strconv.Itoa(g.credit)
	_, w := tinyfont.LineWidth(&freesans.Bold12pt7b, str)
	tinyfont.WriteLine(s, &freesans.Bold12pt7b, 312-int16(w), 26, str, colWhite)
}

func (g *game) clearMessage() {
	g.scr.FillRectangle(0, msgY, screenW, msgH, colBG)
}

func (g *game) drawMessage(msg string, col color.RGBA) {
	g.clearMessage()
	_, w := tinyfont.LineWidth(&freesans.Bold9pt7b, msg)
	tinyfont.WriteLine(g.scr, &freesans.Bold9pt7b, (screenW-int16(w))/2, 213, msg, col)
}

func (g *game) drawReel(i int) {
	g.reels[i].render(g.reelImg, &g.syms)
	g.scr.DrawBitmap(reelX[i], reelY, g.reelImg)
}

func (g *game) drawLabel(i int) {
	bg, fg := colGray, colBG
	if g.reels[i].spinning {
		bg, fg = colRed, colWhite
	}
	x := reelX[i]
	g.scr.FillRectangle(x, labelY, symW, labelH, bg)
	_, w := tinyfont.LineWidth(&freesans.Bold9pt7b, "STOP")
	tinyfont.WriteLine(g.scr, &freesans.Bold9pt7b, x+(symW-int16(w))/2, 236, "STOP", fg)
}
