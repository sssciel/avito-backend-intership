package users

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sssciel/avito-backend-intership/internals/storage"
	"github.com/sssciel/avito-backend-intership/internals/storage/models"
)

type UserService struct {
	UserStorage storage.UserStorage
	TeamStorage storage.TeamStorage
}

var usersPrefix = "users"

func New(userStorage storage.UserStorage, teamStorage storage.TeamStorage) *UserService {
	return &UserService{
		UserStorage: userStorage,
		TeamStorage: teamStorage,
	}
}

func (s *UserService) RegisterRoutes(r *gin.RouterGroup) {
	userRouter := r.Group("/" + usersPrefix)

	userRouter.POST("/setIsActive", s.SetIsActive)
	userRouter.GET("/getReview", s.GetUserReviews)
}

type SetIsActiveRequest struct {
	UserID   string `json:"user_id" binding:"required"`
	IsActive bool   `json:"is_active"`
}

type UserResponse struct {
	User UserWithTeam `json:"user"`
}

type UserWithTeam struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	TeamName string `json:"team_name"`
	IsActive bool   `json:"is_active"`
}

type GetReviewsResponse struct {
	UserID       string               `json:"user_id"`
	PullRequests []models.PullRequest `json:"pull_requests"`
}

func (s *UserService) SetIsActive(c *gin.Context) {
	var req SetIsActiveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		slog.Error("Invalid request body", "err", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": err.Error(),
			},
		})
		return
	}

	err := s.UserStorage.SetIsActive(context.Background(), req.UserID, req.IsActive)
	if err != nil {
		if err.Error() == "NOT_FOUND" {
			c.JSON(http.StatusNotFound, gin.H{
				"error": gin.H{
					"code":    "NOT_FOUND",
					"message": "user not found",
				},
			})
			return
		}
		slog.Error("Failed to set user active status", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "failed to update user",
			},
		})
		return
	}

	user, teamName, err := s.getUserWithTeam(req.UserID)
	if err != nil {
		slog.Error("Failed to get updated user data", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "failed to get updated user data",
			},
		})
		return
	}

	c.JSON(http.StatusOK, UserResponse{
		User: UserWithTeam{
			UserID:   user.ID,
			Username: user.Username,
			TeamName: teamName,
			IsActive: req.IsActive,
		},
	})
}

func (s *UserService) GetUserReviews(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "user_id query parameter is required",
			},
		})
		return
	}

	pullRequests, err := s.UserStorage.GetUserReviews(context.Background(), userID)
	if err != nil {
		slog.Error("Failed to get user reviews", "err", err, "userID", userID)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "failed to get user reviews",
			},
		})
		return
	}

	c.JSON(http.StatusOK, GetReviewsResponse{
		UserID:       userID,
		PullRequests: pullRequests,
	})
}

func (s *UserService) getUserWithTeam(userID string) (models.User, string, error) {
	return models.User{ID: userID}, "", nil
}
