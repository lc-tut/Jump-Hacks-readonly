package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// OCRResult OCR結果を表す構造体
type OCRResult struct {
	ID     int    `json:"id"`
	Text   string `json:"text"`
	Source string `json:"source"`
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

	// バッチサイズ（DeepL のレートやサイズに合わせて調整）
	batchSize := 20

	// 各テキストをバッチで翻訳して置き換
	total := len(ocrResults)
	for start := 0; start < total; start += batchSize {
		end := start + batchSize
		if end > total {
			end = total
		}

		// バッチ抽出
		texts := make([]string, 0, end-start)
		for i := start; i < end; i++ {
			texts = append(texts, ocrResults[i].Text)
		}

		// 翻訳呼び出し（バッチ）
		translatedBatch, err := DeepLTranslateBatch(texts, sourceLang, targetLang)
		if err != nil {
			// フォールバック: 各要素ごとに個別翻訳を試す（失敗時は元のテキストを使用）
			log.Printf("warning: batch translate failed for items %d-%d: %v; falling back to per-item", start+1, end, err)
			// per-item fallback
			for i := start; i < end; i++ {
				translatedText, err := DeepLTranslate(ocrResults[i].Text, sourceLang, targetLang)
				if err != nil {
					log.Printf("warning: translate failed for id=%d text=%q: %v; using original text", ocrResults[i].ID, ocrResults[i].Text, err)
					translatedText = ocrResults[i].Text
				}
				ocrResults[i].Text = translatedText
			}
		} else {
			// バッチ結果を反映
			for j, tr := range translatedBatch {
				ocrResults[start+j].Text = tr
			}
		}

		// 少し待機してレートを抑える
		time.Sleep(200 * time.Millisecond)
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
	// source を固定: ocr_result.json
	inputFile := "/home/luy869/works/hackathon/Jump-Hacks-readonly/ocr_result.json"

	// 出力ファイルと targetLang は引数で指定可能
	outputFile := "/home/luy869/works/hackathon/Jump-Hacks-readonly/translated_result.json"
	targetLang := "en" // デフォルト

	if len(os.Args) >= 2 {
		outputFile = os.Args[1]
	}
	if len(os.Args) >= 3 {
		targetLang = os.Args[2]
	}

	sourceLang := "ja" // 固定

	// 入力ファイル存在確認
	if _, err := os.Stat(inputFile); err != nil {
		fmt.Printf("input file not found: %s\n", inputFile)
		os.Exit(1)
	}

	err := TranslateAndReplaceJSONFile(inputFile, outputFile, sourceLang, targetLang)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

func DeepLTranslate(text, sourceLang, targetLang string) (string, error) {
	// 翻訳APIのURLとパラメータ
	apiURL := "https://api-free.deepl.com/v2/translate"
	params := url.Values{}

	// APIキーが設定されているか確認
	key := loadEnvVar("DEEPL_API_KEY")
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

	// 翻訳APIにリクエストを送信（まずクエリパラメータ認証）
	res, err := http.Get(apiURL + "?" + params.Encode())
	if err != nil {
		return "", fmt.Errorf("failed to send translation request: %v", err)
	}
	defer res.Body.Close()

	// ステータスコードを確認
	if res.StatusCode == http.StatusForbidden {
		// 403 の場合は、ヘッダーによる認証方式で再試行する
		req, _ := http.NewRequest("GET", apiURL, nil)
		req.Header.Set("Authorization", "DeepL-Auth-Key "+key)
		q := req.URL.Query()
		q.Set("text", text)
		if sourceLang != "" {
			q.Set("source_lang", strings.ToUpper(sourceLang))
		}
		if targetLang != "" {
			q.Set("target_lang", strings.ToUpper(targetLang))
		}
		req.URL.RawQuery = q.Encode()

		resp2, err2 := http.DefaultClient.Do(req)
		if err2 != nil {
			return "", fmt.Errorf("retry with header auth failed: %v", err2)
		}
		defer resp2.Body.Close()
		if resp2.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp2.Body)
			return "", fmt.Errorf("translation API returned status %d on header-auth retry: %s", resp2.StatusCode, strings.TrimSpace(string(body)))
		}
		res = resp2
	} else if res.StatusCode != http.StatusOK {
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

// DeepLTranslateBatch は複数テキストを一度に送信して翻訳結果をまとめて返す
func DeepLTranslateBatch(texts []string, sourceLang, targetLang string) ([]string, error) {
	apiURL := "https://api-free.deepl.com/v2/translate"
	key := loadEnvVar("DEEPL_API_KEY")
	if key == "" {
		return nil, fmt.Errorf("DEEPL_API_KEY is not set")
	}

	params := url.Values{}
	params.Set("auth_key", key)
	if sourceLang != "" {
		params.Set("source_lang", strings.ToUpper(sourceLang))
	}
	if targetLang != "" {
		params.Set("target_lang", strings.ToUpper(targetLang))
	}
	for _, t := range texts {
		params.Add("text", t)
	}

	// POST フォームで送信（GETでも可だが長いテキスト対策）
	res, err := http.PostForm(apiURL, params)
	if err != nil {
		return nil, fmt.Errorf("failed to send translation request: %v", err)
	}
	defer res.Body.Close()

	// ステータスチェック
	if res.StatusCode == http.StatusForbidden {
		// ヘッダー認証で再試行
		req, _ := http.NewRequest("POST", apiURL, strings.NewReader(params.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Authorization", "DeepL-Auth-Key "+key)
		resp2, err2 := http.DefaultClient.Do(req)
		if err2 != nil {
			return nil, fmt.Errorf("retry with header auth failed: %v", err2)
		}
		defer resp2.Body.Close()
		if resp2.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp2.Body)
			return nil, fmt.Errorf("translation API returned status %d on header-auth retry: %s", resp2.StatusCode, strings.TrimSpace(string(body)))
		}
		res = resp2
	} else if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("translation API returned status %d: %s", res.StatusCode, strings.TrimSpace(string(body)))
	}

	var result struct {
		Translations []struct {
			Text string `json:"text"`
		} `json:"translations"`
	}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse translation response: %v", err)
	}

	if len(result.Translations) != len(texts) {
		// DeepL は入力順に翻訳を返すはずだが、一致しない場合はエラーとする
		return nil, fmt.Errorf("mismatch translations count: got %d want %d", len(result.Translations), len(texts))
	}

	out := make([]string, len(result.Translations))
	for i, tr := range result.Translations {
		out[i] = tr.Text
	}
	return out, nil
}

// loadEnvVar は指定したキーの環境変数値を返します。未設定なら空文字を返します。
func loadEnvVar(key string) string {
	return strings.TrimSpace(os.Getenv(key))
}
