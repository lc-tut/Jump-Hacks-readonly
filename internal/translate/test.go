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

func test() {
	// .env を読み込む（存在しなければ無視）
	_ = godotenv.Load()

	test_word := "こんばんは"

	// test_Wordを翻訳
	translated_Word, err := TranslateText(test_word, "ja", "en")
	if err != nil {
		fmt.Printf("Failed to translate word: %v", err)
		return
	}

	// 翻訳結果を出力
	fmt.Printf("%q\n", translated_Word)
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

func main() {
	// .env を読み込む（存在しなければ無視）
	_ = godotenv.Load()

	test()
	// test_Wordを翻訳
}