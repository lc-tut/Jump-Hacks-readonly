package ocr

import (
	"bytes"
	"context"
	"fmt"
	"log"

	vision "cloud.google.com/go/vision/apiv1"
	"google.golang.org/api/option"
)

type OCRBlock struct {
	ID     int        `json:"id"`
	Text   string     `json:"text"`
	Bounds [][2]int32 `json:"bounds"`
}

func OCRBytes(imageBytes []byte) ([]OCRBlock, error) {
	ctx := context.Background()
	client, err := vision.NewImageAnnotatorClient(
		ctx,
		option.WithCredentialsFile("./internal/config/service-account.json"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}
	defer client.Close()

	log.Printf("Input image: %d",len(imageBytes))

	image, err := vision.NewImageFromReader(bytes.NewReader(imageBytes))
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