package main

import (
	"bufio"
	"context"
	"fmt"
	"os"

	"github.com/google/generative-ai-go/genai"
	"github.com/joho/godotenv"
	"google.golang.org/api/option"
)

func main() {
	// .envファイルを読み込む
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file")
		return
	}

	ctx := context.Background()

	if err := run(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	// コンソールから質問を入力する
	fmt.Print("質問を入力してください: ")
	reader := bufio.NewReader(os.Stdin)
	question, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("質問の読み取りに失敗しました: %v", err)
	}

	// Gemini APIクライアントを作成する
	client, err := genai.NewClient(ctx, option.WithAPIKey(os.Getenv("GEMINI_APIKEY")))
	if err != nil {
		return fmt.Errorf("Geminiクライアントの作成に失敗しました: %v", err)
	}
	defer client.Close()

	// 質問を送信して回答を取得する
	model := client.GenerativeModel("gemini-2.5-pro") // 新しいモデル名に変更
	prompt := genai.Text(question)
	resp, err := model.GenerateContent(ctx, prompt)
	if err != nil {
		return fmt.Errorf("Gemini APIの呼び出しに失敗しました: %v", err)
	}

	// 回答を表示する
	printCandidates(resp.Candidates)

	return nil
}

func printCandidates(cs []*genai.Candidate) {
	for _, c := range cs {
		for _, p := range c.Content.Parts {
			fmt.Println(p)
		}
	}
}
