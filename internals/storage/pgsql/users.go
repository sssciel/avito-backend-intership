package pgsql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"github.com/jmoiron/sqlx"
	"github.com/sssciel/avito-backend-intership/internals/storage/models"
)

type PGUserStorage struct {
	DB *sqlx.DB
}

func (p *PGUserStorage) SetIsActive(ctx context.Context, userID string, isActive bool) error {
	slog.Debug("Setting user active status in PG", "userID", userID, "isActive", isActive)

	result, err := p.DB.ExecContext(ctx, "UPDATE users SET is_active = $1 WHERE user_id = $2", isActive, userID)
	if err != nil {
		slog.Error("SQL update user active status error", "err", err)
		return fmt.Errorf("failed to update user active status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("NOT_FOUND")
	}

	return nil
}

func (p *PGUserStorage) GetIsActive(ctx context.Context, userID string) (bool, error) {
	slog.Debug("Getting user active status in PG", "userID", userID)

	var isActive bool
	err := p.DB.GetContext(ctx, &isActive, "SELECT is_active FROM users WHERE user_id = $1", userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, errors.New("NOT_FOUND")
		}
		slog.Error("SQL get user active status error", "err", err)
		return false, fmt.Errorf("failed to get user active status: %w", err)
	}

	return isActive, nil
}

func (p *PGUserStorage) GetUserReviews(ctx context.Context, userID string) ([]models.PullRequest, error) {
	slog.Debug("Getting user reviews in PG", "userID", userID)

	var pullRequests []models.PullRequest

	err := p.DB.SelectContext(ctx, &pullRequests, `
  SELECT DISTINCT pr.pull_request_id, pr.name, pr.author_id, pr.status, pr.merged_at, pr.created_at
  FROM pull_requests pr
  INNER JOIN pull_request_reviewers prr ON pr.id = prr.pull_request_id
  WHERE prr.reviewer_id = $1
  ORDER BY pr.created_at DESC
 `, userID)
	if err != nil {
		slog.Error("SQL get user reviews error", "err", err)
		return nil, fmt.Errorf("failed to get user reviews: %w", err)
	}

	for i := range pullRequests {
		var reviewers []string
		err = p.DB.SelectContext(ctx, &reviewers, `
   SELECT reviewer_id 
   FROM pull_request_reviewers 
   WHERE pull_request_id = (SELECT id FROM pull_requests WHERE pull_request_id = $1)
  `, pullRequests[i].ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get reviewers for PR %s: %w", pullRequests[i].ID, err)
		}
		pullRequests[i].AssignedReviewers = reviewers
	}

	return pullRequests, nil
}

func (p *PGUserStorage) GetUserTeamID(ctx context.Context, userID string) (int, error) {
	slog.Debug("Getting user team ID in PG", "userID", userID)

	var teamID int
	err := p.DB.GetContext(ctx, &teamID, `
		SELECT team_id 
		FROM team_members 
		WHERE user_id = $1 
		LIMIT 1
	`, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, errors.New("NOT_FOUND")
		}
		slog.Error("SQL get user team ID error", "err", err)
		return 0, fmt.Errorf("failed to get user team ID: %w", err)
	}

	return teamID, nil
}
