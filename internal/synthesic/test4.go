package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// CreateVerticalTextImage シンプルな縦書きテキスト画像を生成
func CreateVerticalTextImage(text string, outputPath string) error {
	// 固定サイズの画像を作成
	width, height := 400, 600
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// 白い背景で塗りつぶし
	white := color.RGBA{255, 255, 255, 255}
	draw.Draw(img, img.Bounds(), &image.Uniform{white}, image.Point{}, draw.Src)

	// フォントを読み込み
	fontBytes, err := os.ReadFile("font.ttf")
	if err != nil {
		return fmt.Errorf("フォントファイルが読み込めません: %v", err)
	}

	// フォントをパース
	ttf, err := opentype.Parse(fontBytes)
	if err != nil {
		return fmt.Errorf("フォントのパースに失敗しました: %v", err)
	}

	// フォントフェイスを作成
	face, err := opentype.NewFace(ttf, &opentype.FaceOptions{
		Size: 24,
		DPI:  72,
	})
	if err != nil {
		return fmt.Errorf("フォントフェイスの作成に失敗しました: %v", err)
	}
	defer face.Close()

	// テキストを縦書きで描画
	drawSimpleVerticalText(img, face, text, width, height)

	// PNG形式で保存
	return savePNG(img, outputPath)
}

// drawSimpleVerticalText シンプルな縦書き描画
func drawSimpleVerticalText(img *image.RGBA, face font.Face, text string, width, height int) {
	drawer := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(color.RGBA{0, 0, 0, 255}), // 黒
		Face: face,
	}

	// 行に分割
	lines := strings.Split(text, "\n")

	// 右端から開始
	currentX := width - 50

	for _, line := range lines {
		// 行の上端から開始
		currentY := 50

		// 文字を一つずつ縦に配置
		for _, r := range line {
			if r == ' ' {
				currentY += 20 // スペースは少し間隔を空ける
				continue
			}

			// 文字を描画
			drawer.Dot = fixed.Point26_6{
				X: fixed.I(currentX),
				Y: fixed.I(currentY),
			}
			drawer.DrawString(string(r))

			// 次の文字の位置へ移動（下へ）
			currentY += 30

			// 下端に達したら次の行へ
			if currentY > height-50 {
				break
			}
		}

		// 次の行へ移動（左へ）
		currentX -= 50

		// 左端に達したら終了
		if currentX < 50 {
			break
		}
	}
}

// savePNG PNG形式で画像を保存
func savePNG(img image.Image, outputPath string) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("ファイルの作成に失敗しました: %v", err)
	}
	defer file.Close()

	err = png.Encode(file, img)
	if err != nil {
		return fmt.Errorf("PNG保存に失敗しました: %v", err)
	}

	return nil
}

// TestSimpleVerticalText シンプルな縦書きテストの実行
func TestSimpleVerticalText() {
	text := `縦書き
テスト
文字列`

	err := CreateVerticalTextImage(text, "simple_vertical.png")
	if err != nil {
		fmt.Printf("エラー: %v\n", err)
		return
	}

	fmt.Println("縦書き画像が生成されました: simple_vertical.png")
}

func main() {
	TestSimpleVerticalText()
}
