package teams

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sssciel/avito-backend-intership/internals/storage"
	"github.com/sssciel/avito-backend-intership/internals/storage/models"
)

type TeamService struct {
	TeamStorage storage.TeamStorage
	UserStorage storage.UserStorage
}

var teamsPrefix = "team"

func New(teamStorage storage.TeamStorage, userStorage storage.UserStorage) *TeamService {
	return &TeamService{
		TeamStorage: teamStorage,
		UserStorage: userStorage,
	}
}

func (s *TeamService) RegisterRoutes(r *gin.RouterGroup) {
	teamRouter := r.Group("/" + teamsPrefix)

	teamRouter.POST("/add", s.AddTeam)
	teamRouter.GET("/get", s.GetTeam)
}

type AddTeamRequest struct {
	TeamName string        `json:"team_name" binding:"required"`
	Members  []models.User `json:"members" binding:"required,min=1"`
}

type TeamResponse struct {
	Team models.Team `json:"team"`
}

func (s *TeamService) AddTeam(c *gin.Context) {
	var req AddTeamRequest
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

	team := models.Team{
		Name:    req.TeamName,
		Members: req.Members,
	}

	createdTeam, err := s.TeamStorage.AddTeam(context.Background(), team)
	if err != nil {
		if err.Error() == "TEAM_EXISTS" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{
					"code":    "TEAM_EXISTS",
					"message": "team_name already exists",
				},
			})
			return
		}
		slog.Error("Failed to create team", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "failed to create team",
			},
		})
		return
	}

	c.JSON(http.StatusCreated, TeamResponse{Team: createdTeam})
}

func (s *TeamService) GetTeam(c *gin.Context) {
	teamName := c.Query("team_name")
	if teamName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "team_name query parameter is required",
			},
		})
		return
	}

	team, err := s.TeamStorage.GetTeamByName(context.Background(), teamName)
	if err != nil {
		if err.Error() == "NOT_FOUND" {
			c.JSON(http.StatusNotFound, gin.H{
				"error": gin.H{
					"code":    "NOT_FOUND",
					"message": "team not found",
				},
			})
			return
		}
		slog.Error("Failed to get team", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "failed to get team",
			},
		})
		return
	}

	c.JSON(http.StatusOK, TeamResponse{Team: team})
}
