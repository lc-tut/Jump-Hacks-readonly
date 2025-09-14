package main

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"log"
	"math"
	"os"

	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

// TextEntry は translated_result.json の各エントリを表す構造体
type TextEntry struct {
	ID     int        `json:"id"`
	Text   string     `json:"text"`
	Bounds [][]int    `json:"bounds"`
}

func main() {
	// translated_result.json を読み込む
	jsonData, err := os.ReadFile("translated_result.json")
	if err != nil {
		log.Fatalf("JSONファイルの読み込みに失敗しました: %v", err)
	}

	// JSONをパースする
	var entries []TextEntry
	if err := json.Unmarshal(jsonData, &entries); err != nil {
		log.Fatalf("JSONのパースに失敗しました: %v", err)
	}

	// フォントを読み込む
	ttf, err := os.ReadFile("font.ttf")
	if err != nil {
		log.Fatalf("フォントファイルの読み込みに失敗しました: %v", err)
	}
	
	font_, err := truetype.Parse(ttf)
	if err != nil {
		log.Fatalf("フォントのパースに失敗しました: %v", err)
	}

	// 出力ディレクトリを作成
	outputDir := "translated_images"
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatalf("出力ディレクトリの作成に失敗しました: %v", err)
	}

	// 各テキストエントリごとに画像を作成
	for _, entry := range entries {
		generateImageForEntry(entry, font_, outputDir)
	}
	
	log.Println("すべての画像が正常に生成されました: " + outputDir + "/")
}

// generateImageForEntry は各エントリに対して個別の画像を生成します
func generateImageForEntry(entry TextEntry, font_ *truetype.Font, outputDir string) {
	// エントリのboundsから描画領域を計算
	var minX, minY, maxX, maxY int
	minX, minY = math.MaxInt32, math.MaxInt32
	
	for _, point := range entry.Bounds {
		x, y := point[0], point[1]
		if x < minX {
			minX = x
		}
		if y < minY {
			minY = y
		}
		if x > maxX {
			maxX = x
		}
		if y > maxY {
			maxY = y
		}
	}
	
	// キャンバスサイズ計算（boundsの倍率）
	origWidth := maxX - minX
	origHeight := maxY - minY
	
	// 横方向と縦方向に別々の倍率を設定
	scaleFactorX := 1.6 // 横方向の倍率
	scaleFactorY := 1.2 // 縦方向の倍率
	width := int(float64(origWidth) * scaleFactorX)
	height := int(float64(origHeight) * scaleFactorY)
	
	// 画像を作成
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(img, img.Bounds(), &image.Uniform{color.White}, image.Point{}, draw.Src)
	
	// エントリの高さを計算
	entryHeight := maxY - minY
	
	// フォントサイズを調整（エントリの高さの1/5程度を目安に）
	fontSize := float64(entryHeight) / 5
	if fontSize < 12 {
		fontSize = 12 // 最小フォントサイズ
	}
	if fontSize > 48 {
		fontSize = 48 // 最大フォントサイズ（必要に応じて調整）
	}
	
	// フォントフェイスを作成
	face := truetype.NewFace(font_, &truetype.Options{
		Size: fontSize,
	})
	
	// 描画用の構造体を準備
	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(color.Black),
		Face: face,
	}
	
	// フォントメトリクスを取得
	metrics := face.Metrics()
	lineHeight := metrics.Height.Ceil()
	
	// テキストを折り返して描画するための準備
	text := entry.Text
	runes := []rune(text)
	var lines []string
	
	// テキストを折り返す
	start := 0
	// 拡大後のキャンバス幅を使用
	maxWidth := fixed.I(width)
	
	for i := 0; i < len(runes); i++ {
		// 改行文字があれば改行
		if runes[i] == '\n' {
			lines = append(lines, string(runes[start:i]))
			start = i + 1
			continue
		}
		
		// 現在の幅を計算
		width := d.MeasureString(string(runes[start : i+1]))
		
		// 幅が最大幅を超えたら折り返し
		if width > maxWidth && i > start {
			lines = append(lines, string(runes[start:i]))
			start = i
		}
	}
	
	// 最後の行を追加
	if start < len(runes) {
		lines = append(lines, string(runes[start:]))
	}
	
	// テキストを横書きで中央に配置して描画
	// キャンバスの中央位置を計算
	centerX := width / 2
	centerY := height / 2
	
	// 全体の行数から全体の高さを計算
	totalHeight := len(lines) * lineHeight
	
	// 描画開始位置を設定（中央から半分上に）
	startY := centerY - totalHeight/2
	
	for lineIndex, line := range lines {
		// 各行の幅を測定
		lineWidth := d.MeasureString(line)
		
		// 行の中央位置を計算（固定小数点から整数に変換）
		lineWidthPx := lineWidth.Round()
		
		// 行の開始X位置（中央揃え）
		startX := centerX - lineWidthPx/2
		
		// 描画位置を設定
		d.Dot = fixed.P(startX, startY + (lineIndex+1)*lineHeight)
		d.DrawString(line)
	}
	
	// 画像を保存
	filename := fmt.Sprintf("%s/id_%d.jpg", outputDir, entry.ID)
	out, err := os.Create(filename)
	if err != nil {
		log.Fatalf("出力ファイル %s の作成に失敗しました: %v", filename, err)
	}
	defer out.Close()
	
	if err := jpeg.Encode(out, img, &jpeg.Options{Quality: 90}); err != nil {
		log.Fatalf("JPEGエンコードに失敗しました: %v", err)
	}
	
	log.Printf("ID %d の画像を生成しました: %s", entry.ID, filename)
}