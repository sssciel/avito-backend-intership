package teams

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

func setupRouter(service *TeamService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	api := r.Group("/api/v1")
	service.RegisterRoutes(api)
	return r
}

func TestAddTeam_Success(t *testing.T) {
	teamStorage := mocks.NewMockTeamStorage()
	userStorage := mocks.NewMockUserStorage()
	service := New(teamStorage, userStorage)
	router := setupRouter(service)

	reqBody := AddTeamRequest{
		TeamName: "Backend",
		Members: []models.User{
			{ID: "u1", Username: "Alice", IsActive: true},
			{ID: "u2", Username: "Bob", IsActive: true},
		},
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/team/add", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", w.Code)
	}

	var response TeamResponse
	json.Unmarshal(w.Body.Bytes(), &response)

	if response.Team.Name != "Backend" {
		t.Errorf("Expected team name 'Backend', got %s", response.Team.Name)
	}
	if len(response.Team.Members) != 2 {
		t.Errorf("Expected 2 members, got %d", len(response.Team.Members))
	}
}

func TestAddTeam_AlreadyExists(t *testing.T) {
	teamStorage := mocks.NewMockTeamStorage()
	userStorage := mocks.NewMockUserStorage()
	service := New(teamStorage, userStorage)
	router := setupRouter(service)

	team := models.Team{
		Name: "Backend",
		Members: []models.User{
			{ID: "u1", Username: "Alice", IsActive: true},
		},
	}
	teamStorage.Teams["Backend"] = team

	reqBody := AddTeamRequest{
		TeamName: "Backend",
		Members: []models.User{
			{ID: "u2", Username: "Bob", IsActive: true},
		},
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/team/add", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestGetTeam_Success(t *testing.T) {
	teamStorage := mocks.NewMockTeamStorage()
	userStorage := mocks.NewMockUserStorage()
	service := New(teamStorage, userStorage)
	router := setupRouter(service)

	team := models.Team{
		ID:   1,
		Name: "Backend",
		Members: []models.User{
			{ID: "u1", Username: "Alice", IsActive: true},
			{ID: "u2", Username: "Bob", IsActive: true},
		},
	}
	teamStorage.Teams["Backend"] = team

	req := httptest.NewRequest(http.MethodGet, "/api/v1/team/get?team_name=Backend", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response TeamResponse
	json.Unmarshal(w.Body.Bytes(), &response)

	if response.Team.Name != "Backend" {
		t.Errorf("Expected team name 'Backend', got %s", response.Team.Name)
	}
}

func TestGetTeam_NotFound(t *testing.T) {
	teamStorage := mocks.NewMockTeamStorage()
	userStorage := mocks.NewMockUserStorage()
	service := New(teamStorage, userStorage)
	router := setupRouter(service)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/team/get?team_name=NonExistent", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestGetTeam_MissingParameter(t *testing.T) {
	teamStorage := mocks.NewMockTeamStorage()
	userStorage := mocks.NewMockUserStorage()
	service := New(teamStorage, userStorage)
	router := setupRouter(service)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/team/get", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}
