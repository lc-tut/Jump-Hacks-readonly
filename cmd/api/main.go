package main

import (
	"log"

	"github.com/digi-con/hackathon-template/internal/config"
	"github.com/digi-con/hackathon-template/internal/database"
	"github.com/digi-con/hackathon-template/internal/handlers"
	"github.com/digi-con/hackathon-template/internal/middleware"
	translate "github.com/digi-con/hackathon-template/internal/translate"
	ocr "github.com/digi-con/hackathon-template/internal/ocr"
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
				Key   string `json:"key"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				// Fallback to query parameters if no JSON provided
				req.Pages = c.Query("pages")
				req.Key = c.Query("key")
			}
			// If key not provided, derive from pages (convention: uploads/page-<pages>.jpg)
			key := req.Key
			if key == "" && req.Pages != "" {
				key = "uploads/page-" + req.Pages + ".jpg"
			}
			// Call orchestrator with router's cfg
			ocrTranslateReplace(cfg, key, req.Pages)
			c.JSON(200, gin.H{"message": "Test executed", "pages": req.Pages, "key": key})
		})
	}

	return router
}

func ocrTranslateReplace(cfg *config.Config, key string, pages string) {
	if pages == "" {
		log.Println("Test called without pages")
		return
	}
	log.Printf("Test called with pages: %s, key: %s", pages, key)

	// 1) Receive image using handlers (both bytes and base64 are available)
	if key == "" {
		log.Println("No key provided")
		return
	}
	image, err := handlers.GetImageBytes(cfg, key)
	if err != nil {
		log.Printf("failed to fetch image bytes: %v", err)
		return
	}
	b64, err := handlers.GetImageBase64(cfg, key)
	if err != nil {
		log.Printf("failed to fetch image base64: %v", err)
		return
	}
	log.Printf("Received image bytes: %d, base64 len: %d", len(image), len(b64))

	// 2) OCR (placeholder)
	boxes, err := ocr.OCRBytes(image)
	if err != nil{
		log.Printf("failed to ocr: %v",err)
	}
	log.Printf("OCR result: %s, boxes: %d", boxes, len(boxes))

	// 3) Translate (using internal/translate.TranslateText)
	translated, err := translate.TranslateText("test", "", "EN")
	if err != nil {
		log.Printf("translation failed: %v", err)
		// fallback to original OCR text
		translated = "Test"
	}
	log.Printf("Translated text: %s", translated)

	// 4) Replace text on image (placeholder)
	outImage := replaceTextPlaceholder(image, translated, []Box{})
	log.Printf("Replaced image size: %d", len(outImage))

	// Done: in real flow you'd return or save outImage
}

// Box is a placeholder bounding box type for OCR results
type Box struct {
	X int
	Y int
	W int
	H int
}

// replaceTextPlaceholder simulates drawing translated text onto the image and returns new bytes
func replaceTextPlaceholder(image []byte, translated string, boxes []Box) []byte {
	log.Printf("Simulating replace text: translated=%s boxes=%d", translated, len(boxes))
	// no-op: return original image bytes
	return image
}
