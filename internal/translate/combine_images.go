package main

import (
	"encoding/json"
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"log"
	"os"
	"path/filepath"
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

	// 元の画像（0002.jpg）を読み込む
	baseImagePath := "cmd/api/ocrcli/0002.jpg"
	baseImageFile, err := os.Open(baseImagePath)
	if err != nil {
		log.Fatalf("ベース画像のオープンに失敗しました: %v", err)
	}
	defer baseImageFile.Close()

	baseImage, _, err := image.Decode(baseImageFile)
	if err != nil {
		log.Fatalf("ベース画像のデコードに失敗しました: %v", err)
	}

	// 編集可能な画像を作成
	bounds := baseImage.Bounds()
	canvas := image.NewRGBA(bounds)
	draw.Draw(canvas, bounds, baseImage, bounds.Min, draw.Src)

	// 各テキスト画像をboundsの位置に貼り付ける
	for _, entry := range entries {
		// 各エントリの画像ファイルを読み込む
		imgPath := filepath.Join("translated_images", fmt.Sprintf("id_%d.jpg", entry.ID))
		imgFile, err := os.Open(imgPath)
		if err != nil {
			log.Printf("警告: ID %d の画像ファイルを開けませんでした: %v", entry.ID, err)
			continue
		}
		defer imgFile.Close()

		img, _, err := image.Decode(imgFile)
		if err != nil {
			log.Printf("警告: ID %d の画像ファイルのデコードに失敗しました: %v", entry.ID, err)
			continue
		}

		// boundsから貼り付け位置を計算
		if len(entry.Bounds) < 4 {
			log.Printf("警告: ID %d のboundsが不正です", entry.ID)
			continue
		}

		// boundsの左上と右下の座標を取得
		minX, minY := entry.Bounds[0][0], entry.Bounds[0][1]
		maxX, maxY := minX, minY
		
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

		// 元のboundsの中央点を計算
		centerX := (minX + maxX) / 2
		centerY := (minY + maxY) / 2
		
		// テキスト画像のサイズを取得
		imgWidth := img.Bounds().Dx()
		imgHeight := img.Bounds().Dy()
		
		// テキスト画像の左上の座標を計算（中央に配置）
		startX := centerX - imgWidth/2
		startY := centerY - imgHeight/2
		
		// 貼り付け先の矩形を定義（テキスト画像のサイズに合わせる）
		targetRect := image.Rect(startX, startY, startX + imgWidth, startY + imgHeight)
		
		// 画像を貼り付け
		draw.Draw(canvas, targetRect, img, img.Bounds().Min, draw.Over)
		
		log.Printf("ID %d の画像を座標 (%d,%d)-(%d,%d) に貼り付けました", entry.ID, startX, startY, startX + imgWidth, startY + imgHeight)
	}

	// 結果を保存
	outputPath := "combined_output.jpg"
	outputFile, err := os.Create(outputPath)
	if err != nil {
		log.Fatalf("出力ファイルの作成に失敗しました: %v", err)
	}
	defer outputFile.Close()

	if err := jpeg.Encode(outputFile, canvas, &jpeg.Options{Quality: 90}); err != nil {
		log.Fatalf("JPEGエンコードに失敗しました: %v", err)
	}

	log.Printf("全ての画像が正常に貼り付けられました: %s", outputPath)
}