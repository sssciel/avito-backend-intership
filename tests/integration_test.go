package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/sssciel/avito-backend-intership/internals/pullrequests"
	"github.com/sssciel/avito-backend-intership/internals/storage/pgsql"
	"github.com/sssciel/avito-backend-intership/internals/teams"
	"github.com/sssciel/avito-backend-intership/internals/users"
)

var testDB *sqlx.DB
var router *gin.Engine

func setupTestDB() *sqlx.DB {
	dbURL := os.Getenv("TEST_DB_URL")
	if dbURL == "" {
		dbURL = "postgresql://admin:admin@localhost:54322/avito_test?sslmode=disable"
	}

	db, err := sqlx.Open("pgx", dbURL)
	if err != nil {
		panic(fmt.Sprintf("Failed to connect to test database: %v", err))
	}

	if err := db.Ping(); err != nil {
		panic(fmt.Sprintf("Failed to ping test database: %v", err))
	}

	return db
}

func cleanupDB(db *sqlx.DB) {
	db.Exec("TRUNCATE TABLE pull_request_reviewers CASCADE")
	db.Exec("TRUNCATE TABLE pull_requests CASCADE")
	db.Exec("TRUNCATE TABLE team_members CASCADE")
	db.Exec("TRUNCATE TABLE users CASCADE")
	db.Exec("TRUNCATE TABLE teams CASCADE")
}

func setupRouter(db *sqlx.DB) *gin.Engine {
	gin.SetMode(gin.TestMode)

	teamStorage := &pgsql.PGTeamStorage{DB: db}
	userStorage := &pgsql.PGUserStorage{DB: db}
	requestStorage := &pgsql.PGPullRequestStorage{DB: db}

	teamService := teams.New(teamStorage, userStorage)
	userService := users.New(userStorage, teamStorage)
	prService := pullrequests.New(requestStorage, teamStorage, userStorage)

	r := gin.Default()
	api := r.Group("/api/v1")

	teamService.RegisterRoutes(api)
	userService.RegisterRoutes(api)
	prService.RegisterRoutes(api)

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	return r
}

func TestMain(m *testing.M) {
	testDB = setupTestDB()
	router = setupRouter(testDB)

	code := m.Run()

	testDB.Close()
	os.Exit(code)
}

func TestIntegration_HealthCheck(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestIntegration_TeamFlow(t *testing.T) {
	cleanupDB(testDB)

	teamData := map[string]interface{}{
		"team_name": "Backend Team",
		"members": []map[string]interface{}{
			{"user_id": "u1", "username": "Alice", "is_active": true},
			{"user_id": "u2", "username": "Bob", "is_active": true},
			{"user_id": "u3", "username": "Charlie", "is_active": true},
		},
	}

	body, _ := json.Marshal(teamData)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/team/add", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Expected status 201, got %d. Body: %s", w.Code, w.Body.String())
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	team := response["team"].(map[string]interface{})
	if team["name"] != "Backend Team" {
		t.Errorf("Expected team name 'Backend Team', got %v", team["name"])
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/team/get?team_name=Backend%20Team", nil)
	w = httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestIntegration_CreatePullRequest(t *testing.T) {
	cleanupDB(testDB)

	teamData := map[string]interface{}{
		"team_name": "DevOps",
		"members": []map[string]interface{}{
			{"user_id": "dev1", "username": "Dave", "is_active": true},
			{"user_id": "dev2", "username": "Emma", "is_active": true},
			{"user_id": "dev3", "username": "Frank", "is_active": true},
		},
	}

	body, _ := json.Marshal(teamData)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/team/add", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Failed to create team: %s", w.Body.String())
	}

	prData := map[string]interface{}{
		"pull_request_id":   "pr-integration-1",
		"pull_request_name": "Add integration tests",
		"author_id":         "dev1",
	}

	body, _ = json.Marshal(prData)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/pullRequest/create", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Expected status 201, got %d. Body: %s", w.Code, w.Body.String())
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	pr := response["pr"].(map[string]interface{})
	if pr["pull_request_id"] != "pr-integration-1" {
		t.Errorf("Expected PR ID 'pr-integration-1', got %v", pr["pull_request_id"])
	}
	if pr["status"] != "OPEN" {
		t.Errorf("Expected status 'OPEN', got %v", pr["status"])
	}

	reviewers := pr["assigned_reviewers"].([]interface{})
	if len(reviewers) > 2 {
		t.Errorf("Expected at most 2 reviewers, got %d", len(reviewers))
	}

	for _, rev := range reviewers {
		if rev == "dev1" {
			t.Error("Author should not be assigned as reviewer")
		}
	}
}

func TestIntegration_MergePullRequest(t *testing.T) {
	cleanupDB(testDB)

	teamData := map[string]interface{}{
		"team_name": "Frontend",
		"members": []map[string]interface{}{
			{"user_id": "fe1", "username": "Grace", "is_active": true},
			{"user_id": "fe2", "username": "Henry", "is_active": true},
		},
	}

	body, _ := json.Marshal(teamData)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/team/add", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	prData := map[string]interface{}{
		"pull_request_id":   "pr-merge-test",
		"pull_request_name": "Fix bug",
		"author_id":         "fe1",
	}

	body, _ = json.Marshal(prData)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/pullRequest/create", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	mergeData := map[string]interface{}{
		"pull_request_id": "pr-merge-test",
	}

	body, _ = json.Marshal(mergeData)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/pullRequest/merge", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	pr := response["pr"].(map[string]interface{})
	if pr["status"] != "MERGED" {
		t.Errorf("Expected status 'MERGED', got %v", pr["status"])
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/pullRequest/merge", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected idempotent merge to return 200, got %d", w.Code)
	}
}

func TestIntegration_SetUserActive(t *testing.T) {
	cleanupDB(testDB)

	teamData := map[string]interface{}{
		"team_name": "QA",
		"members": []map[string]interface{}{
			{"user_id": "qa1", "username": "Ian", "is_active": true},
		},
	}

	body, _ := json.Marshal(teamData)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/team/add", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	userData := map[string]interface{}{
		"user_id":   "qa1",
		"is_active": false,
	}

	body, _ = json.Marshal(userData)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/users/setIsActive", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	user := response["user"].(map[string]interface{})
	if user["is_active"] != false {
		t.Errorf("Expected is_active to be false, got %v", user["is_active"])
	}
}

func TestIntegration_GetUserReviews(t *testing.T) {
	cleanupDB(testDB)

	teamData := map[string]interface{}{
		"team_name": "Platform",
		"members": []map[string]interface{}{
			{"user_id": "plat1", "username": "Jack", "is_active": true},
			{"user_id": "plat2", "username": "Kelly", "is_active": true},
			{"user_id": "plat3", "username": "Leo", "is_active": true},
		},
	}

	body, _ := json.Marshal(teamData)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/team/add", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	prData := map[string]interface{}{
		"pull_request_id":   "pr-review-test",
		"pull_request_name": "Add feature",
		"author_id":         "plat1",
	}

	body, _ = json.Marshal(prData)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/pullRequest/create", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var createResponse map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &createResponse)
	pr := createResponse["pr"].(map[string]interface{})
	reviewers := pr["assigned_reviewers"].([]interface{})

	if len(reviewers) == 0 {
		t.Skip("No reviewers assigned, skipping review test")
	}

	reviewerID := reviewers[0].(string)

	req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/users/getReview?user_id=%s", reviewerID), nil)
	w = httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	prs := response["pull_requests"].([]interface{})
	if len(prs) == 0 {
		t.Error("Expected at least one PR in reviews")
	}
}

func TestIntegration_ReassignReviewer(t *testing.T) {
	cleanupDB(testDB)

	teamData := map[string]interface{}{
		"team_name": "Mobile",
		"members": []map[string]interface{}{
			{"user_id": "mob1", "username": "Mike", "is_active": true},
			{"user_id": "mob2", "username": "Nina", "is_active": true},
			{"user_id": "mob3", "username": "Oscar", "is_active": true},
			{"user_id": "mob4", "username": "Paula", "is_active": true},
		},
	}

	body, _ := json.Marshal(teamData)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/team/add", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	prData := map[string]interface{}{
		"pull_request_id":   "pr-reassign-test",
		"pull_request_name": "Refactor code",
		"author_id":         "mob1",
	}

	body, _ = json.Marshal(prData)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/pullRequest/create", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var createResponse map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &createResponse)
	pr := createResponse["pr"].(map[string]interface{})
	reviewers := pr["assigned_reviewers"].([]interface{})

	if len(reviewers) == 0 {
		t.Skip("No reviewers assigned, skipping reassign test")
	}

	oldReviewerID := reviewers[0].(string)

	reassignData := map[string]interface{}{
		"pull_request_id": "pr-reassign-test",
		"old_reviewer_id": oldReviewerID,
	}

	body, _ = json.Marshal(reassignData)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/pullRequest/reassign", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	replacedBy := response["replaced_by"].(string)
	if replacedBy == "" {
		t.Error("Expected replaced_by to be set")
	}
	if replacedBy == oldReviewerID {
		t.Error("New reviewer should be different from old reviewer")
	}
}

func TestIntegration_CannotReassignMergedPR(t *testing.T) {
	cleanupDB(testDB)

	teamData := map[string]interface{}{
		"team_name": "Security",
		"members": []map[string]interface{}{
			{"user_id": "sec1", "username": "Quinn", "is_active": true},
			{"user_id": "sec2", "username": "Rachel", "is_active": true},
			{"user_id": "sec3", "username": "Steve", "is_active": true},
		},
	}

	body, _ := json.Marshal(teamData)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/team/add", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	prData := map[string]interface{}{
		"pull_request_id":   "pr-merged-reassign",
		"pull_request_name": "Security patch",
		"author_id":         "sec1",
	}

	body, _ = json.Marshal(prData)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/pullRequest/create", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var createResponse map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &createResponse)
	pr := createResponse["pr"].(map[string]interface{})
	reviewers := pr["assigned_reviewers"].([]interface{})

	mergeData := map[string]interface{}{
		"pull_request_id": "pr-merged-reassign",
	}

	body, _ = json.Marshal(mergeData)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/pullRequest/merge", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if len(reviewers) > 0 {
		oldReviewerID := reviewers[0].(string)

		reassignData := map[string]interface{}{
			"pull_request_id": "pr-merged-reassign",
			"old_reviewer_id": oldReviewerID,
		}

		body, _ = json.Marshal(reassignData)
		req = httptest.NewRequest(http.MethodPost, "/api/v1/pullRequest/reassign", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusConflict {
			t.Errorf("Expected status 409 for reassigning merged PR, got %d", w.Code)
		}
	}
}

func TestIntegration_InactiveUserNotAssigned(t *testing.T) {
	cleanupDB(testDB)

	teamData := map[string]interface{}{
		"team_name": "DataScience",
		"members": []map[string]interface{}{
			{"user_id": "ds1", "username": "Tom", "is_active": true},
			{"user_id": "ds2", "username": "Uma", "is_active": false},
			{"user_id": "ds3", "username": "Vince", "is_active": true},
		},
	}

	body, _ := json.Marshal(teamData)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/team/add", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	prData := map[string]interface{}{
		"pull_request_id":   "pr-inactive-test",
		"pull_request_name": "ML model",
		"author_id":         "ds1",
	}

	body, _ = json.Marshal(prData)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/pullRequest/create", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	pr := response["pr"].(map[string]interface{})
	reviewers := pr["assigned_reviewers"].([]interface{})

	for _, rev := range reviewers {
		if rev == "ds2" {
			t.Error("Inactive user should not be assigned as reviewer")
		}
	}
}
