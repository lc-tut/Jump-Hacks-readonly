package handlers

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	appconfig "github.com/digi-con/hackathon-template/internal/config"
	"github.com/gin-gonic/gin"
)

// GetImage returns image bytes from S3 as base64 in JSON. Public endpoint (no auth).
func GetImage(cfg *appconfig.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// When using wildcard route /image/*key Gin includes a leading '/', so strip it
		key := c.Param("key")
		if len(key) > 0 && key[0] == '/' {
			key = key[1:]
		}
		if key == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "key is required"})
			return
		}

		// Load AWS config
		awsCfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(cfg.AWS.Region))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load aws config", "details": err.Error()})
			return
		}

		s3Client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
			if cfg.AWS.EndpointURL != "" {
				o.BaseEndpoint = aws.String(cfg.AWS.EndpointURL)
				o.UsePathStyle = cfg.AWS.S3ForcePathStyle
			}
		})

		out, err := s3Client.GetObject(context.TODO(), &s3.GetObjectInput{
			Bucket: aws.String(cfg.AWS.S3BucketName),
			Key:    aws.String(key),
		})
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("failed to get object: %v", err)})
			return
		}
		defer out.Body.Close()

		// Read all bytes
		data, err := io.ReadAll(out.Body)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read object body", "details": err.Error()})
			return
		}

		encoded := base64.StdEncoding.EncodeToString(data)

		c.PureJSON(http.StatusOK, gin.H{
			"key":  key,
			"data": encoded,
		})
	}
}
