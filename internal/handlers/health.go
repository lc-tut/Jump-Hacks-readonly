package handlers

import (
	"net/http"
	"time"

	"github.com/digi-con/hackathon-template/internal/database"
	"github.com/gin-gonic/gin"
)

type HealthStatus struct {
	Status    string            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
	Version   string            `json:"version"`
	Services  map[string]string `json:"services"`
}

// HealthCheck returns the health status of the application
func HealthCheck(db database.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		health := HealthStatus{
			Status:    "healthy",
			Timestamp: time.Now(),
			Version:   "1.0.0", // This could be injected from build
			Services:  make(map[string]string),
		}

		// Check database health
		if err := db.Health(); err != nil {
			health.Status = "unhealthy"
			health.Services["database"] = "unhealthy: " + err.Error()
			c.JSON(http.StatusServiceUnavailable, health)
			return
		}
		health.Services["database"] = "healthy"

		// Add more service checks here (Redis, S3, etc.)
		// health.Services["redis"] = "healthy"
		// health.Services["s3"] = "healthy"

		c.JSON(http.StatusOK, health)
	}
}

// ReadinessCheck checks if the application is ready to serve requests
func ReadinessCheck(db database.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if database is ready
		if err := db.Health(); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "not ready",
				"error":  "database not ready",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status": "ready",
		})
	}
}

// LivenessCheck checks if the application is alive
func LivenessCheck() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "alive",
			"timestamp": time.Now(),
		})
	}
} 