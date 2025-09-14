package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	vision "cloud.google.com/go/vision/apiv1"
	"google.golang.org/api/option"
)

type OCRBlock struct {
	ID     int        `json:"id"`
	Text   string     `json:"text"`
	Bounds [][2]int32 `json:"bounds"` // 各頂点の座標
}

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("%s /path/to/image.png", os.Args[0])
	}
	path := os.Args[1]

	blocks, err := OCR(path)
	if err != nil {
		log.Fatalf("OCR failed: %v", err)
	}

	err = SaveJSON("ocr_result.json", blocks)
	if err != nil {
		log.Fatalf("Failed to save JSON: %v", err)
	}

	fmt.Println("OCR結果をocr_result.jsonに保存しました")
}

// OCR は画像ファイルを受け取り OCRBlock のスライスを返す
func OCR(filename string) ([]OCRBlock, error) {
	ctx := context.Background()

	client, err := vision.NewImageAnnotatorClient(
		ctx,
		option.WithCredentialsFile("./internal/config/service-account.json"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}
	defer client.Close()

	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	image, err := vision.NewImageFromReader(file)
	if err != nil {
		return nil, fmt.Errorf("failed to create image: %w", err)
	}

	annotation, err := client.DetectDocumentText(ctx, image, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to detect text: %w", err)
	}

	var blocks []OCRBlock
	id := 1
	for _, page := range annotation.Pages {
		for _, block := range page.Blocks {
			var text string
			for _, paragraph := range block.Paragraphs {
				for _, word := range paragraph.Words {
					for _, symbol := range word.Symbols {
						text += symbol.Text
					}
					text += " "
				}
			}

			var bounds [][2]int32
			for _, v := range block.BoundingBox.Vertices {
				bounds = append(bounds, [2]int32{v.X, v.Y})
			}

			blocks = append(blocks, OCRBlock{
				ID:     id,
				Text:   text,
				Bounds: bounds,
			})
			id++
		}
	}

	return blocks, nil
}

// SaveJSON は OCRBlock のスライスを JSON ファイルに保存する
func SaveJSON(filename string, blocks []OCRBlock) error {
	data, err := json.MarshalIndent(blocks, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
