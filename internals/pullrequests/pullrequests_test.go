package pullrequests

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/sssciel/avito-backend-intership/internals/storage/mocks"
	"github.com/sssciel/avito-backend-intership/internals/storage/models"
)

func setupRouter(service *PullRequestService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	api := r.Group("/api/v1")
	service.RegisterRoutes(api)
	return r
}

func TestCreatePullRequest_Success(t *testing.T) {
	requestStorage := mocks.NewMockRequestStorage()
	teamStorage := mocks.NewMockTeamStorage()
	userStorage := mocks.NewMockUserStorage()

	team := models.Team{
		ID:   1,
		Name: "Backend",
		Members: []models.User{
			{ID: "u1", Username: "Alice", IsActive: true},
			{ID: "u2", Username: "Bob", IsActive: true},
			{ID: "u3", Username: "Charlie", IsActive: true},
		},
	}
	teamStorage.TeamsByID[1] = team

	teamStorage.GetRandomReviewersFunc = func(ctx context.Context, teamID int, authorID string, limit int) ([]string, error) {
		return []string{"u2", "u3"}, nil
	}

	service := New(requestStorage, teamStorage, userStorage)
	service.GetAuthorTeamIDFn = func(ctx context.Context, userID string) (int, error) {
		return 1, nil
	}

	router := setupRouter(service)

	reqBody := CreatePRRequest{
		PullRequestID:   "pr-1",
		PullRequestName: "Add feature",
		AuthorID:        "u1",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/pullRequest/create", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d. Body: %s", w.Code, w.Body.String())
	}

	var response PRResponse
	json.Unmarshal(w.Body.Bytes(), &response)

	if response.PR.ID != "pr-1" {
		t.Errorf("Expected PR ID 'pr-1', got %s", response.PR.ID)
	}
	if response.PR.Status != "OPEN" {
		t.Errorf("Expected status 'OPEN', got %s", response.PR.Status)
	}
	if len(response.PR.AssignedReviewers) != 2 {
		t.Errorf("Expected 2 reviewers, got %d", len(response.PR.AssignedReviewers))
	}
}

func TestCreatePullRequest_AlreadyExists(t *testing.T) {
	requestStorage := mocks.NewMockRequestStorage()
	teamStorage := mocks.NewMockTeamStorage()
	userStorage := mocks.NewMockUserStorage()

	requestStorage.PullRequests["pr-1"] = models.PullRequest{
		ID:       "pr-1",
		Name:     "Existing PR",
		AuthorID: "u1",
		Status:   "OPEN",
	}

	teamStorage.GetRandomReviewersFunc = func(ctx context.Context, teamID int, authorID string, limit int) ([]string, error) {
		return []string{"u2"}, nil
	}

	service := New(requestStorage, teamStorage, userStorage)
	service.GetAuthorTeamIDFn = func(ctx context.Context, userID string) (int, error) {
		return 1, nil
	}

	router := setupRouter(service)

	reqBody := CreatePRRequest{
		PullRequestID:   "pr-1",
		PullRequestName: "Add feature",
		AuthorID:        "u1",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/pullRequest/create", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("Expected status 409, got %d", w.Code)
	}
}

func TestMergePullRequest_Success(t *testing.T) {
	requestStorage := mocks.NewMockRequestStorage()
	teamStorage := mocks.NewMockTeamStorage()
	userStorage := mocks.NewMockUserStorage()

	requestStorage.PullRequests["pr-1"] = models.PullRequest{
		ID:                "pr-1",
		Name:              "Test PR",
		AuthorID:          "u1",
		Status:            "OPEN",
		AssignedReviewers: []string{"u2"},
	}

	service := New(requestStorage, teamStorage, userStorage)
	router := setupRouter(service)

	reqBody := MergePRRequest{
		PullRequestID: "pr-1",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/pullRequest/merge", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response PRResponse
	json.Unmarshal(w.Body.Bytes(), &response)

	if response.PR.Status != "MERGED" {
		t.Errorf("Expected status 'MERGED', got %s", response.PR.Status)
	}
}

func TestMergePullRequest_NotFound(t *testing.T) {
	requestStorage := mocks.NewMockRequestStorage()
	teamStorage := mocks.NewMockTeamStorage()
	userStorage := mocks.NewMockUserStorage()

	service := New(requestStorage, teamStorage, userStorage)
	router := setupRouter(service)

	reqBody := MergePRRequest{
		PullRequestID: "nonexistent",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/pullRequest/merge", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestMergePullRequest_Idempotent(t *testing.T) {
	requestStorage := mocks.NewMockRequestStorage()
	teamStorage := mocks.NewMockTeamStorage()
	userStorage := mocks.NewMockUserStorage()

	requestStorage.PullRequests["pr-1"] = models.PullRequest{
		ID:                "pr-1",
		Name:              "Test PR",
		AuthorID:          "u1",
		Status:            "MERGED",
		AssignedReviewers: []string{"u2"},
	}

	service := New(requestStorage, teamStorage, userStorage)
	router := setupRouter(service)

	reqBody := MergePRRequest{
		PullRequestID: "pr-1",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/pullRequest/merge", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestReassignReviewer_Success(t *testing.T) {
	requestStorage := mocks.NewMockRequestStorage()
	teamStorage := mocks.NewMockTeamStorage()
	userStorage := mocks.NewMockUserStorage()

	requestStorage.PullRequests["pr-1"] = models.PullRequest{
		ID:                "pr-1",
		Name:              "Test PR",
		AuthorID:          "u1",
		Status:            "OPEN",
		AssignedReviewers: []string{"u2"},
	}

	team := models.Team{
		ID:   1,
		Name: "Backend",
		Members: []models.User{
			{ID: "u2", Username: "Bob", IsActive: true},
			{ID: "u3", Username: "Charlie", IsActive: true},
		},
	}
	teamStorage.TeamsByID[1] = team

	service := New(requestStorage, teamStorage, userStorage)
	service.GetAuthorTeamIDFn = func(ctx context.Context, userID string) (int, error) {
		return 1, nil
	}

	router := setupRouter(service)

	reqBody := ReassignRequest{
		PullRequestID: "pr-1",
		OldReviewerID: "u2",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/pullRequest/reassign", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var response ReassignResponse
	json.Unmarshal(w.Body.Bytes(), &response)

	if response.ReplacedBy == "" {
		t.Error("Expected ReplacedBy to be set")
	}
}

func TestReassignReviewer_PRMerged(t *testing.T) {
	requestStorage := mocks.NewMockRequestStorage()
	teamStorage := mocks.NewMockTeamStorage()
	userStorage := mocks.NewMockUserStorage()

	requestStorage.PullRequests["pr-1"] = models.PullRequest{
		ID:                "pr-1",
		Name:              "Test PR",
		AuthorID:          "u1",
		Status:            "MERGED",
		AssignedReviewers: []string{"u2"},
	}

	teamStorage.GetReplacementCandidateFunc = func(ctx context.Context, teamID int, excludeUserIDs []string) (string, error) {
		return "u3", nil
	}

	service := New(requestStorage, teamStorage, userStorage)
	service.GetAuthorTeamIDFn = func(ctx context.Context, userID string) (int, error) {
		return 1, nil
	}

	router := setupRouter(service)

	reqBody := ReassignRequest{
		PullRequestID: "pr-1",
		OldReviewerID: "u2",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/pullRequest/reassign", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("Expected status 409, got %d", w.Code)
	}
}

func TestReassignReviewer_NotAssigned(t *testing.T) {
	requestStorage := mocks.NewMockRequestStorage()
	teamStorage := mocks.NewMockTeamStorage()
	userStorage := mocks.NewMockUserStorage()

	requestStorage.PullRequests["pr-1"] = models.PullRequest{
		ID:                "pr-1",
		Name:              "Test PR",
		AuthorID:          "u1",
		Status:            "OPEN",
		AssignedReviewers: []string{"u2"},
	}

	teamStorage.GetReplacementCandidateFunc = func(ctx context.Context, teamID int, excludeUserIDs []string) (string, error) {
		return "u3", nil
	}

	service := New(requestStorage, teamStorage, userStorage)
	service.GetAuthorTeamIDFn = func(ctx context.Context, userID string) (int, error) {
		return 1, nil
	}

	router := setupRouter(service)

	reqBody := ReassignRequest{
		PullRequestID: "pr-1",
		OldReviewerID: "u3",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/pullRequest/reassign", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("Expected status 409, got %d", w.Code)
	}
}

func TestReassignReviewer_NoCandidate(t *testing.T) {
	requestStorage := mocks.NewMockRequestStorage()
	teamStorage := mocks.NewMockTeamStorage()
	userStorage := mocks.NewMockUserStorage()

	requestStorage.PullRequests["pr-1"] = models.PullRequest{
		ID:                "pr-1",
		Name:              "Test PR",
		AuthorID:          "u1",
		Status:            "OPEN",
		AssignedReviewers: []string{"u2"},
	}

	teamStorage.GetReplacementCandidateFunc = func(ctx context.Context, teamID int, excludeUserIDs []string) (string, error) {
		return "", errors.New("NO_CANDIDATE")
	}

	service := New(requestStorage, teamStorage, userStorage)
	service.GetAuthorTeamIDFn = func(ctx context.Context, userID string) (int, error) {
		return 1, nil
	}

	router := setupRouter(service)

	reqBody := ReassignRequest{
		PullRequestID: "pr-1",
		OldReviewerID: "u2",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/pullRequest/reassign", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("Expected status 409, got %d", w.Code)
	}
}
