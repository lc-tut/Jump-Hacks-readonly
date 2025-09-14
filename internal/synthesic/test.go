package main

import (
	"fmt"
	"image"

	// "image/color"
	"image/jpeg"
	"os"

	"golang.org/x/image/draw"
)

type SubImager interface {
	SubImage(r image.Rectangle) image.Image
}

func renderImage() {
	f, err := os.Open("internal/synthesic/IMG_0288.jpg")
	if err != nil {
		fmt.Println("open:", err)
		return
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		fmt.Println("decode:", err)
		return
	}

	fso, err := os.Create("out.jpg")
	if err != nil {
		fmt.Println("create:", err)
		return
	}
	defer fso.Close()

	// test.jpgの大きさに合わせてキャンバスを作成
	bounds := img.Bounds()
	m := image.NewRGBA(bounds)
	// c := color.RGBA{255, 255, 255, 255} // RGBA で白色を指定

	// まずtest.jpgを描画（背景として全体に描画）
	draw.Draw(m, bounds, img, bounds.Min, draw.Src) // test.jpgを背景として描画

	// 上に貼り付ける画像を読み込み
	f2, err := os.Open("output.jpg") // 貼り付ける画像のパス（適宜変更してください）
	if err != nil {
		fmt.Println("overlay image open:", err)
		return
	}
	defer f2.Close()

	overlayImg, _, err := image.Decode(f2)
	if err != nil {
		fmt.Println("overlay image decode:", err)
		return
	}

	// その上に画像をそのままの大きさで描画（右下の位置から貼り付け）
	imgWidth := bounds.Dx()
	imgHeight := bounds.Dy()
	overlayBounds := overlayImg.Bounds()

	// 貼り付け位置を指定（右下の位置から画像をそのまま貼り付け）
	startX := imgWidth * 3 / 4
	startY := imgHeight * 3 / 4
	overlayRect := image.Rectangle{
		image.Point{startX, startY},
		image.Point{startX + overlayBounds.Dx(), startY + overlayBounds.Dy()},
	} // 画像をそのままの大きさで配置
	draw.Draw(m, overlayRect, overlayImg, overlayBounds.Min, draw.Src) // 既存画像をそのままの大きさで描画

	// 白い矩形を描画する場合（コメントアウト）
	// c := color.RGBA{255, 255, 255, 255} // RGBA で白色を指定
	// draw.Draw(m, overlayRect, &image.Uniform{c}, image.ZP, draw.Src) // 白い矩形を部分的に描画

	jpeg.Encode(fso, m, &jpeg.Options{Quality: 100})
}

func main() {
	renderImage()
}
