package pgsql

import (
	"context"
	"github.com/jmoiron/sqlx"
	"github.com/sssciel/avito-backend-intership/pkg/config"
	"log/slog"
	"os"
)

func CreatePGConnection(ctx context.Context) *sqlx.DB {
	slog.Debug("Creating Postgres pool connection")
	db, err := sqlx.Open("pgx", config.GetDBURL("postgresql"))
	if err != nil {
		slog.Error("SQL create connection error", "err", err)
		os.Exit(1)
	}

	slog.Debug("Try to ping Postgres")
	if err := db.PingContext(ctx); err != nil {
		slog.Error("SQL ping error", "err", err)
		os.Exit(1)
	}
	slog.Debug("Ping is successful")

	return db
}
