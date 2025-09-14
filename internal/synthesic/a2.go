package main

import (
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"log"
	"os"
	"strings"

	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

// RenderVertical は与えられた text を左から右に進む縦書きで描画し、outPath に 512x512 JPG を書き出します。
// fontPath が空ならリポジトリの font.ttf を使います。
func RenderVertical(text, outPath, fontPath string) error {
	if fontPath == "" {
		fontPath = "font.ttf"
	}
	// 512x512 の白背景画像
	w, h := 512, 512
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(img, img.Bounds(), &image.Uniform{color.White}, image.Point{}, draw.Src)

	// 描画領域（余白を少し取る）
	area := image.Rect(16, 16, w-16, h-16)

	// フォント読み込み
	ttf, err := os.ReadFile(fontPath)
	if err != nil {
		return err
	}
	f, err := truetype.Parse(ttf)
	if err != nil {
		return err
	}
	face := truetype.NewFace(f, &truetype.Options{Size: 48})

	// 描画用ドロワー（計測用にも使う）
	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(color.Black),
		Face: face,
	}

	// 1文字の高さ（行高さ）
	lineHeight := face.Metrics().Height.Ceil()
	// 横書き描画: テキストを行ごとに描画（描画領域を越えたら自動で改行）
	var wrappedLines []string
	maxW := area.Dx()
	for _, orig := range strings.Split(text, "\n") {
		runes := []rune(orig)
		start := 0
		for start < len(runes) {
			// extend end while width fits
			end := start + 1
			for end <= len(runes) {
				wpx := d.MeasureString(string(runes[start:end])).Ceil()
				if wpx > maxW {
					break
				}
				end++
			}
			if end > len(runes) {
				// all remaining runes fit
				wrappedLines = append(wrappedLines, string(runes[start:len(runes)]))
				break
			}
			// now end is the first index that does NOT fit (or end==start+1 when single rune too wide)
			if end == start+1 {
				// single rune doesn't fit; still output it to avoid infinite loop
				wrappedLines = append(wrappedLines, string(runes[start:end]))
				start = end
			} else {
				// take fitting runes up to end-1
				wrappedLines = append(wrappedLines, string(runes[start:end-1]))
				start = end - 1
			}
		}
	}

	for i, line := range wrappedLines {
		y := area.Min.Y + i*lineHeight + face.Metrics().Ascent.Ceil()
		d.Dot = fixed.Point26_6{X: fixed.I(area.Min.X), Y: fixed.I(y)}
		d.DrawString(line)
	}

	// 保存
	out, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer out.Close()
	if err := jpeg.Encode(out, img, &jpeg.Options{Quality: 90}); err != nil {
		return err
	}
	log.Println("wrote", outPath)
	return nil
}

func main() {
	RenderVertical("Hello, World! こんにちは、世界！\nThis is a test.", "out2.jpg", "")
}
