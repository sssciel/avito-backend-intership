package pgsql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/sssciel/avito-backend-intership/internals/storage/models"
)

type PGPullRequestStorage struct {
	DB *sqlx.DB
}

func (p *PGPullRequestStorage) CreatePullRequest(ctx context.Context, pr models.PullRequest, reviewerIDs []string) (models.PullRequest, error) {
	slog.Debug("Creating pull request in PG", "prID", pr.ID, "authorID", pr.AuthorID, "reviewers", reviewerIDs)

	tx, err := p.DB.BeginTxx(ctx, nil)
	if err != nil {
		return models.PullRequest{}, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	var exists bool
	err = tx.GetContext(ctx, &exists, "SELECT EXISTS(SELECT 1 FROM pull_requests WHERE pull_request_id = $1)", pr.ID)
	if err != nil {
		return models.PullRequest{}, fmt.Errorf("check PR exists: %w", err)
	}
	if exists {
		return models.PullRequest{}, errors.New("PR_EXISTS")
	}

	var prDBID int
	err = tx.QueryRowContext(ctx, `
  INSERT INTO pull_requests (pull_request_id, name, author_id, status)
  VALUES ($1, $2, $3, 'OPEN')
  RETURNING id
 `, pr.ID, pr.Name, pr.AuthorID).Scan(&prDBID)
	if err != nil {
		return models.PullRequest{}, fmt.Errorf("insert pull request: %w", err)
	}

	for _, reviewerID := range reviewerIDs {
		_, err = tx.ExecContext(ctx, `
   INSERT INTO pull_request_reviewers (pull_request_id, reviewer_id)
   VALUES ($1, $2)
  `, prDBID, reviewerID)
		if err != nil {
			return models.PullRequest{}, fmt.Errorf("assign reviewer %s: %w", reviewerID, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return models.PullRequest{}, fmt.Errorf("commit transaction: %w", err)
	}

	pr.Status = "OPEN"
	pr.AssignedReviewers = reviewerIDs
	pr.MergedAt = sql.NullTime{}
	return pr, nil
}

func (p *PGPullRequestStorage) MergePullRequest(ctx context.Context, pullRequestID string) (models.PullRequest, error) {
	slog.Debug("Merging pull request in PG", "prID", pullRequestID)

	tx, err := p.DB.BeginTxx(ctx, nil)
	if err != nil {
		return models.PullRequest{}, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	var prDBID int
	var pr models.PullRequest
	err = tx.QueryRowContext(ctx, `
  SELECT id, pull_request_id, name, author_id, status, merged_at
  FROM pull_requests
  WHERE pull_request_id = $1
 `, pullRequestID).Scan(&prDBID, &pr.ID, &pr.Name, &pr.AuthorID, &pr.Status, &pr.MergedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.PullRequest{}, errors.New("NOT_FOUND")
		}
		return models.PullRequest{}, fmt.Errorf("get pull request: %w", err)
	}

	if pr.Status != "MERGED" {
		_, err = tx.ExecContext(ctx, `
   UPDATE pull_requests
   SET status = 'MERGED', merged_at = NOW()
   WHERE pull_request_id = $1
  `, pullRequestID)
		if err != nil {
			return models.PullRequest{}, fmt.Errorf("update PR status: %w", err)
		}
		pr.Status = "MERGED"
		pr.MergedAt = sql.NullTime{Time: time.Now(), Valid: true}
	}

	err = tx.SelectContext(ctx, &pr.AssignedReviewers, `
  SELECT reviewer_id FROM pull_request_reviewers WHERE pull_request_id = $1
 `, prDBID)
	if err != nil {
		return models.PullRequest{}, fmt.Errorf("get reviewers: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return models.PullRequest{}, fmt.Errorf("commit transaction: %w", err)
	}

	return pr, nil
}

func (p *PGPullRequestStorage) ReassignReviewer(ctx context.Context, pullRequestID string, oldReviewerID string, newReviewerID string) (models.PullRequest, error) {
	slog.Debug("Reassigning reviewer in PG", "prID", pullRequestID, "oldID", oldReviewerID, "newID", newReviewerID)

	tx, err := p.DB.BeginTxx(ctx, nil)
	if err != nil {
		return models.PullRequest{}, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	var prDBID int
	var pr models.PullRequest
	err = tx.QueryRowContext(ctx, `
  SELECT id, pull_request_id, name, author_id, status, merged_at
  FROM pull_requests
  WHERE pull_request_id = $1
 `, pullRequestID).Scan(&prDBID, &pr.ID, &pr.Name, &pr.AuthorID, &pr.Status, &pr.MergedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.PullRequest{}, errors.New("NOT_FOUND")
		}
		return models.PullRequest{}, fmt.Errorf("get pull request: %w", err)
	}

	if pr.Status == "MERGED" {
		return models.PullRequest{}, errors.New("PR_MERGED")
	}

	var isAssigned bool
	err = tx.GetContext(ctx, &isAssigned, `
  SELECT EXISTS(SELECT 1 FROM pull_request_reviewers WHERE pull_request_id = $1 AND reviewer_id = $2)
 `, prDBID, oldReviewerID)
	if err != nil {
		return models.PullRequest{}, fmt.Errorf("check assignment: %w", err)
	}
	if !isAssigned {
		return models.PullRequest{}, errors.New("NOT_ASSIGNED")
	}

	_, err = tx.ExecContext(ctx, `
  DELETE FROM pull_request_reviewers
  WHERE pull_request_id = $1 AND reviewer_id = $2
 `, prDBID, oldReviewerID)
	if err != nil {
		return models.PullRequest{}, fmt.Errorf("delete old reviewer: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
  INSERT INTO pull_request_reviewers (pull_request_id, reviewer_id)
  VALUES ($1, $2)
  ON CONFLICT (pull_request_id, reviewer_id) DO NOTHING
 `, prDBID, newReviewerID)
	if err != nil {
		return models.PullRequest{}, fmt.Errorf("insert new reviewer: %w", err)
	}

	err = tx.SelectContext(ctx, &pr.AssignedReviewers, `
  SELECT reviewer_id FROM pull_request_reviewers WHERE pull_request_id = $1
 `, prDBID)
	if err != nil {
		return models.PullRequest{}, fmt.Errorf("get reviewers: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return models.PullRequest{}, fmt.Errorf("commit: %w", err)
	}

	return pr, nil
}
