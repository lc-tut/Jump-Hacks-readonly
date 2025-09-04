package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gin-gonic/gin"
)

type serverConfig struct {
	Port             string
	AWSRegion        string
	S3Bucket         string
	EndpointURL      string
	ForcePathStyle   bool
	UploadExpires    time.Duration
	GetExpires       time.Duration
	AllowedFileTypes []string
	TeamTokenPairs   map[string]string
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvBool(key string, def bool) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	if v == "" {
		return def
	}
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

func getEnvDurationSeconds(key string, def int) time.Duration {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return time.Duration(def) * time.Second
	}
	var n int
	_, err := fmt.Sscanf(v, "%d", &n)
	if err != nil || n <= 0 {
		return time.Duration(def) * time.Second
	}
	return time.Duration(n) * time.Second
}

func parseTeamTokens(env string) map[string]string {
	// Format: TEAM_TOKENS="team-01:tokenA,team-02:tokenB"
	res := make(map[string]string)
	if strings.TrimSpace(env) == "" {
		return res
	}
	pairs := strings.Split(env, ",")
	for _, p := range pairs {
		p = strings.TrimSpace(p)
		parts := strings.SplitN(p, ":", 2)
		if len(parts) != 2 {
			continue
		}
		teamID := strings.TrimSpace(parts[0])
		token := strings.TrimSpace(parts[1])
		if teamID != "" && token != "" {
			res[teamID] = token
		}
	}
	return res
}

func loadConfig() serverConfig {
	return serverConfig{
		Port:             getEnv("PORT", "8080"),
		AWSRegion:        getEnv("AWS_REGION", "us-east-1"),
		S3Bucket:         getEnv("S3_BUCKET_NAME", "digicon-hackathon-2025-uploads"),
		EndpointURL:      getEnv("AWS_ENDPOINT_URL", ""),
		ForcePathStyle:   getEnvBool("AWS_S3_FORCE_PATH_STYLE", true),
		UploadExpires:    getEnvDurationSeconds("PRESIGN_UPLOAD_EXPIRES_SECONDS", 600),
		GetExpires:       getEnvDurationSeconds("PRESIGN_GET_EXPIRES_SECONDS", 600),
		AllowedFileTypes: parseCSV(getEnv("ALLOWED_FILE_TYPES", "jpg,jpeg,png,gif,pdf")),
		TeamTokenPairs:   parseTeamTokens(getEnv("TEAM_TOKENS", "")),
	}
}

func parseCSV(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	res := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.ToLower(strings.TrimSpace(p))
		if p != "" {
			res = append(res, p)
		}
	}
	return res
}

func isAllowedFileType(filename string, allowed []string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	if ext != "" {
		ext = ext[1:]
	}
	for _, a := range allowed {
		if a == ext {
			return true
		}
	}
	return false
}

func buildS3Clients(cfg serverConfig) (*s3.Client, *s3.PresignClient, error) {
	awsCfg, err := awsconfig.LoadDefaultConfig(context.TODO(), awsconfig.WithRegion(cfg.AWSRegion))
	if err != nil {
		return nil, nil, err
	}
	s3c := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if cfg.EndpointURL != "" {
			o.BaseEndpoint = aws.String(cfg.EndpointURL)
			o.UsePathStyle = cfg.ForcePathStyle
		}
	})
	pre := s3.NewPresignClient(s3c)
	return s3c, pre, nil
}

func main() {
	cfg := loadConfig()
	if len(cfg.TeamTokenPairs) == 0 {
		log.Println("Warning: TEAM_TOKENS is empty. All requests will be unauthorized.")
	}

	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	// Health
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "time": time.Now()})
	})

	// Middleware: simple team auth via headers X-Team-Id and X-Team-Token
	authn := func(c *gin.Context) (string, bool) {
		teamID := strings.TrimSpace(c.GetHeader("X-Team-Id"))
		token := strings.TrimSpace(c.GetHeader("X-Team-Token"))
		if teamID == "" || token == "" {
			return "", false
		}
		expected, ok := cfg.TeamTokenPairs[teamID]
		if !ok || expected != token {
			return "", false
		}
		return teamID, true
	}

	// Upload presign (PUT)
	r.POST("/presign-upload", func(c *gin.Context) {
		teamID, ok := authn(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		var req struct {
			FileName    string `json:"filename" binding:"required"`
			ContentType string `json:"content_type"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request", "details": err.Error()})
			return
		}
		if !isAllowedFileType(req.FileName, cfg.AllowedFileTypes) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "file type not allowed", "allowed_types": cfg.AllowedFileTypes})
			return
		}

		_, pre, err := buildS3Clients(cfg)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "aws config error", "details": err.Error()})
			return
		}

		timestamp := time.Now().Unix()
		key := fmt.Sprintf("uploads/%s/%d_%s", teamID, timestamp, filepath.Base(req.FileName))
		in := &s3.PutObjectInput{
			Bucket:      aws.String(cfg.S3Bucket),
			Key:         aws.String(key),
			ContentType: aws.String(req.ContentType),
		}
		po, err := pre.PresignPutObject(context.TODO(), in, s3.WithPresignExpires(cfg.UploadExpires))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to presign", "details": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"method":       "PUT",
			"url":          po.URL,
			"headers":      po.SignedHeader,
			"expires_in":   int(cfg.UploadExpires.Seconds()),
			"key":          key,
			"bucket":       cfg.S3Bucket,
			"content_type": req.ContentType,
		})
	})

	// Download presign (GET)
	r.POST("/presign-get", func(c *gin.Context) {
		teamID, ok := authn(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		var req struct {
			Key string `json:"key" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request", "details": err.Error()})
			return
		}
		prefix := fmt.Sprintf("uploads/%s/", teamID)
		if !strings.HasPrefix(req.Key, prefix) {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden key"})
			return
		}

		_, pre, err := buildS3Clients(cfg)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "aws config error", "details": err.Error()})
			return
		}

		goi := &s3.GetObjectInput{Bucket: aws.String(cfg.S3Bucket), Key: aws.String(req.Key)}
		url, err := pre.PresignGetObject(context.TODO(), goi, s3.WithPresignExpires(cfg.GetExpires))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to presign", "details": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"method":     "GET",
			"url":        url.URL,
			"expires_in": int(cfg.GetExpires.Seconds()),
			"key":        req.Key,
			"bucket":     cfg.S3Bucket,
		})
	})

	// One-step multipart upload (server-side)
	r.POST("/upload-multipart", func(c *gin.Context) {
		teamID, ok := authn(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		fileHeader, err := c.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
			return
		}
		filename := fileHeader.Filename
		if !isAllowedFileType(filename, cfg.AllowedFileTypes) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "file type not allowed", "allowed_types": cfg.AllowedFileTypes})
			return
		}

		f, err := fileHeader.Open()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "failed to open file"})
			return
		}
		defer f.Close()

		s3c, _, err := buildS3Clients(cfg)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "aws config error", "details": err.Error()})
			return
		}

		ts := time.Now().Unix()
		key := fmt.Sprintf("uploads/%s/%d_%s", teamID, ts, filepath.Base(filename))
		contentType := c.PostForm("content_type")
		if contentType == "" {
			contentType = http.DetectContentType(make([]byte, 512))
		}

		_, err = s3c.PutObject(context.TODO(), &s3.PutObjectInput{
			Bucket:      aws.String(cfg.S3Bucket),
			Key:         aws.String(key),
			Body:        f,
			ContentType: aws.String(contentType),
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to upload", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message":      "uploaded",
			"bucket":       cfg.S3Bucket,
			"key":          key,
			"size":         fileHeader.Size,
			"content_type": contentType,
		})
	})

	addr := ":" + cfg.Port
	log.Printf("presign service listening on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatal(err)
	}
}
