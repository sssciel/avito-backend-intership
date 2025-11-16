package models

import (
	"database/sql"
	"time"
)

type PullRequest struct {
	ID                string       `json:"pull_request_id" db:"pull_request_id"`
	Name              string       `json:"pull_request_name" db:"name"`
	AuthorID          string       `json:"author_id" db:"author_id"`
	Status            string       `json:"status" db:"status"`
	AssignedReviewers []string     `json:"assigned_reviewers" db:"-"`
	MergedAt          sql.NullTime `json:"merged_at,omitempty" db:"merged_at"`
	CreatedAt         time.Time    `json:"created_at" db:"created_at"`
}
