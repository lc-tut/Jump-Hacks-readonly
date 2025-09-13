package main

import (
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"log"
	"os"

	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

func main() {
	// 4032x3024の白い画像を作る。
	img := image.NewRGBA(image.Rect(0, 0, 4032, 3024))
	draw.Draw(img, img.Bounds(), &image.Uniform{color.White}, image.Point{}, draw.Src)

	// 描画する範囲を決めておく（灰色背景は削除）
	area := image.Rect(30, 20, 4032-30, 3024-20)

	// フォントを読み込んで、image/font.faceを作る。
	ttf, err := os.ReadFile("font.ttf")
	if err != nil {
		log.Fatal(err)
	}
	font_, err := truetype.Parse(ttf)
	if err != nil {
		log.Fatal(err)
	}
	face := truetype.NewFace(font_, &truetype.Options{
		Size: 256,
	})

	// 描画用の構造体を準備する。
	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(color.Black),
		Face: face,
	}

	// フォントフェイスから1行の高さを取得する。
	lineHeight := face.Metrics().Height.Ceil()

	// 描画する文字列。
	text := "Hello, World! こんにちは、世界！\nThis is a test."

	// 折り返しを考慮しながら1行ずつに分割する。
	runes := []rune(text)
	var lines []string
	start := 0
	for i := 0; i < len(runes); i++ {
		// 改行文字を見つけたら改行する。
		if runes[i] == '\n' {
			lines = append(lines, string(runes[start:i]))
			start = i + 1
			continue
		}

		// ここまでの文字列の横幅を計算する。
		width := d.MeasureString(string(runes[start:i]))

		// 横幅が描画範囲を越えていたら改行する。
		if width > fixed.I(area.Dx()) {
			i--
			lines = append(lines, string(runes[start:i]))
			start = i
		}
	}
	// 最後の1行をlinesに加えておく。
	if start < len(runes) {
		lines = append(lines, string(runes[start:]))
	}

	// 1行ずつ描画する。
	for lineOffset, line := range lines {
		y := area.Min.Y + (lineOffset+1)*lineHeight
		d.Dot = fixed.Point26_6{X: fixed.I(area.Min.X), Y: fixed.I(y)}
		d.DrawString(line)
	}

	// 画像をoutput.jpgとして保存する。
	out, err := os.Create("output.jpg")
	if err != nil {
		log.Fatal(err)
	}
	defer out.Close()
	if err := jpeg.Encode(out, img, nil); err != nil {
		log.Fatal(err)
	}
}
