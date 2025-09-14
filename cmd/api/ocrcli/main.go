package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	vision "cloud.google.com/go/vision/apiv1"
	"google.golang.org/api/option"
)

type OCRBlock struct {
	ID     int    `json:"id"`
	Text   string `json:"text"`
	Source string `json:"source,omitempty"`
}

func main() {
	// デフォルトの出力ディレクトリ（引数がなければここを処理）
	defaultDir := "/home/luy869/works/hackathon/Jump-Hacks-readonly/internal/asobi/cut_output"

	var targets []string
	if len(os.Args) < 2 {
		// 引数がない場合は defaultDir を処理
		targets = append(targets, defaultDir)
	} else {
		targets = append(targets, os.Args[1])
	}

	ctx := context.Background()
	client, err := vision.NewImageAnnotatorClient(ctx, option.WithCredentialsFile("./internal/config/service-account.json"))
	if err != nil {
		log.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	var allBlocks []OCRBlock
	idCounter := 1

	for _, t := range targets {
		info, err := os.Stat(t)
		if err != nil {
			log.Printf("skip %s: %v", t, err)
			continue
		}

		if info.IsDir() {
			// ディレクトリを再帰走査。masks ディレクトリはスキップ
			_ = filepath.WalkDir(t, func(p string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if d.IsDir() && d.Name() == "masks" {
					return fs.SkipDir
				}
				if d.IsDir() {
					return nil
				}
				ext := strings.ToLower(filepath.Ext(p))
				if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".bmp" {
					return nil
				}

				blocks, err := OCRFile(ctx, client, p)
				if err != nil {
					log.Printf("OCR failed for %s: %v", p, err)
					return nil
				}

				// 同じ source(file) のものは同じ ID を付与する
				baseID := idCounter
				for i := range blocks {
					blocks[i].ID = baseID
					blocks[i].Source = p
					allBlocks = append(allBlocks, blocks[i])
				}
				if len(blocks) > 0 {
					idCounter++
				}
				return nil
			})
		} else {
			// 単一ファイル
			ext := strings.ToLower(filepath.Ext(t))
			if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".bmp" {
				log.Fatalf("unsupported file type: %s", t)
			}
			blocks, err := OCRFile(ctx, client, t)
			if err != nil {
				log.Fatalf("OCR failed: %v", err)
			}

			// 同じ source(file) のものは同じ ID を付与する
			baseID := idCounter
			for i := range blocks {
				blocks[i].ID = baseID
				blocks[i].Source = t
				allBlocks = append(allBlocks, blocks[i])
			}
			if len(blocks) > 0 {
				idCounter++
			}
		}
	}

	if len(allBlocks) == 0 {
		fmt.Println("処理対象の画像が見つかりませんでした")
		return
	}

	if err := SaveJSON("ocr_result.json", allBlocks); err != nil {
		log.Fatalf("Failed to save JSON: %v", err)
	}

	fmt.Println("OCR結果をocr_result.jsonに保存しました")
}

// OCRFile は既に作成した vision クライアントを使って画像ファイルを処理する
func OCRFile(ctx context.Context, client *vision.ImageAnnotatorClient, filename string) ([]OCRBlock, error) {
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

	// annotation が nil または Pages が空の場合に備える
	if annotation == nil || len(annotation.Pages) == 0 {
		return []OCRBlock{}, nil
	}

	var blocks []OCRBlock
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

			// 座標は不要なので Text のみを記録
			blocks = append(blocks, OCRBlock{
				Text: text,
			})
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
