package main

import (
	"log"
	// "os"
	// "os/exec"
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/digi-con/hackathon-template/internal/config"
	"github.com/digi-con/hackathon-template/internal/database"
	"github.com/digi-con/hackathon-template/internal/handlers"
	"github.com/digi-con/hackathon-template/internal/middleware"
	// translate "github.com/digi-con/hackathon-template/internal/translate"
	// ocr "github.com/digi-con/hackathon-template/internal/ocr"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

var (
	Version   = "development"
	BuildTime = "unknown"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Load configuration
	cfg := config.Load()

	// Set Gin mode
	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Initialize database
	db, err := database.Initialize(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Initialize router
	router := setupRouter(cfg, db)

	// Start server
	port := cfg.Port
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting %s server on port %s (version: %s, built: %s)", cfg.AppName, port, Version, BuildTime)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func setupRouter(cfg *config.Config, db database.DB) *gin.Engine {
	router := gin.New()

	// Essential middleware (always needed)
	router.Use(gin.Logger())         // Simple logging (easier to debug)
	router.Use(gin.Recovery())       // Panic recovery
	router.Use(middleware.CORS(cfg)) // CORS (needed for frontend)

	// Set development mode for better debugging
	if cfg.Env != "production" {
		gin.SetMode(gin.DebugMode)
	}

	// Health check endpoint
	router.GET("/health", handlers.HealthCheck(db))

	// API routes
	v1 := router.Group("/api/v1")
	{
		// Auth routes (no auth required)
		auth := v1.Group("/auth")
		{
			auth.POST("/register", handlers.Register(db, cfg))
			auth.POST("/login", handlers.Login(db, cfg))
			auth.POST("/refresh", handlers.RefreshToken(db, cfg))
		}

		// Protected routes (auth required)
		protected := v1.Group("/")
		protected.Use(middleware.AuthRequired(cfg))
		{
			// User routes
			protected.GET("/profile", handlers.GetProfile(db))
			protected.PUT("/profile", handlers.UpdateProfile(db))

			// File upload routes
			protected.POST("/upload", handlers.UploadFile(db, cfg))
			protected.GET("/files", handlers.ListFiles(db))
			protected.DELETE("/files/:id", handlers.DeleteFile(db, cfg))

			// Storage presign
			protected.POST("/storage/presign", handlers.PresignUpload(db, cfg))
			protected.POST("/storage/presign-get", handlers.PresignGet(db, cfg))
		}

		// Admin routes (admin role required)
		admin := v1.Group("/admin")
		admin.Use(middleware.AuthRequired(cfg))
		admin.Use(middleware.AdminRequired())
		{
			admin.GET("/users", handlers.ListUsers(db))
			admin.GET("/stats", handlers.GetStats(db))
		}

		// 公開画像取得エンドポイント（認証不要）
		// S3 のオブジェクトキーをパスとして受け取り、画像を返す
		// 例: GET /api/v1/image/uploads/local-image.png
		v1.GET("/image/*key", handlers.GetImage(cfg))

		// OCR テスト用エンドポイント（認証不要）
		// GET /api/v1/get_ocr_img -> 呼び出し時に ocrTranslateReplace(cfg, key, pages) を実行
		v1.GET("/get_ocr_img", func(c *gin.Context) {
			// Try to read JSON body: {"pages": "...", "key": "..."}
			var req struct {
				Pages string `json:"pages"`
				Title   string `json:"title"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				// Fallback to query parameters if no JSON provided
				req.Pages = c.Query("pages")
				req.Title = c.Query("title")
			}
			// If key not provided, derive from pages (convention: uploads/page-<pages>.jpg)
			
			key := "uploads/"+ req.Title +"/page-" + req.Pages + ".jpg"
			
			// Call orchestrator with router's cfg
			image_out := ocrTranslateReplace(cfg, key, req.Pages)
			c.JSON(200, gin.H{"image": image_out})
		})
	}

	return router
}

func ocrTranslateReplace(cfg *config.Config, key string, pages string) []byte {
	if pages == "" {
		log.Println("Test called without pages")
		return nil
	}
	log.Printf("Test called with pages: %s, key: %s", pages, key)

	// 1) Receive image using handlers (both bytes and base64 are available)
	if key == "" {
		log.Println("No key provided")
		return nil
	}
	image, err := handlers.GetImageBytes(cfg, key)
	if err != nil {
		log.Printf("failed to fetch image bytes: %v", err)
		return nil
	}
	b64, err := handlers.GetImageBase64(cfg, key)
	if err != nil {
		log.Printf("failed to fetch image base64: %v", err)
		return nil
	}
	log.Printf("Received image bytes: %d, base64 len: %d", len(image), len(b64))

	asobiURL := "http://asobi:5001/run_pipeline"
	payload := map[string]string{
		"pages": pages,
		"key":   key,
	}
	jsonPayload, _ := json.Marshal(payload)
	resp, err := http.Post(asobiURL, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		log.Printf("failed to call asobi pipeline: %v", err)
		return nil
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	// ここでAPIレスポンス(JSON)をパースしてstdout, stderrを個別に出す
	var result struct {
		Stdout     string `json:"stdout"`
		Stderr     string `json:"stderr"`
		Returncode int    `json:"returncode"`
		Error      string `json:"error"`
	}
	_ = json.Unmarshal(body, &result)
	if result.Stdout != "" {
		log.Printf("[asobi pipeline stdout]\n%s", result.Stdout)
	}
	if result.Stderr != "" {
		log.Printf("[asobi pipeline stderr]\n%s", result.Stderr)
	}
	if result.Error != "" {
		log.Printf("[asobi pipeline error] %s", result.Error)
	}
	log.Printf("asobi pipeline return code: %d", result.Returncode)
	// 必要ならbodyの内容で何か処理

	return image
}
