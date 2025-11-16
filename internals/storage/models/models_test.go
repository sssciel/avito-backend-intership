package models

import (
	"database/sql"
	"testing"
	"time"
)

func TestPullRequest_Fields(t *testing.T) {
	pr := PullRequest{
		ID:                "pr-1",
		Name:              "Test PR",
		AuthorID:          "user-1",
		Status:            "OPEN",
		AssignedReviewers: []string{"user-2", "user-3"},
		MergedAt:          sql.NullTime{},
	}

	if pr.ID != "pr-1" {
		t.Errorf("Expected ID to be 'pr-1', got %s", pr.ID)
	}
	if pr.Name != "Test PR" {
		t.Errorf("Expected Name to be 'Test PR', got %s", pr.Name)
	}
	if pr.AuthorID != "user-1" {
		t.Errorf("Expected AuthorID to be 'user-1', got %s", pr.AuthorID)
	}
	if pr.Status != "OPEN" {
		t.Errorf("Expected Status to be 'OPEN', got %s", pr.Status)
	}
	if len(pr.AssignedReviewers) != 2 {
		t.Errorf("Expected 2 reviewers, got %d", len(pr.AssignedReviewers))
	}
	if pr.MergedAt.Valid {
		t.Error("Expected MergedAt to be null for OPEN PR")
	}
}

func TestPullRequest_MergedState(t *testing.T) {
	now := time.Now()
	pr := PullRequest{
		ID:                "pr-2",
		Name:              "Merged PR",
		AuthorID:          "user-1",
		Status:            "MERGED",
		AssignedReviewers: []string{"user-2"},
		MergedAt:          sql.NullTime{Time: now, Valid: true},
	}

	if pr.Status != "MERGED" {
		t.Errorf("Expected Status to be 'MERGED', got %s", pr.Status)
	}
	if !pr.MergedAt.Valid {
		t.Error("Expected MergedAt to be valid for MERGED PR")
	}
	if pr.MergedAt.Time != now {
		t.Errorf("Expected MergedAt to be %v, got %v", now, pr.MergedAt.Time)
	}
}

func TestTeam_Fields(t *testing.T) {
	team := Team{
		ID:   1,
		Name: "Backend Team",
		Members: []User{
			{ID: "user-1", Username: "Alice", IsActive: true},
			{ID: "user-2", Username: "Bob", IsActive: false},
		},
	}

	if team.ID != 1 {
		t.Errorf("Expected ID to be 1, got %d", team.ID)
	}
	if team.Name != "Backend Team" {
		t.Errorf("Expected Name to be 'Backend Team', got %s", team.Name)
	}
	if len(team.Members) != 2 {
		t.Errorf("Expected 2 members, got %d", len(team.Members))
	}
}

func TestUser_Fields(t *testing.T) {
	user := User{
		ID:       "user-1",
		Username: "Alice",
		IsActive: true,
	}

	if user.ID != "user-1" {
		t.Errorf("Expected ID to be 'user-1', got %s", user.ID)
	}
	if user.Username != "Alice" {
		t.Errorf("Expected Username to be 'Alice', got %s", user.Username)
	}
	if !user.IsActive {
		t.Error("Expected IsActive to be true")
	}
}

func TestUser_InactiveUser(t *testing.T) {
	user := User{
		ID:       "user-2",
		Username: "Bob",
		IsActive: false,
	}

	if user.IsActive {
		t.Error("Expected IsActive to be false")
	}
}
