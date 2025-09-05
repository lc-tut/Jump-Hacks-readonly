package handlers

import (
	"context"
	"fmt"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	appconfig "github.com/digi-con/hackathon-template/internal/config"
	"github.com/digi-con/hackathon-template/internal/database"
	"github.com/digi-con/hackathon-template/internal/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// UploadFile handles file upload to S3
func UploadFile(db database.DB, cfg *appconfig.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := middleware.GetUserID(c)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "User not authenticated",
			})
			return
		}

		// Parse multipart form
		file, header, err := c.Request.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "No file provided",
				"details": err.Error(),
			})
			return
		}
		defer file.Close()

		// Validate file size
		maxSize := int64(cfg.Upload.MaxFileSizeMB * 1024 * 1024) // Convert MB to bytes
		if header.Size > maxSize {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("File size exceeds maximum allowed size of %dMB", cfg.Upload.MaxFileSizeMB),
			})
			return
		}

		// Validate file type
		if !isAllowedFileType(header.Filename, cfg.Upload.AllowedFileTypes) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":         "File type not allowed",
				"allowed_types": cfg.Upload.AllowedFileTypes,
			})
			return
		}

		// Upload to S3
		s3URL, s3Key, err := uploadToS3(file, header, cfg)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to upload file",
				"details": err.Error(),
			})
			return
		}

		// Save file metadata to database
		fileRecord := database.File{
			UserID:       userID,
			FileName:     generateFileName(header.Filename),
			OriginalName: header.Filename,
			MimeType:     header.Header.Get("Content-Type"),
			Size:         header.Size,
			S3Key:        s3Key,
			S3Bucket:     cfg.AWS.S3BucketName,
			URL:          s3URL,
			IsPublic:     false, // Default to private
		}

		if err := db.GetDB().Create(&fileRecord).Error; err != nil {
			// If database save fails, attempt to clean up S3 file
			deleteFromS3(s3Key, cfg)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to save file metadata",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "File uploaded successfully",
			"file": gin.H{
				"id":            fileRecord.ID,
				"filename":      fileRecord.FileName,
				"original_name": fileRecord.OriginalName,
				"size":          fileRecord.Size,
				"url":           fileRecord.URL,
				"mime_type":     fileRecord.MimeType,
			},
		})
	}
}

// ListFiles returns user's uploaded files
func ListFiles(db database.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := middleware.GetUserID(c)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "User not authenticated",
			})
			return
		}

		// Get pagination parameters
		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

		if page < 1 {
			page = 1
		}
		if limit < 1 || limit > 50 {
			limit = 10
		}

		offset := (page - 1) * limit

		var files []database.File
		var total int64

		// Get total count first
		if err := db.GetDB().Model(&database.File{}).Where("user_id = ?", userID).Count(&total).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to count files",
			})
			return
		}

		// Get user's files with pagination
		if err := db.GetDB().Where("user_id = ?", userID).Offset(offset).Limit(limit).Order("created_at DESC").Find(&files).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to fetch files",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"files": files,
			"pagination": gin.H{
				"page":        page,
				"limit":       limit,
				"total":       total,
				"total_pages": (total + int64(limit) - 1) / int64(limit),
			},
		})
	}
}

// DeleteFile removes a file from S3 and database
func DeleteFile(db database.DB, cfg *appconfig.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := middleware.GetUserID(c)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "User not authenticated",
			})
			return
		}

		fileID := c.Param("id")
		if fileID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "File ID is required",
			})
			return
		}

		// Get file from database
		var file database.File
		if err := db.GetDB().Where("id = ? AND user_id = ?", fileID, userID).First(&file).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "File not found",
			})
			return
		}

		// Delete from S3
		if err := deleteFromS3(file.S3Key, cfg); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to delete file from storage",
				"details": err.Error(),
			})
			return
		}

		// Delete from database
		if err := db.GetDB().Delete(&file).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to delete file record",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "File deleted successfully",
		})
	}
}

// PresignUpload returns a presigned S3 PUT URL
func PresignUpload(db database.DB, cfg *appconfig.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := middleware.GetUserID(c)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "User not authenticated",
			})
			return
		}

		var req struct {
			FileName    string `json:"filename" binding:"required"`
			ContentType string `json:"content_type"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid request data",
				"details": err.Error(),
			})
			return
		}

		if !isAllowedFileType(req.FileName, cfg.Upload.AllowedFileTypes) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":         "File type not allowed",
				"allowed_types": cfg.Upload.AllowedFileTypes,
			})
			return
		}

		awsCfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(cfg.AWS.Region))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to load AWS config",
				"details": err.Error(),
			})
			return
		}

		s3Client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
			if cfg.AWS.EndpointURL != "" {
				o.BaseEndpoint = aws.String(cfg.AWS.EndpointURL)
				o.UsePathStyle = cfg.AWS.S3ForcePathStyle
			}
		})

		// Use a public endpoint for presigned URLs (host visible to clients) if provided
		presignBaseClient := s3Client
		if cfg.AWS.PublicEndpointURL != "" {
			presignBaseClient = s3.NewFromConfig(awsCfg, func(o *s3.Options) {
				o.BaseEndpoint = aws.String(cfg.AWS.PublicEndpointURL)
				o.UsePathStyle = cfg.AWS.S3ForcePathStyle
			})
		}

		presigner := s3.NewPresignClient(presignBaseClient)

		key := fmt.Sprintf("uploads/user-%d/%s", userID, generateFileName(req.FileName))
		input := &s3.PutObjectInput{
			Bucket:      aws.String(cfg.AWS.S3BucketName),
			Key:         aws.String(key),
			ContentType: aws.String(req.ContentType),
		}

		presigned, err := presigner.PresignPutObject(context.TODO(), input, s3.WithPresignExpires(10*time.Minute))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to generate presigned URL",
				"details": err.Error(),
			})
			return
		}

		c.PureJSON(http.StatusOK, gin.H{
			"method":       "PUT",
			"url":          presigned.URL,
			"headers":      presigned.SignedHeader,
			"expires_in":   int(10 * 60),
			"key":          key,
			"bucket":       cfg.AWS.S3BucketName,
			"content_type": req.ContentType,
		})
	}
}

// PresignGet returns a presigned S3 GET URL for a specific file
func PresignGet(db database.DB, cfg *appconfig.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := middleware.GetUserID(c)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "User not authenticated",
			})
			return
		}

		var req struct {
			ID uint `json:"id" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid request data",
				"details": err.Error(),
			})
			return
		}

		var fileRec database.File
		if err := db.GetDB().Where("id = ? AND user_id = ?", req.ID, userID).First(&fileRec).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "File not found",
			})
			return
		}

		awsCfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(cfg.AWS.Region))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to load AWS config",
				"details": err.Error(),
			})
			return
		}

		s3Client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
			if cfg.AWS.EndpointURL != "" {
				o.BaseEndpoint = aws.String(cfg.AWS.EndpointURL)
				o.UsePathStyle = cfg.AWS.S3ForcePathStyle
			}
		})
		// Use a public endpoint for presigned URLs if provided
		presignBaseClient := s3Client
		if cfg.AWS.PublicEndpointURL != "" {
			presignBaseClient = s3.NewFromConfig(awsCfg, func(o *s3.Options) {
				o.BaseEndpoint = aws.String(cfg.AWS.PublicEndpointURL)
				o.UsePathStyle = cfg.AWS.S3ForcePathStyle
			})
		}
		presigner := s3.NewPresignClient(presignBaseClient)

		out, err := presigner.PresignGetObject(context.TODO(), &s3.GetObjectInput{
			Bucket: aws.String(cfg.AWS.S3BucketName),
			Key:    aws.String(fileRec.S3Key),
		}, s3.WithPresignExpires(10*time.Minute))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to generate presigned URL",
				"details": err.Error(),
			})
			return
		}

		c.PureJSON(http.StatusOK, gin.H{
			"method":     "GET",
			"url":        out.URL,
			"expires_in": int(10 * 60),
			"key":        fileRec.S3Key,
			"bucket":     cfg.AWS.S3BucketName,
		})
	}
}

// Helper functions

func isAllowedFileType(filename string, allowedTypes []string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	if ext != "" {
		ext = ext[1:] // Remove the dot
	}

	for _, allowedType := range allowedTypes {
		if ext == strings.ToLower(allowedType) {
			return true
		}
	}
	return false
}

func generateFileName(originalName string) string {
	ext := filepath.Ext(originalName)
	uuid := uuid.New().String()
	timestamp := time.Now().Unix()
	return fmt.Sprintf("%d_%s%s", timestamp, uuid, ext)
}

func uploadToS3(file multipart.File, header *multipart.FileHeader, cfg *appconfig.Config) (string, string, error) {
	// Create AWS config
	awsCfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(cfg.AWS.Region),
	)
	if err != nil {
		return "", "", fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create S3 client (supports LocalStack/custom endpoint)
	s3Client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if cfg.AWS.EndpointURL != "" {
			o.BaseEndpoint = aws.String(cfg.AWS.EndpointURL)
			o.UsePathStyle = cfg.AWS.S3ForcePathStyle
		}
	})

	// Generate unique key
	key := fmt.Sprintf("uploads/%s", generateFileName(header.Filename))

	// Upload file
	_, err = s3Client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket:      aws.String(cfg.AWS.S3BucketName),
		Key:         aws.String(key),
		Body:        file,
		ContentType: aws.String(header.Header.Get("Content-Type")),
	})
	if err != nil {
		return "", "", fmt.Errorf("failed to upload to S3: %w", err)
	}

	// Generate URL (for reference)
	url := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s",
		cfg.AWS.S3BucketName, cfg.AWS.Region, key)

	return url, key, nil
}

func deleteFromS3(key string, cfg *appconfig.Config) error {
	// Create AWS config
	awsCfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(cfg.AWS.Region),
	)
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create S3 client (supports LocalStack/custom endpoint)
	s3Client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if cfg.AWS.EndpointURL != "" {
			o.BaseEndpoint = aws.String(cfg.AWS.EndpointURL)
			o.UsePathStyle = cfg.AWS.S3ForcePathStyle
		}
	})

	// Delete object
	_, err = s3Client.DeleteObject(context.TODO(), &s3.DeleteObjectInput{
		Bucket: aws.String(cfg.AWS.S3BucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to delete from S3: %w", err)
	}

	return nil
}
