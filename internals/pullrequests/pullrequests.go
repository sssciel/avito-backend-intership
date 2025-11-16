package pullrequests

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sssciel/avito-backend-intership/internals/storage"
	"github.com/sssciel/avito-backend-intership/internals/storage/models"
)

type PullRequestService struct {
	RequestStorage    storage.RequestStorage
	TeamStorage       storage.TeamStorage
	UserStorage       storage.UserStorage
	GetAuthorTeamIDFn func(ctx context.Context, userID string) (int, error)
}

var pullRequestPrefix = "pullRequest"

func New(requestStorage storage.RequestStorage, teamStorage storage.TeamStorage, userStorage storage.UserStorage) *PullRequestService {
	s := &PullRequestService{
		RequestStorage: requestStorage,
		TeamStorage:    teamStorage,
		UserStorage:    userStorage,
	}
	s.GetAuthorTeamIDFn = s.getAuthorTeamID
	return s
}

func (s *PullRequestService) RegisterRoutes(r *gin.RouterGroup) {
	prRouter := r.Group("/" + pullRequestPrefix)

	prRouter.POST("/create", s.CreatePullRequest)
	prRouter.POST("/merge", s.MergePullRequest)
	prRouter.POST("/reassign", s.ReassignReviewer)
}

type CreatePRRequest struct {
	PullRequestID   string `json:"pull_request_id" binding:"required"`
	PullRequestName string `json:"pull_request_name" binding:"required"`
	AuthorID        string `json:"author_id" binding:"required"`
}

type MergePRRequest struct {
	PullRequestID string `json:"pull_request_id" binding:"required"`
}

type ReassignRequest struct {
	PullRequestID string `json:"pull_request_id" binding:"required"`
	OldReviewerID string `json:"old_reviewer_id" binding:"required"`
}

type PRResponse struct {
	PR models.PullRequest `json:"pr"`
}

type ReassignResponse struct {
	PR         models.PullRequest `json:"pr"`
	ReplacedBy string             `json:"replaced_by"`
}

func (s *PullRequestService) CreatePullRequest(c *gin.Context) {
	var req CreatePRRequest
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

	ctx := context.Background()

	teamID, err := s.GetAuthorTeamIDFn(ctx, req.AuthorID)
	if err != nil {
		if err.Error() == "NOT_FOUND" {
			c.JSON(http.StatusNotFound, gin.H{
				"error": gin.H{
					"code":    "NOT_FOUND",
					"message": "author or team not found",
				},
			})
			return
		}
		slog.Error("Failed to get author team", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "failed to get author team",
			},
		})
		return
	}

	reviewerIDs, err := s.TeamStorage.GetRandomReviewers(ctx, teamID, req.AuthorID, 2)
	if err != nil {
		slog.Error("Failed to get reviewers", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "failed to assign reviewers",
			},
		})
		return
	}

	pr := models.PullRequest{
		ID:       req.PullRequestID,
		Name:     req.PullRequestName,
		AuthorID: req.AuthorID,
	}

	createdPR, err := s.RequestStorage.CreatePullRequest(ctx, pr, reviewerIDs)
	if err != nil {
		if err.Error() == "PR_EXISTS" {
			c.JSON(http.StatusConflict, gin.H{
				"error": gin.H{
					"code":    "PR_EXISTS",
					"message": "PR id already exists",
				},
			})
			return
		}
		slog.Error("Failed to create PR", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "failed to create pull request",
			},
		})
		return
	}

	c.JSON(http.StatusCreated, PRResponse{PR: createdPR})
}

func (s *PullRequestService) MergePullRequest(c *gin.Context) {
	var req MergePRRequest
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

	mergedPR, err := s.RequestStorage.MergePullRequest(context.Background(), req.PullRequestID)
	if err != nil {
		if err.Error() == "NOT_FOUND" {
			c.JSON(http.StatusNotFound, gin.H{
				"error": gin.H{
					"code":    "NOT_FOUND",
					"message": "pull request not found",
				},
			})
			return
		}
		slog.Error("Failed to merge PR", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "failed to merge pull request",
			},
		})
		return
	}

	c.JSON(http.StatusOK, PRResponse{PR: mergedPR})
}

func (s *PullRequestService) ReassignReviewer(c *gin.Context) {
	var req ReassignRequest
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

	ctx := context.Background()

	teamID, err := s.GetAuthorTeamIDFn(ctx, req.OldReviewerID)
	if err != nil {
		if err.Error() == "NOT_FOUND" {
			c.JSON(http.StatusNotFound, gin.H{
				"error": gin.H{
					"code":    "NOT_FOUND",
					"message": "reviewer not found",
				},
			})
			return
		}
		slog.Error("Failed to get reviewer team", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "failed to get reviewer team",
			},
		})
		return
	}

	excludeIDs := []string{req.OldReviewerID}

	newReviewerID, err := s.TeamStorage.GetReplacementCandidate(ctx, teamID, excludeIDs)
	if err != nil {
		if err.Error() == "NO_CANDIDATE" {
			c.JSON(http.StatusConflict, gin.H{
				"error": gin.H{
					"code":    "NO_CANDIDATE",
					"message": "no active replacement candidate in team",
				},
			})
			return
		}
		slog.Error("Failed to get replacement candidate", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "failed to find replacement",
			},
		})
		return
	}

	updatedPR, err := s.RequestStorage.ReassignReviewer(ctx, req.PullRequestID, req.OldReviewerID, newReviewerID)
	if err != nil {
		if err.Error() == "NOT_FOUND" {
			c.JSON(http.StatusNotFound, gin.H{
				"error": gin.H{
					"code":    "NOT_FOUND",
					"message": "pull request not found",
				},
			})
			return
		}
		if err.Error() == "PR_MERGED" {
			c.JSON(http.StatusConflict, gin.H{
				"error": gin.H{
					"code":    "PR_MERGED",
					"message": "cannot reassign on merged PR",
				},
			})
			return
		}
		if err.Error() == "NOT_ASSIGNED" {
			c.JSON(http.StatusConflict, gin.H{
				"error": gin.H{
					"code":    "NOT_ASSIGNED",
					"message": "reviewer is not assigned to this PR",
				},
			})
			return
		}
		slog.Error("Failed to reassign reviewer", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "failed to reassign reviewer",
			},
		})
		return
	}

	c.JSON(http.StatusOK, ReassignResponse{
		PR:         updatedPR,
		ReplacedBy: newReviewerID,
	})
}

func (s *PullRequestService) getAuthorTeamID(ctx context.Context, userID string) (int, error) {
	return s.UserStorage.GetUserTeamID(ctx, userID)
}
