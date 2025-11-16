package mocks

import (
	"context"
	"errors"
	"sync"

	"github.com/sssciel/avito-backend-intership/internals/storage/models"
)

type MockTeamStorage struct {
	mu                          sync.RWMutex
	Teams                       map[string]models.Team
	TeamsByID                   map[int]models.Team
	TeamMembers                 map[int][]string
	AddTeamFunc                 func(ctx context.Context, team models.Team) (models.Team, error)
	GetTeamByNameFunc           func(ctx context.Context, name string) (models.Team, error)
	GetRandomReviewersFunc      func(ctx context.Context, teamID int, authorID string, limit int) ([]string, error)
	GetReplacementCandidateFunc func(ctx context.Context, teamID int, excludeUserIDs []string) (string, error)
}

func NewMockTeamStorage() *MockTeamStorage {
	return &MockTeamStorage{
		Teams:       make(map[string]models.Team),
		TeamsByID:   make(map[int]models.Team),
		TeamMembers: make(map[int][]string),
	}
}

func (m *MockTeamStorage) AddTeam(ctx context.Context, team models.Team) (models.Team, error) {
	if m.AddTeamFunc != nil {
		return m.AddTeamFunc(ctx, team)
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.Teams[team.Name]; exists {
		return models.Team{}, errors.New("TEAM_EXISTS")
	}

	team.ID = len(m.Teams) + 1
	m.Teams[team.Name] = team
	m.TeamsByID[team.ID] = team
	return team, nil
}

func (m *MockTeamStorage) GetTeamByName(ctx context.Context, name string) (models.Team, error) {
	if m.GetTeamByNameFunc != nil {
		return m.GetTeamByNameFunc(ctx, name)
	}
	m.mu.RLock()
	defer m.mu.RUnlock()

	team, exists := m.Teams[name]
	if !exists {
		return models.Team{}, errors.New("NOT_FOUND")
	}
	return team, nil
}

func (m *MockTeamStorage) GetRandomReviewers(ctx context.Context, teamID int, authorID string, limit int) ([]string, error) {
	if m.GetRandomReviewersFunc != nil {
		return m.GetRandomReviewersFunc(ctx, teamID, authorID, limit)
	}
	m.mu.RLock()
	defer m.mu.RUnlock()

	team, exists := m.TeamsByID[teamID]
	if !exists {
		return nil, errors.New("NOT_FOUND")
	}

	var reviewers []string
	for _, member := range team.Members {
		if member.ID != authorID && member.IsActive && len(reviewers) < limit {
			reviewers = append(reviewers, member.ID)
		}
	}
	return reviewers, nil
}

func (m *MockTeamStorage) GetReplacementCandidate(ctx context.Context, teamID int, excludeUserIDs []string) (string, error) {
	if m.GetReplacementCandidateFunc != nil {
		return m.GetReplacementCandidateFunc(ctx, teamID, excludeUserIDs)
	}
	m.mu.RLock()
	defer m.mu.RUnlock()

	team, exists := m.TeamsByID[teamID]
	if !exists {
		return "", errors.New("NOT_FOUND")
	}

	excludeMap := make(map[string]bool)
	for _, id := range excludeUserIDs {
		excludeMap[id] = true
	}

	for _, member := range team.Members {
		if member.IsActive && !excludeMap[member.ID] {
			return member.ID, nil
		}
	}
	return "", errors.New("NO_CANDIDATE")
}

type MockUserStorage struct {
	mu                 sync.RWMutex
	Users              map[string]models.User
	UserReviews        map[string][]models.PullRequest
	UserTeams          map[string]int
	SetIsActiveFunc    func(ctx context.Context, userID string, isActive bool) error
	GetIsActiveFunc    func(ctx context.Context, userID string) (bool, error)
	GetUserReviewsFunc func(ctx context.Context, userID string) ([]models.PullRequest, error)
	GetUserTeamIDFunc  func(ctx context.Context, userID string) (int, error)
}

func NewMockUserStorage() *MockUserStorage {
	return &MockUserStorage{
		Users:       make(map[string]models.User),
		UserReviews: make(map[string][]models.PullRequest),
		UserTeams:   make(map[string]int),
	}
}

func (m *MockUserStorage) SetIsActive(ctx context.Context, userID string, isActive bool) error {
	if m.SetIsActiveFunc != nil {
		return m.SetIsActiveFunc(ctx, userID, isActive)
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	user, exists := m.Users[userID]
	if !exists {
		return errors.New("NOT_FOUND")
	}

	user.IsActive = isActive
	m.Users[userID] = user
	return nil
}

func (m *MockUserStorage) GetIsActive(ctx context.Context, userID string) (bool, error) {
	if m.GetIsActiveFunc != nil {
		return m.GetIsActiveFunc(ctx, userID)
	}
	m.mu.RLock()
	defer m.mu.RUnlock()

	user, exists := m.Users[userID]
	if !exists {
		return false, errors.New("NOT_FOUND")
	}
	return user.IsActive, nil
}

func (m *MockUserStorage) GetUserReviews(ctx context.Context, userID string) ([]models.PullRequest, error) {
	if m.GetUserReviewsFunc != nil {
		return m.GetUserReviewsFunc(ctx, userID)
	}
	m.mu.RLock()
	defer m.mu.RUnlock()

	reviews, exists := m.UserReviews[userID]
	if !exists {
		return []models.PullRequest{}, nil
	}
	return reviews, nil
}

func (m *MockUserStorage) GetUserTeamID(ctx context.Context, userID string) (int, error) {
	if m.GetUserTeamIDFunc != nil {
		return m.GetUserTeamIDFunc(ctx, userID)
	}
	m.mu.RLock()
	defer m.mu.RUnlock()

	teamID, exists := m.UserTeams[userID]
	if !exists {
		return 0, errors.New("NOT_FOUND")
	}
	return teamID, nil
}

type MockRequestStorage struct {
	mu                    sync.RWMutex
	PullRequests          map[string]models.PullRequest
	PRReviewers           map[string][]string
	CreatePullRequestFunc func(ctx context.Context, pr models.PullRequest, reviewerIDs []string) (models.PullRequest, error)
	MergePullRequestFunc  func(ctx context.Context, pullRequestID string) (models.PullRequest, error)
	ReassignReviewerFunc  func(ctx context.Context, pullRequestID string, oldReviewerID string, newReviewerID string) (models.PullRequest, error)
}

func NewMockRequestStorage() *MockRequestStorage {
	return &MockRequestStorage{
		PullRequests: make(map[string]models.PullRequest),
		PRReviewers:  make(map[string][]string),
	}
}

func (m *MockRequestStorage) CreatePullRequest(ctx context.Context, pr models.PullRequest, reviewerIDs []string) (models.PullRequest, error) {
	if m.CreatePullRequestFunc != nil {
		return m.CreatePullRequestFunc(ctx, pr, reviewerIDs)
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.PullRequests[pr.ID]; exists {
		return models.PullRequest{}, errors.New("PR_EXISTS")
	}

	pr.Status = "OPEN"
	pr.AssignedReviewers = reviewerIDs
	m.PullRequests[pr.ID] = pr
	m.PRReviewers[pr.ID] = reviewerIDs
	return pr, nil
}

func (m *MockRequestStorage) MergePullRequest(ctx context.Context, pullRequestID string) (models.PullRequest, error) {
	if m.MergePullRequestFunc != nil {
		return m.MergePullRequestFunc(ctx, pullRequestID)
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	pr, exists := m.PullRequests[pullRequestID]
	if !exists {
		return models.PullRequest{}, errors.New("NOT_FOUND")
	}

	pr.Status = "MERGED"
	m.PullRequests[pullRequestID] = pr
	return pr, nil
}

func (m *MockRequestStorage) ReassignReviewer(ctx context.Context, pullRequestID string, oldReviewerID string, newReviewerID string) (models.PullRequest, error) {
	if m.ReassignReviewerFunc != nil {
		return m.ReassignReviewerFunc(ctx, pullRequestID, oldReviewerID, newReviewerID)
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	pr, exists := m.PullRequests[pullRequestID]
	if !exists {
		return models.PullRequest{}, errors.New("NOT_FOUND")
	}

	if pr.Status == "MERGED" {
		return models.PullRequest{}, errors.New("PR_MERGED")
	}

	found := false
	for i, id := range pr.AssignedReviewers {
		if id == oldReviewerID {
			pr.AssignedReviewers[i] = newReviewerID
			found = true
			break
		}
	}

	if !found {
		return models.PullRequest{}, errors.New("NOT_ASSIGNED")
	}

	m.PullRequests[pullRequestID] = pr
	return pr, nil
}
