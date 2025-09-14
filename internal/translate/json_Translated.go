package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// OCRResult OCR結果を表す構造体
type OCRResult struct {
	ID     int     `json:"id"`
	Text   string  `json:"text"`
	Bounds [][]int `json:"bounds"`
}

// TranslateAndReplaceJSONFile JSONファイルのtextフィールドを翻訳して置き換える
func TranslateAndReplaceJSONFile(inputFile, outputFile, sourceLang, targetLang string) error {
	// .env を読み込む（存在しなければ無視）
	_ = godotenv.Load()

	// 入力JSONファイルを読み込み
	ocrResults, err := loadOCRResults(inputFile)
	if err != nil {
		return fmt.Errorf("failed to load OCR results: %v", err)
	}

	// 各テキストを翻訳して置き換え
	for i, result := range ocrResults {
		translatedText, err := TranslateText(result.Text, sourceLang, targetLang)
		if err != nil {
			return fmt.Errorf("failed to translate text '%s': %v", result.Text, err)
		}

		// 翻訳されたテキストで置き換え
		ocrResults[i].Text = translatedText
		
		fmt.Printf("Translated %d/%d: %s\n", i+1, len(ocrResults), translatedText)
	}

	// 翻訳済みの結果をJSONファイルに出力
	if err := saveOCRResults(ocrResults, outputFile); err != nil {
		return fmt.Errorf("failed to save translated results: %v", err)
	}

	fmt.Printf("Translation completed. Output saved to: %s\n", outputFile)
	return nil
}

// loadOCRResults OCR結果JSONファイルを読み込む
func loadOCRResults(filename string) ([]OCRResult, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	var results []OCRResult
	if err := json.NewDecoder(file).Decode(&results); err != nil {
		return nil, fmt.Errorf("failed to decode JSON: %v", err)
	}

	return results, nil
}

// saveOCRResults OCR結果をJSONファイルに保存
func saveOCRResults(results []OCRResult, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %v", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(results); err != nil {
		return fmt.Errorf("failed to encode JSON: %v", err)
	}

	return nil
}

func main() {
	// 使用例
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run json_Translated.go <input_file> <output_file> [source_lang] [target_lang]")
		fmt.Println("Example: go run json_Translated.go ocr_result.json translated_result.json ja en")
		return
	}

	inputFile := os.Args[1]
	outputFile := os.Args[2]
	sourceLang := "ja" // デフォルト
	targetLang := "en" // デフォルト

	if len(os.Args) >= 4 {
		sourceLang = os.Args[3]
	}
	if len(os.Args) >= 5 {
		targetLang = os.Args[4]
	}

	err := TranslateAndReplaceJSONFile(inputFile, outputFile, sourceLang, targetLang)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

func TranslateText(text, sourceLang, targetLang string) (string, error) {
	// 翻訳APIのURLとパラメータ
	apiURL := "https://api-free.deepl.com/v2/translate"
	params := url.Values{}

	// APIキーが設定されているか確認
	key := loadEnv("DEEPL_API_KEY")
	if key == "" {
		return "", fmt.Errorf("DEEPL_API_KEY is not set")
	}
	params.Set("auth_key", key)

	// ソース言語とターゲット言語を設定（空文字の場合は自動検出）
	if sourceLang != "" {
		params.Set("source_lang", strings.ToUpper(sourceLang))
	}
	if targetLang != "" {
		params.Set("target_lang", strings.ToUpper(targetLang))
	}
	params.Set("text", text)

	// 翻訳APIにリクエストを送信
	res, err := http.Get(apiURL + "?" + params.Encode())
	if err != nil {
		return "", fmt.Errorf("failed to send translation request: %v", err)
	}
	defer res.Body.Close()

	// ステータスコードを確認してエラー時はレスポンスを返す
	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		return "", fmt.Errorf("translation API returned status %d: %s", res.StatusCode, strings.TrimSpace(string(body)))
	}

	// レスポンスのJSONをパースして翻訳結果を取得
	var result struct {
		Translations []struct {
			Text string `json:"text"`
		} `json:"translations"`
	}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to parse translation response: %v", err)
	}
	if len(result.Translations) == 0 {
		return "", fmt.Errorf("no translations found")
	}
	return result.Translations[0].Text, nil
}

// loadEnv は指定したキーの環境変数値を返します。未設定なら空文字を返します。
func loadEnv(key string) string {
	return strings.TrimSpace(os.Getenv(key))
}
