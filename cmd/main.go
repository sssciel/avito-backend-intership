package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/sssciel/avito-backend-intership/internals/pullrequests"
	"github.com/sssciel/avito-backend-intership/internals/storage/pgsql"
	"github.com/sssciel/avito-backend-intership/internals/teams"
	"github.com/sssciel/avito-backend-intership/internals/users"
	"github.com/sssciel/avito-backend-intership/pkg/config"
)

var serviceLogger *slog.Logger

func setupLogger() {
	opts := &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}
	logJsonHandler := slog.NewJSONHandler(os.Stdout, opts)
	serviceLogger = slog.New(logJsonHandler)
	slog.SetDefault(serviceLogger)
}

func init() {
	setupLogger()
	slog.Debug("Logger initialized")

	config.SetupConfigs()
	slog.Debug("Config initialized")
}

func main() {
	slog.Debug("Starting server")

	ctx := context.Background()

	db := pgsql.CreatePGConnection(ctx)
	slog.Debug("DB pool created successfully")
	defer db.Close()

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

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	slog.Info("Server started", "port", port)
	if err := r.Run(":" + port); err != nil {
		slog.Error("Failed to start server", "err", err)
		os.Exit(1)
	}
}
