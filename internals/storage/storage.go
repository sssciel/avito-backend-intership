package storage

import (
	"context"

	"github.com/sssciel/avito-backend-intership/internals/storage/models"
)

type TeamStorage interface {
	AddTeam(ctx context.Context, team models.Team) (models.Team, error)
	GetTeamByName(ctx context.Context, name string) (models.Team, error)
	GetRandomReviewers(ctx context.Context, teamID int, authorID string, limit int) ([]string, error)
	GetReplacementCandidate(ctx context.Context, teamID int, excludeUserIDs []string) (string, error)
}

type RequestStorage interface {
	CreatePullRequest(ctx context.Context, pr models.PullRequest, reviewerIDs []string) (models.PullRequest, error)
	MergePullRequest(ctx context.Context, pullrequestID string) (models.PullRequest, error)
	ReassignReviewer(ctx context.Context, pullrequestID string, oldReviewerID string, newReviewerID string) (models.PullRequest, error)
}
type UserStorage interface {
	SetIsActive(ctx context.Context, userID string, isActive bool) error
	GetIsActive(ctx context.Context, userID string) (bool, error)
	GetUserReviews(ctx context.Context, userID string) ([]models.PullRequest, error)
	GetUserTeamID(ctx context.Context, userID string) (int, error)
}
