package handlers

import (
	"net/http"
	"strconv"

	"github.com/digi-con/hackathon-template/internal/database"
	"github.com/digi-con/hackathon-template/internal/middleware"
	"github.com/gin-gonic/gin"
)

// GetProfile returns the current user's profile
func GetProfile(db database.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := middleware.GetUserID(c)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "User not authenticated",
			})
			return
		}

		var user database.User
		if err := db.GetDB().First(&user, userID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "User not found",
			})
			return
		}

		c.JSON(http.StatusOK, user.ToResponse())
	}
}

// UpdateProfile updates the current user's profile
func UpdateProfile(db database.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := middleware.GetUserID(c)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "User not authenticated",
			})
			return
		}

		var req database.UpdateProfileRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid request data",
				"details": err.Error(),
			})
			return
		}

		// Get current user
		var user database.User
		if err := db.GetDB().First(&user, userID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "User not found",
			})
			return
		}

		// Update fields if provided
		updates := make(map[string]interface{})
		if req.FirstName != "" {
			updates["first_name"] = req.FirstName
		}
		if req.LastName != "" {
			updates["last_name"] = req.LastName
		}
		if req.Avatar != "" {
			updates["avatar"] = req.Avatar
		}

		if len(updates) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "No valid fields to update",
			})
			return
		}

		// Update user
		if err := db.GetDB().Model(&user).Updates(updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to update profile",
			})
			return
		}

		// Fetch updated user
		if err := db.GetDB().First(&user, userID).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to fetch updated profile",
			})
			return
		}

		c.JSON(http.StatusOK, user.ToResponse())
	}
}

// DeleteProfile deactivates the current user's account
func DeleteProfile(db database.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := middleware.GetUserID(c)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "User not authenticated",
			})
			return
		}

		// Deactivate user instead of hard delete
		if err := db.GetDB().Model(&database.User{}).Where("id = ?", userID).Update("is_active", false).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to deactivate account",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Account deactivated successfully",
		})
	}
}

// ListUsers returns a list of users (admin only)
func ListUsers(db database.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get pagination parameters
		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
		
		if page < 1 {
			page = 1
		}
		if limit < 1 || limit > 100 {
			limit = 10
		}

		offset := (page - 1) * limit

		// Get search parameter
		search := c.Query("search")

		var users []database.User
		var total int64

		query := db.GetDB().Model(&database.User{})

		// Apply search filter
		if search != "" {
			searchPattern := "%" + search + "%"
			query = query.Where("email ILIKE ? OR first_name ILIKE ? OR last_name ILIKE ?", 
				searchPattern, searchPattern, searchPattern)
		}

		// Get total count
		query.Count(&total)

		// Get users with pagination
		if err := query.Offset(offset).Limit(limit).Find(&users).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to fetch users",
			})
			return
		}

		// Convert to response format
		userResponses := make([]database.UserResponse, len(users))
		for i, user := range users {
			userResponses[i] = user.ToResponse()
		}

		c.JSON(http.StatusOK, gin.H{
			"users": userResponses,
			"pagination": gin.H{
				"page":       page,
				"limit":      limit,
				"total":      total,
				"total_pages": (total + int64(limit) - 1) / int64(limit),
			},
		})
	}
}

// GetStats returns user statistics (admin only)
func GetStats(db database.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var stats struct {
			TotalUsers   int64 `json:"total_users"`
			ActiveUsers  int64 `json:"active_users"`
			AdminUsers   int64 `json:"admin_users"`
			TotalFiles   int64 `json:"total_files"`
		}

		// Get user counts
		db.GetDB().Model(&database.User{}).Count(&stats.TotalUsers)
		db.GetDB().Model(&database.User{}).Where("is_active = ?", true).Count(&stats.ActiveUsers)
		db.GetDB().Model(&database.User{}).Where("role = ?", "admin").Count(&stats.AdminUsers)
		
		// Get file count
		db.GetDB().Model(&database.File{}).Count(&stats.TotalFiles)

		c.JSON(http.StatusOK, stats)
	}
} 