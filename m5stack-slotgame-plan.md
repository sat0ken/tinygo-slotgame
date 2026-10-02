# M5Stack Basic スロットマシン 実装仕様書（TinyGo）

Oct 1, 2026 · @satoken

## 概要

M5Stack Basic 上で動く3リールのスロットマシンを TinyGo で実装する。本体の物理ボタン A / B / C が、そのまま左 / 中 / 右リールのストップボタンになる。

同じコードを Wio Terminal と、Waveshare RP2040-Zero に ST7789 240×240 液晶を載せた 2 種類の基板（自作基板、gocon2026badge）でも動かせる。ボードごとに違うのは初期化、ボタンの割り当て、画面幅に応じた横方向の配置（`layout`）だけで、ゲーム本体は共通（「Wio Terminal 対応」「Waveshare RP2040-Zero 対応」を参照）。

この仕様書は Claude Code に実装を任せるためのもの。ピン番号、ドライバ API、座標は TinyGo 0.42.0・drivers v0.36.0・tinyfont v0.7.0 のソースで確認済みで、この仕様で書いた試作コードは `tinygo build -target=m5stack` を通過している。

完成条件:

- `tinygo flash -target=m5stack` で書き込み、実機で遊べること
- ボタン A/B/C で対応するリールだけが止まり、タイミング次第で「7」を狙えること（目押し可能）
- 回転中のリールがちらつかず、滑らかに流れること（目安 30fps）
- 中段1ラインの役判定とクレジット管理が配当表どおりに動くこと

## 開発環境と依存

TinyGo 0.42.0 は Go 1.25〜1.27 を要求する。Go 1.24 以前では「requires go version 1.25 through 1.27」で失敗する。

| 項目 | 内容 |
| --- | --- |
| TinyGo | 0.42.0（確認済み） |
| Go | 1.26 系（1.26.8 で確認） |
| ターゲット | `m5stack`（ESP32、M5Stack Basic / Gray 共通）、`wioterminal`（ATSAMD51）、`waveshare-rp2040-zero`（RP2040） |
| ドライバ | `tinygo.org/x/drivers` v0.36.0（`ili9341`, `st7789`, `pixel`） |
| フォント | `tinygo.org/x/tinyfont` v0.7.0（`freesans`） |
| 標準ライブラリ | `image/color`, `machine`, `strconv`, `time` のみ。バイナリを小さく保つため `fmt` は使わない |

ファイル構成はリポジトリ直下に次のとおり。ハード依存部分とそれ以外を分け、ゲーム本体はホストの Go でもテストできるようにしている。

| ファイル | 内容 |
| --- | --- |
| `go.mod` / `go.sum` | モジュール `m5slot` |
| `main.go` | メインループ（`//go:build tinygo`）。`initBoard()` で液晶と `layout` を受け取り、ボタンを読んで `game.step` を 30ms ごとに呼ぶ |
| `board_m5stack.go` | M5Stack の初期化（`//go:build m5stack`）。`displayInversion`、`buttons`、`initBoard()` |
| `board_wioterminal.go` | Wio Terminal の初期化（`//go:build wioterminal`）。中身の構成は M5Stack と同じ |
| `board_st7789.go` | RP2040-Zero + ST7789 の基板に共通の初期化（`//go:build waveshare_rp2040_zero`）。`st7789Board` 型 |
| `board_rp2040zero.go` | RP2040-Zero 自作基板のピン（`//go:build waveshare_rp2040_zero && !gocon2026badge`） |
| `board_gocon2026badge.go` | gocon2026badge のピン（`//go:build waveshare_rp2040_zero && gocon2026badge`） |
| `slot.go` | ハードに依存しない部分。画面幅ごとの配置（`layout`）、図柄の事前描画、リール、停止ロジック、役判定、状態遷移、画面描画 |
| `slot_test.go` | ホストで動くテスト。`go test -run TestPreview -preview .` で画面イメージを PNG に書き出す |
| `tools/gensprite/` | 画像を図柄用スプライト（`.rgba`）に変換するホスト用ツール |
| `img/` | 生成済みの Gopher スプライト（`.rgba`）。元画像はリポジトリに含めない |

```
module m5slot

go 1.22

require (
	tinygo.org/x/drivers v0.36.0
	tinygo.org/x/tinyfont v0.7.0
)
```

ビルドと書き込み:

```
go mod tidy
# M5Stack
tinygo build -target=m5stack -size short -o slot_m5stack.bin .
tinygo flash -target=m5stack -port=/dev/ttyUSB0 .

# Wio Terminal
tinygo build -target=wioterminal -size short -o slot_wioterminal.bin .
tinygo flash -target=wioterminal .

# Waveshare RP2040-Zero 自作基板
tinygo build -target=waveshare-rp2040-zero -size short -o slot_waveshare-rp2040-zero.bin .
tinygo flash -target=waveshare-rp2040-zero .

# gocon2026badge（ターゲットは同じ RP2040-Zero。ビルドタグで基板を選ぶ）
tinygo build -target=waveshare-rp2040-zero -tags gocon2026badge -size short -o slot_gocon2026badge.bin .
tinygo flash -target=waveshare-rp2040-zero -tags gocon2026badge .
```

`-size short` の値（Gopher 図柄入り）は、M5Stack が flash 55,196 バイト / RAM 23,520 バイト、Wio Terminal が flash 51,692 バイト / RAM 26,160 バイト、RP2040-Zero の 2 基板がどちらも約 flash 54,370 バイト / RAM 24,796 バイト。ポート名は環境に合わせて変える（Linux の M5Stack では `/dev/ttyUSB0`、Wio Terminal では `/dev/ttyACM0` が多い）。Wio Terminal で書き込みに失敗するときは、電源スイッチを下に素早く 2 回スライドしてブートローダーに入れてから再実行する。RP2040-Zero で失敗するときは、BOOT ボタンを押しながら RESET を押して（または BOOT を押しながら USB を挿して）ブートローダーに入れる。

## ハードウェア仕様

ピンはすべて TinyGo の `machine` パッケージ（`board_m5stack.go`）の定数名で参照し、GPIO 番号を直書きしない。

| 用途 | 定数名 | GPIO | 備考 |
| --- | --- | --- | --- |
| ボタン A（左リール） | `machine.BUTTON_A` | 39 | 入力専用ピン。基板側でプルアップ済み、押すと Low |
| ボタン B（中リール） | `machine.BUTTON_B` | 38 | 同上 |
| ボタン C（右リール） | `machine.BUTTON_C` | 37 | 同上 |
| 液晶 SCK | `machine.LCD_SCK_PIN` | 18 | SPI |
| 液晶 SDO（MOSI） | `machine.LCD_SDO_PIN` | 23 | SPI |
| 液晶 SDI（MISO） | `machine.LCD_SDI_PIN` | 19 | SPI |
| 液晶 CS | `machine.LCD_SS_PIN` | 14 |  |
| 液晶 DC | `machine.LCD_DC_PIN` | 27 |  |
| 液晶 RST | `machine.LCD_RST_PIN` | 33 |  |
| バックライト | `machine.LCD_BL_PIN` | 32 | 出力 High で点灯 |
| スピーカー | `machine.SPEAKER_PIN` | 25 | 今回は音を出さない。浮いているとノイズが出るので出力 Low に固定する |

### 液晶の初期化

液晶は ILI9342C（320×240、横向きがネイティブ）。`ili9341` ドライバで動かし、手順は drivers 公式の `examples/ili9341/initdisplay/m5stack.go` と同じにする。

```go
machine.SPI2.Configure(machine.SPIConfig{
	SCK:       machine.LCD_SCK_PIN,
	SDO:       machine.LCD_SDO_PIN,
	SDI:       machine.LCD_SDI_PIN,
	Frequency: 40e6,
})
bl := machine.LCD_BL_PIN
bl.Configure(machine.PinConfig{Mode: machine.PinOutput})

display = ili9341.NewSPI(machine.SPI2, machine.LCD_DC_PIN, machine.LCD_SS_PIN, machine.LCD_RST_PIN)
display.Configure(ili9341.Config{
	Width:            320,
	Height:           240,
	DisplayInversion: displayInversion, // 定数。既定 false
})
display.SetRotation(ili9341.Rotation0Mirror) // M5Stack の横向き。MADCTL = BGR のみになる
bl.High()
```

- `Rotation0Mirror` が正解。`Rotation0` だと左右反転し、`Rotation90` だと縦長扱いになる。
- `DisplayInversion` は液晶の個体によって正解が異なる。手元の M5Stack Basic 実機では `false` で正しい色になった（`true` では背景が白っぽく、7 が水色になるなど全色が補色になる）。色が反転して見えたら切り替える。切り替えやすいようにファイル先頭の定数にする。
- SPI は `machine.SPI2` を使う。

### ボタン

`machine.PinInput` で設定し、`!pin.Get()` を「押されている」とする。外部プルアップがあるので内部プルアップは不要（GPIO37〜39 にはそもそも無い）。チャタリング対策は、フレーム周期（30ms）でのサンプリングと立ち下がり検出で兼ねる。

### Wio Terminal 対応

液晶は ILI9341（ネイティブ 240×320 の縦長）。drivers 公式の `examples/ili9341/initdisplay/wioterminal.go` と同じ手順で初期化し、`Rotation270` で 320×240 の横向きにする。`Size()` が 320×240 を返すので、画面レイアウトや座標は M5Stack と共通のまま使える。

| 用途 | 定数名 | ピン | 備考 |
| --- | --- | --- | --- |
| 左リール | `machine.WIO_KEY_C` | PC28 | 本体上面、画面側から見て左端。内部プルアップ、押すと Low |
| 中リール | `machine.WIO_KEY_B` | PC27 | 同上、中央 |
| 右リール | `machine.WIO_KEY_A` | PC26 | 同上、右端 |
| 液晶 SCK / SDO / SDI | `machine.LCD_SCK_PIN` / `LCD_SDO_PIN` / `LCD_SDI_PIN` | PB20 / PB19 / PB18 | `machine.SPI3` |
| 液晶 CS | `machine.LCD_SS_PIN` | PB21 |  |
| 液晶 DC | `machine.LCD_DC` | PC06 | M5Stack と定数名が違う |
| 液晶 RST | `machine.LCD_RESET` | PC07 | 同上 |
| バックライト | `machine.LCD_BACKLIGHT` | PC05 | 出力 High で点灯 |
| ブザー | `machine.WIO_BUZZER` | PD11 | 音は出さない。出力 Low に固定する |

- ボタンは `machine.PinInputPullup` で設定する。押したときの判定（`!pin.Get()`）は M5Stack と同じ。
- KEY_A〜C は、画面側から見て右から A / B / C の順に並ぶ。左・中・右リールに合わせるため `buttons` は `{WIO_KEY_C, WIO_KEY_B, WIO_KEY_A}` の順にする。
- ボタンが画面の上にあるので、STOP ラベルとボタンの位置は M5Stack ほど直感的には対応しない。ラベルは回転中かどうかの表示として残す。
- `displayInversion` は `false`。色が反転して見えたら `board_wioterminal.go` の定数を切り替える。
- 実機で動作確認済み。ボタンの割り当て（左から KEY_C / KEY_B / KEY_A）、色（`displayInversion` = `false`）、回転の滑らかさは M5Stack と同じく問題なし。

### Waveshare RP2040-Zero 対応

RP2040-Zero に ST7789（240×240）液晶とタクトスイッチを載せた基板を 2 種類サポートする。どちらもターゲットは `waveshare-rp2040-zero` なので、ビルドタグ `gocon2026badge` の有無で基板を選ぶ。

共通部分（`board_st7789.go`）:

- 液晶は SPI1（SCK=GP10、SDA/MOSI=GP11）。GP12 はどちらの基板でも液晶の制御線（DC または BL）に使うので、SPI の `SDI` は `machine.NoPin` にする。
- 基板ごとの違い（RESET / DC / CS / BL のピン、回転、使わないピン）は `st7789Board` 型で渡し、`init()` で初期化する。
- ボタンは押すと GND に落ちる。外部プルアップが無いので `machine.PinInputPullup` を使う。
- 使わない出力（スピーカー、DAC、WS2812B のデータ線）は出力 Low に固定する。

```go
machine.SPI1.Configure(machine.SPIConfig{
	SCK:       machine.GP10,
	SDO:       machine.GP11,
	SDI:       machine.NoPin,
	Frequency: 40e6,
})
display := st7789.New(machine.SPI1, b.rst, b.dc, b.cs, b.bl)
display.Configure(st7789.Config{Width: 240, Height: 240, Rotation: b.rotation})
display.InvertColors(displayInversion) // 既定 true
```

ピンの違い:

| 用途 | 自作基板（`board_rp2040zero.go`） | gocon2026badge（`board_gocon2026badge.go`） |
| --- | --- | --- |
| ストップボタン（1 ボタン） | GP3（SW1） | GP3（BTN_A） |
| 3 ボタンにする場合の左 / 中 / 右 | GP3 / GP4 / GP5（SW1 / SW2 / SW3） | GP28 / GP29 / GP4（十字キーの左 / 下 / 右） |
| 液晶 RESET | GP9 | GP15 |
| 液晶 DC | GP12 | GP14 |
| 液晶 CS | GP13 | GP13 |
| 液晶 BL | GP14 | GP12 |
| 回転 | `ROTATION_90` | `ROTATION_90` |
| Low に固定するピン | GP2（スピーカー）、GP29（WS2812B） | GP0〜GP2（PCM510x DIN / BCK / LRCK）、GP9（WS2812B × 16） |
| 使わないピン | SW4（GP6）、SW5（GP26）、SW6（GP15）、I2C（GP0/GP1） | BTN_UP（GP8）、BTN_A（GP3）、BTN_B（GP6）、I2C0（GP4/GP5）、I2C1（GP26/GP27） |

- どちらの基板も A ボタンだけで遊ぶ（1 ボタンモード、「操作」を参照）。`buttons` を 3 つにすれば、左・中・右を別々に止める遊び方に戻せる。
- `st7789` ドライバには `DrawRectangle` が無く、`DrawFastVLine` は error を返さない。どちらのドライバでも使えるよう、`screen` インターフェースは `FillRectangle` と `DrawBitmap` だけにし、枠と三角マーカーは `FillRectangle` で描く。
- `st7789.Configure` は常に反転 ON（INVON）にする。`displayInversion` で `InvertColors` を呼び直すので、色が反転して見えたら `false` にする。
- 回転を `ROTATION_180` / `ROTATION_270` にする場合は、240×240 の液晶ではドライバの `RowOffset: 80` が必要になる（液晶コントローラーのメモリが 240×320 のため）。
- 基板上の WS2812（GP16）は使わない。
- 画面が 240×240 なので、横方向は `layoutSquare` の配置を使う（「画面レイアウト」を参照）。
- どちらの基板も実機で表示を確認済み（`ROTATION_90`、`displayInversion` = `true`）。1 ボタンモードは実機では未確認。

## ゲーム仕様

乱数は使わない。リール配列は固定で、結果はプレイヤーがボタンを押したタイミングだけで決まる（目押し型）。

### 操作

1. 全リール停止中に A/B/C のどれかを押すとスタート。1ゲーム 3 クレジットを消費し、3本同時に回転を始める。
2. 回転中に A を押すと左、B で中、C で右のリールが止まる。止める順番は自由。
3. スタート直後 300ms はストップを受け付けない。スタートに使った押下や押し直しで、いきなり止まるのを防ぐ。
4. 3本とも止まったら中段1ラインで役を判定し、配当を加算して結果を表示する。
5. クレジットが 3 未満になったら GAME OVER。ボタンで 50 クレジットに戻して再開する。

ボタンは「押された瞬間」（前フレームで離されていて、今フレームで押されている）だけを入力として扱う。押しっぱなしで連続スタートしないようにするため。

#### 1 ボタンモード

ボードの `buttons` が 1 つだけのとき（RP2040-Zero の 2 基板は A ボタンだけ）は、`main.go` が `game.oneButton = true` にする。

- スタートは 3 ボタンのときと同じく、ボタンを押すと始まる。
- 回転中（開始から 300ms 以降）にボタンを押すたびに、左 → 中 → 右の順に 1 本ずつ停止要求を出す。次に止めるのは「回転中で停止要求の出ていない一番左のリール」（`nextReel()`）。前のリールが滑っている間に次を押してもよい。
- STOP ラベルは、次に止まるリールを赤地に白文字、順番待ちのリールを暗い赤 (120,0,0) 地に灰文字、停止要求を出したリールと停止中のリールを灰地にする。

### 状態遷移

| 状態 | 入力・条件 | 処理 | 次の状態 |
| --- | --- | --- | --- |
| `stateIdle` | いずれかのボタンが押された | クレジット -3、メッセージ消去、全リール回転開始、ラベルを赤に | `stateSpinning` |
| `stateSpinning` | 開始から 300ms 以降にボタン i が押された | リール i に停止要求 | `stateSpinning` |
| `stateSpinning`（1 ボタン） | 開始から 300ms 以降にボタンが押された | `nextReel()` に停止要求、ラベルを更新 | `stateSpinning` |
| `stateSpinning` | 全リール停止、配当加算後もクレジット ≥ 3 | 役判定、配当加算、結果メッセージ | `stateIdle` |
| `stateSpinning` | 全リール停止、配当加算後のクレジット < 3 | 「GAME OVER - PRESS ANY BUTTON」を赤で表示 | `stateGameOver` |
| `stateGameOver` | いずれかのボタンが押された | クレジットを 50 に戻す | `stateIdle` |

### 図柄とリール配列

図柄は 5 種類（`symSeven`, `symBar`, `symBell`, `symGrape`, `symCherry`）。`symBell` は茶色の Gopher、`symGrape` は青い Gopher の絵で表示する（定数名と表中の「ベル」「ブドウ」は役の名前として残している）。各リール 12 コマで、下表の上から順に帯状に並ぶ。各リールとも先頭が 7。

| コマ | 左リール | 中リール | 右リール |
| --- | --- | --- | --- |
| 0 | 7 | 7 | 7 |
| 1 | ブドウ | ベル | ブドウ |
| 2 | チェリー | ブドウ | ベル |
| 3 | ベル | BAR | ブドウ |
| 4 | ブドウ | ブドウ | BAR |
| 5 | BAR | ベル | ベル |
| 6 | チェリー | チェリー | ブドウ |
| 7 | ブドウ | ブドウ | チェリー |
| 8 | ベル | ベル | ベル |
| 9 | ブドウ | ブドウ | ブドウ |
| 10 | チェリー | BAR | BAR |
| 11 | ベル | ブドウ | ブドウ |

### 配当表（1ゲーム 3 クレジット、中段1ライン）

上から順に判定し、最初に当てはまった役だけを払い出す。チェリー3つ揃いは 10 だけで、左チェリーの 2 は加算しない。

| 役 | 配当 | メッセージ |
| --- | --- | --- |
| 7 ・ 7 ・ 7 | 100 | `BIG BONUS!! +100` |
| BAR ・ BAR ・ BAR | 50 | `BAR BAR BAR! +50` |
| ベル（茶色の Gopher）× 3 | 15 | `BROWN GOPHER +15` |
| チェリー × 3 | 10 | `CHERRY x3 +10` |
| ブドウ（青い Gopher）× 3 | 8 | `BLUE GOPHER +8` |
| 左リールのみチェリー | 2 | `CHERRY +2` |
| ハズレ | 0 | `PRESS ANY BUTTON`（白） |

当たり時のメッセージは黄色で表示する。

## 画面レイアウト

画面は 320×240。リールの中心 X（68, 160, 252）を物理ボタン A/B/C の真上に合わせ、「このボタンでこのリールが止まる」と一目で分かるようにする。

240×240 の画面（RP2040-Zero の 2 基板）でも縦方向の配置は同じ。横方向だけを `layout` で切り替える。

| 項目 | `layoutWide`（320×240） | `layoutSquare`（240×240） |
| --- | --- | --- |
| 1コマの幅 `symW` | 84 | 70 |
| リールの左端 `reelX` | 26 / 118 / 210 | 8 / 85 / 162 |
| 三角マーカー | x=8 から幅 10（高さ 19） | x=0 から幅 5（高さ 9） |
| GAME OVER のメッセージ | `GAME OVER - PRESS ANY BUTTON` | `GAME OVER`（長いメッセージは 315px で画面に収まらない） |

- 図柄は幅 84 のコマを基準にした座標で描き、`canvas.ox = symW/2 - 42` だけ横にずらして実際の幅に収める。
- BAR の黒枠はコマ幅 − 16（幅 84 で 68）。「BAR」を `Bold12pt7b` で描くと枠に収まらないときは `Bold9pt7b`（ベースライン y=31）にする。
- クレジットを消す範囲は `max(screenW-170, 「SLOT」の右端+4)` から画面右端まで。文字の右端は `screenW-8`。

```
y=0   +--------------------------------------------------+
      | SLOT                              CREDIT 50      |  ヘッダー (0-34)
y=42  |      +--------+  +--------+  +--------+          |
      |      | 上段   |  | 上段   |  | 上段   |          |
      |  ▶  |========|  |========|  |========|  ◀      |  中段 = 有効ライン
      |      | 下段   |  | 下段   |  | 下段   |          |
y=192 |      +--------+  +--------+  +--------+          |
      |              PRESS ANY BUTTON                    |  メッセージ (196-218)
y=221 |      [ STOP ]    [ STOP ]    [ STOP ]            |  ラベル (221-240)
      +--------------------------------------------------+
             ボタンA     ボタンB     ボタンC
```

### 座標

| 要素 | 位置・サイズ | 内容 |
| --- | --- | --- |
| タイトル | x=8, ベースライン y=26 | 「SLOT」、`freesans.Bold12pt7b`、黄 |
| クレジット | 右端 x=312 に右寄せ、ベースライン y=26 | 「CREDIT n」、`Bold12pt7b`、白。更新時は (150,4) から 170×30 を背景色で消してから描く |
| リール窓 | x = 26 / 118 / 210、y=42、各 84×150 | 1コマ 84×50 を 3 コマ分表示 |
| リール枠 | 各窓の外周 2px 分 | 金色の `DrawRectangle` を 2 重 |
| 有効ラインマーカー | 中心 y=117、左 x=8〜17 / 右 x=302〜311 | 内向きの赤い三角（高さ 19px）。`DrawFastVLine` を高さを減らしながら 10 本 |
| 有効ライン | リール内 y=49 と y=100 | 赤い 1px 線。リールバッファに直接描く |
| メッセージ | y=196〜218、中央寄せ、ベースライン y=213 | `Bold9pt7b` |
| STOP ラベル | 各リールと同じ x、y=221、84×19 | 回転中は赤地に白文字、停止中は灰地に背景色文字。ベースライン y=236 |

### 色（RGB）

| 名前 | 値 | 用途 |
| --- | --- | --- |
| 背景 | 18, 18, 40 | 画面全体 |
| 枠 | 200, 170, 60 | リール枠 |
| コマ地 | 250, 246, 232 | 図柄の背景 |
| コマ区切り | 215, 210, 195 | 各コマ最下行の 1px 線 |
| 有効ライン | 230, 30, 30 | 赤線・三角マーカー |
| 黄 | 255, 220, 40 | タイトル、当たりメッセージ |
| 赤 | 220, 30, 30 | 7、チェリー、STOP ラベル |
| 灰 | 90, 90, 110 | 停止中のラベル |

### 図柄デザイン（84×50 のコマ内座標）

7・BAR・チェリーは画像素材を使わず、円・矩形・太線と tinyfont で起動時に描く。ベルとブドウは Gopher の画像から作ったスプライトを、起動時にコマ地の色と合成して描く。

| 図柄 | 描き方 |
| --- | --- |
| 7 | `Bold24pt7b` の「7」を中央寄せ。影として暗い赤 (120,0,0) を中心 x=44・ベースライン y=44 に描き、その上に赤を x=42・y=42 に重ねる |
| BAR | 黒の矩形 (8,11) 68×28、内側上下に白線 (10,13) と (10,35)、各 64×2。「BAR」を `Bold12pt7b` の白で中心 x=42・ベースライン y=33 |
| ベル | 茶色の Gopher（`img/Gogophercolor.png`）。中心 x=42、上端 y=3、高さ 44px（33×44） |
| ブドウ | 青い Gopher（`img/gopher.svg`）。中心 x=42、上端 y=3、高さ 44px（32×44） |
| チェリー | 軸 (30,28)→(46,7) と (54,26)→(46,7)（緑、太さ半径 1）、葉 円(53,8) r4、実 円(30,36) と (54,34) r9（赤）、ハイライト (27,33) (51,31) r2（白） |

Gopher のスプライト:

- `tools/gensprite` が PNG の透明部分を切り落とし、高さ 44px に面積平均で縮小して `img/gopher_blue.rgba` / `img/gopher_brown.rgba` に書き出す。形式は [幅, 高さ] の 2 バイトに続いて各ピクセルの R, G, B, A（非乗算）。
- SVG は先に `rsvg-convert` で PNG にする。手順は `slot.go` の `//go:generate` にあり、`go generate` で作り直せる。生成済みの `.rgba` はリポジトリに含めるので、通常のビルドに `rsvg-convert` は不要。
- `//go:embed` で `string` として埋め込む。TinyGo では flash に置かれ、RAM を使わない。

## 実装設計

高速化の要点は「図柄は起動時に1回だけ描き、回転中は行コピーと一括転送だけにする」こと。`display.SetPixel` や `FillRectangle` を回転中に呼ぶとちらつき、遅くなる。

### 定数（ファイル先頭にまとめる）

| 定数 | 値 | 意味 |
| --- | --- | --- |
| `symH` | 50 | 1コマの高さ。幅 `symW` は `layout` ごとに決める（84 または 70） |
| `visibleRows` | 3 | 窓に見えるコマ数。`reelH = symH * visibleRows` = 150 |
| `reelY` | 42 | リール窓の上端 |
| `spinSpeed` | 12 | 1フレームで進むピクセル数。目押しの難易度を決める |
| `frameTime` | 30ms | 1フレームの目標時間 |
| `stopLockout` | 300ms | スタート直後にストップを無視する時間 |
| `bet` / `startCredit` | 3 / 50 |  |
| `displayInversion` | false | 液晶の色反転 |
| `layoutWide` / `layoutSquare` | 「画面レイアウト」を参照 | 画面幅ごとの横方向の配置。`initBoard()` が返す |

### 図柄の事前描画

- 図柄ごとに `pixel.NewImage[pixel.RGB565BE](symW, 50)` を作り、`[numSymbols]` の配列に保持する（幅 84 で計 42KB、幅 70 で計 35KB）。
- 画像に描くための `canvas` 型を作る。`drivers.Displayer`（`Size()`, `SetPixel()`, `Display() error`）を実装し、`tinyfont.WriteLine` からも文字を描けるようにする。`SetPixel` では範囲外を無視し、`pixel.NewColor[pixel.RGB565BE](r, g, b)` で変換して `img.Set` する。
- `canvas` には `fillRect`, `fillCircle`, `line`（半径付きの太線、円を線上に並べる）, `textCentered`（`tinyfont.LineWidth` の outbox 幅で中央寄せ）を持たせる。

### リールのデータ構造

```go
type reel struct {
	strip    []uint8 // リール配列
	offset   int     // 0 .. len(strip)*symH-1。増えると図柄が下に流れる
	spinning bool
	stopping bool
	remain   int // 停止までに進む残りピクセル
}
```

窓の y 行目（0〜149）に表示するのは、帯上の位置 `s = ((y - offset) mod total + total) mod total` で、図柄は `strip[s / symH]`、その中の行は `s % symH`。`offset` を増やすと、同じ y に帯の前の方が来るので図柄が下へ流れる。

### リールの描画

1. リール1本分の共有バッファ `reelImg = pixel.NewImage[pixel.RGB565BE](84, 150)` を起動時に1つだけ作る。
2. 各行 y について、該当図柄画像の `RawBuffer()` から 1行分（84×2 = 168 バイト）を `copy` する。
3. 有効ラインとして y=49 と y=100 の行を赤で塗る。`RGB565BE` の値 `v` はメモリ上 `byte(v), byte(v>>8)` の順に書く。
4. `display.DrawBitmap(reelX[i], reelY, reelImg)` で一括転送する。

描き直すのは、そのフレームで動いたリールだけ。`DrawRGBBitmap` は非推奨なので使わない。

### 停止ロジック（最大1コマ滑り）

- `requestStop()`: 回転中かつ未停止要求なら `stopping = true`、`remain = (symH - offset % symH) % symH`。次のコマ境界までの距離で、既に境界なら 0。
- `update()`: 回転中でなければ false。進む量は `spinSpeed`、停止要求中なら `min(spinSpeed, remain)` で `remain` を減らす。`offset = (offset + step) % total`。停止要求中に `remain == 0` になったら回転も停止要求も解除。動いたら true を返す（止まったフレームも最終位置を描くため true）。
- `center()`: 止まった状態では `offset` が `symH` の倍数になっている。中段の図柄は `strip[((symH - offset) mod total) / symH]`。

`spinSpeed` 12、`symH` 50 だと滑りは最大 5 フレーム（約 150ms）。実機のパチスロの「最大 190ms」に近く、狙った図柄がそのまま止まる感覚になる。

### メインループ

1. フレーム開始時刻を記録する。
2. 各ボタンについて `now := !pin.Get()`、`pressed[i] = now && !prev[i]` で立ち下がりを検出し、`prev` を更新する。組み込み関数名 `any` は変数名に使わない（`anyPressed` とする）。
3. 状態ごとの処理（「状態遷移」の表どおり）。回転中は全リールの `update()` を呼び、動いたものを描画し、このフレームで止まったリールのラベルを灰に更新する。
4. `frameTime - 経過時間` が正ならその分 `time.Sleep` する。

起動時の順序: ハード初期化 → 図柄の事前描画 → リールバッファ確保 → 各リールに配列を設定 → 静的部分（背景、タイトル、枠、三角マーカー） → クレジット → 3リールとラベル → 「PRESS ANY BUTTON」を黄色で表示。

### 文字の描画

画面への文字は `tinyfont.WriteLine(display, &freesans.Bold9pt7b, x, y, s, col)` で直接描く（`y` はベースライン）。書き換えるときは先に `FillRectangle` で背景色に戻す。数値の文字列化は `strconv.Itoa`。

## 注意点と検証手順

### 既知の注意点

- `pixel.Image` は値型だが、内部データはポインタで共有される。`RawBuffer()` はそのまま書き込めるスライスを返す。
- `RGB565BE` は既にバイトスワップ済みの値。自前でバイトを書くときに再度スワップしない。
- `ili9341` の描画関数は画面外の座標で error を返すだけで何も描かない。リールや枠が表示されないときは、まず座標が 320×240 に収まっているかを確認する。
- `display.SetPixel` は1ピクセルごとにウィンドウ設定のコマンドを送るので遅い。画面への直接の文字描画は、メッセージやラベルなど短い文字列に限る。
- `tinyfont` の日本語フォントは使わない。画面上の文字はすべて英字。
- スピーカーを使わなくても `SPEAKER_PIN` は必ず出力 Low にする。

### 検証手順

1. `tinygo build -target=m5stack -size short -o slot.bin .` がエラーなく通ることを確認する。
2. 可能なら、図柄描画と `reel.render` をホストの Go で実行し、`image/png` で書き出して見た目を確認する。`pixel` と `tinyfont` はホストでも動く。ハード依存部分（`machine`, `ili9341`）を分けておくとこの確認がしやすい。
3. 実機に書き込み、下の受け入れ基準を確認する。

### 受け入れ基準

- [ ] 起動直後、各リールの中段にブドウ・ベル・ブドウが並び、上段に 7 が見える（`offset` = 0）
- [ ] 画面が左右反転せず、「SLOT」が左上に正しく読める
- [ ] 背景が濃紺、コマ地がクリーム色に見える（反転していれば `displayInversion` を切り替える）
- [ ] どのボタンでもスタートし、クレジットが 3 減る
- [ ] A/B/C でそれぞれ左/中/右だけが止まり、止まったリールのラベルが灰になる
- [ ] 止まったリールは図柄がコマの枠にぴったり揃う
- [ ] 押しっぱなしで次のゲームが始まらない
- [ ] 中段の図柄と配当・メッセージが配当表どおり
- [ ] クレジット 3 未満で GAME OVER、ボタンで 50 に戻る
- [ ] 回転中にちらつきがなく、積極的に狙えば 7 を中段に止められる

## 拡張案

今回のスコープ外。初版が受け入れ基準を満たしたあとに検討する。

| 案 | 概要 |
| --- | --- |
| 効果音 | GPIO25（DAC）のスピーカーで、回転音・停止音・当たり音を鳴らす。メインループを止めない仕組みが必要 |
| 当たり演出 | 当たったら中段の有効ラインや図柄を点滅させる |
| 複数ライン | 上段・下段・斜め 2 本を加えた 5 ライン判定。現状は中段 1 ラインのみ判定している（`finishSpin` が各リールの `center()` だけを `judge` に渡す）。実装時に決めること: (1) 配当バランス。1ゲーム 3 クレジットのまま 5 ラインにすると当たりが大幅に増えるため、ベット数か配当の見直しが必要 (2) 「左リールのみチェリー」をどのラインで数えるか。ライン数が増えると重複しやすい (3) 当たったラインを画面でどう示すか |
| 難易度設定 | 起動時にボタンを押していると `spinSpeed` を切り替える |
| クレジット保存 | 電源を切ってもクレジットを残す |
