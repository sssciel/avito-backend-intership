package users

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/sssciel/avito-backend-intership/internals/storage/mocks"
	"github.com/sssciel/avito-backend-intership/internals/storage/models"
)

func setupRouter(service *UserService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	api := r.Group("/api/v1")
	service.RegisterRoutes(api)
	return r
}

func TestSetIsActive_Success(t *testing.T) {
	userStorage := mocks.NewMockUserStorage()
	teamStorage := mocks.NewMockTeamStorage()
	service := New(userStorage, teamStorage)
	router := setupRouter(service)

	userStorage.Users["u1"] = models.User{
		ID:       "u1",
		Username: "Alice",
		IsActive: true,
	}

	reqBody := SetIsActiveRequest{
		UserID:   "u1",
		IsActive: false,
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/setIsActive", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if userStorage.Users["u1"].IsActive {
		t.Error("Expected user to be inactive")
	}
}

func TestSetIsActive_UserNotFound(t *testing.T) {
	userStorage := mocks.NewMockUserStorage()
	teamStorage := mocks.NewMockTeamStorage()
	service := New(userStorage, teamStorage)
	router := setupRouter(service)

	reqBody := SetIsActiveRequest{
		UserID:   "nonexistent",
		IsActive: false,
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/setIsActive", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestGetUserReviews_Success(t *testing.T) {
	userStorage := mocks.NewMockUserStorage()
	teamStorage := mocks.NewMockTeamStorage()
	service := New(userStorage, teamStorage)
	router := setupRouter(service)

	userStorage.UserReviews["u1"] = []models.PullRequest{
		{
			ID:                "pr-1",
			Name:              "Test PR",
			AuthorID:          "u2",
			Status:            "OPEN",
			AssignedReviewers: []string{"u1"},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/getReview?user_id=u1", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response GetReviewsResponse
	json.Unmarshal(w.Body.Bytes(), &response)

	if response.UserID != "u1" {
		t.Errorf("Expected user_id 'u1', got %s", response.UserID)
	}
	if len(response.PullRequests) != 1 {
		t.Errorf("Expected 1 PR, got %d", len(response.PullRequests))
	}
}

func TestGetUserReviews_EmptyList(t *testing.T) {
	userStorage := mocks.NewMockUserStorage()
	teamStorage := mocks.NewMockTeamStorage()
	service := New(userStorage, teamStorage)
	router := setupRouter(service)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/getReview?user_id=u1", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response GetReviewsResponse
	json.Unmarshal(w.Body.Bytes(), &response)

	if len(response.PullRequests) != 0 {
		t.Errorf("Expected 0 PRs, got %d", len(response.PullRequests))
	}
}

func TestGetUserReviews_MissingParameter(t *testing.T) {
	userStorage := mocks.NewMockUserStorage()
	teamStorage := mocks.NewMockTeamStorage()
	service := New(userStorage, teamStorage)
	router := setupRouter(service)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/getReview", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}
