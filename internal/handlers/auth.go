package handlers

import (
	"net/http"
	"strings"

	"github.com/digi-con/hackathon-template/internal/auth"
	"github.com/digi-con/hackathon-template/internal/config"
	"github.com/digi-con/hackathon-template/internal/database"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Register handles user registration
func Register(db database.DB, cfg *config.Config) gin.HandlerFunc {
	authService := auth.NewService(cfg)

	return func(c *gin.Context) {
		var req database.RegisterRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid request data",
				"details": err.Error(),
			})
			return
		}

		// Normalize email
		req.Email = strings.ToLower(strings.TrimSpace(req.Email))

		// Check if user already exists
		var existingUser database.User
		if err := db.GetDB().Where("email = ?", req.Email).First(&existingUser).Error; err == nil {
			c.JSON(http.StatusConflict, gin.H{
				"error": "User with this email already exists",
			})
			return
		}

		// Hash password
		hashedPassword, err := auth.HashPassword(req.Password)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid password",
				"details": err.Error(),
			})
			return
		}

		// Create user
		user := database.User{
			Email:     req.Email,
			Password:  hashedPassword,
			FirstName: req.FirstName,
			LastName:  req.LastName,
			Role:      "user", // Default role
			IsActive:  true,
		}

		if err := db.GetDB().Create(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to create user",
			})
			return
		}

		// Generate JWT token
		token, err := authService.GenerateToken(&user)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to generate token",
			})
			return
		}

		response := database.LoginResponse{
			User:  user.ToResponse(),
			Token: token,
		}

		c.JSON(http.StatusCreated, response)
	}
}

// Login handles user authentication
func Login(db database.DB, cfg *config.Config) gin.HandlerFunc {
	authService := auth.NewService(cfg)

	return func(c *gin.Context) {
		var req database.LoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid request data",
				"details": err.Error(),
			})
			return
		}

		// Normalize email
		req.Email = strings.ToLower(strings.TrimSpace(req.Email))

		// Find user by email
		var user database.User
		if err := db.GetDB().Where("email = ?", req.Email).First(&user).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(http.StatusUnauthorized, gin.H{
					"error": "Invalid email or password",
				})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Database error",
			})
			return
		}

		// Check if user is active
		if !user.IsActive {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Account is deactivated",
			})
			return
		}

		// Verify password
		if err := auth.CheckPassword(user.Password, req.Password); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid email or password",
			})
			return
		}

		// Generate JWT token
		token, err := authService.GenerateToken(&user)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to generate token",
			})
			return
		}

		response := database.LoginResponse{
			User:  user.ToResponse(),
			Token: token,
		}

		c.JSON(http.StatusOK, response)
	}
}

// RefreshToken handles token refresh
func RefreshToken(db database.DB, cfg *config.Config) gin.HandlerFunc {
	authService := auth.NewService(cfg)

	return func(c *gin.Context) {
		var req struct {
			Token string `json:"token" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Token is required",
			})
			return
		}

		// Refresh token
		newToken, err := authService.RefreshToken(req.Token, db)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid or expired token",
				"details": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"token": newToken,
		})
	}
}

// Logout handles user logout (for stateless JWT, this is mainly client-side)
func Logout() gin.HandlerFunc {
	return func(c *gin.Context) {
		// In a stateless JWT system, logout is typically handled client-side
		// by removing the token from storage. However, you could implement
		// a token blacklist here if needed.

		c.JSON(http.StatusOK, gin.H{
			"message": "Successfully logged out",
		})
	}
}

// ChangePassword allows authenticated users to change their password
func ChangePassword(db database.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "User not authenticated",
			})
			return
		}

		var req struct {
			CurrentPassword string `json:"current_password" binding:"required"`
			NewPassword     string `json:"new_password" binding:"required,min=8"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid request data",
				"details": err.Error(),
			})
			return
		}

		// Get user from database
		var user database.User
		if err := db.GetDB().First(&user, userID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "User not found",
			})
			return
		}

		// Verify current password
		if err := auth.CheckPassword(user.Password, req.CurrentPassword); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Current password is incorrect",
			})
			return
		}

		// Hash new password
		hashedPassword, err := auth.HashPassword(req.NewPassword)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid new password",
				"details": err.Error(),
			})
			return
		}

		// Update password
		if err := db.GetDB().Model(&user).Update("password", hashedPassword).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to update password",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Password updated successfully",
		})
	}
} 