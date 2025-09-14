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
	ocr(path)
}

func ocr(filename string) {
	ctx := context.Background()

	client, err := vision.NewImageAnnotatorClient(
		ctx,
		option.WithCredentialsFile("./internal/config/service-account.json"),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	file, err := os.Open(filename)
	if err != nil {
		log.Fatalf("Failed to read file: %v", err)
	}
	defer file.Close()

	image, err := vision.NewImageFromReader(file)
	if err != nil {
		log.Fatalf("Failed to create image: %v", err)
	}

	// ImageContext を省略して nil を渡す
	annotation, err := client.DetectDocumentText(ctx, image, nil)
	if err != nil {
		log.Fatalf("Failed to detect text: %v", err)
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
					text += " " // 単語間スペース
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

	jsonData, err := json.MarshalIndent(blocks, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal json: %v", err)
	}

	outputfile:="ocr_result.json"
	if err:=os.WriteFile(outputfile,jsonData,0644); err != nil{
		log.Fatalf("Failed to write json file: %v",err)
	}
	fmt.Printf("OCR結果を%sに保存しました\n",outputfile)
}
