package main

import (
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"os"

	"golang.org/x/image/draw"
)

type SubImager interface {
	SubImage(r image.Rectangle) image.Image
}

func main() {
	f, err := os.Open("test.jpg")
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

	m := image.NewRGBA(image.Rect(0, 0, 200, 200)) // 200x200 の画像に test.jpg をのせる
	c := color.RGBA{0, 0, 255, 255}                // RGBA で色を指定(B が 255 なので青)

	draw.Draw(m, m.Bounds(), &image.Uniform{c}, image.ZP, draw.Src) // 青い画像を描画

	rct := image.Rectangle{image.Point{25, 25}, m.Bounds().Size()} // test.jpg をのせる位置を指定する(中央に配置する為に横:25 縦:25 の位置を指定)

	draw.Draw(m, rct, img, image.Point{0, 0}, draw.Src) // 合成する画像を描画

	jpeg.Encode(fso, m, &jpeg.Options{Quality: 100})
}
