package pgsql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"github.com/jmoiron/sqlx"
	"github.com/sssciel/avito-backend-intership/internals/storage/models"
)

type PGTeamStorage struct {
	DB *sqlx.DB
}

func (p *PGTeamStorage) AddTeam(ctx context.Context, team models.Team) (models.Team, error) {
	slog.Debug("Adding team in PG", "teamName", team.Name)

	tx, err := p.DB.BeginTxx(ctx, nil)
	if err != nil {
		return models.Team{}, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	var exists bool
	err = tx.GetContext(ctx, &exists, "SELECT EXISTS(SELECT 1 FROM teams WHERE name = $1)", team.Name)
	if err != nil {
		return models.Team{}, fmt.Errorf("check team exists: %w", err)
	}
	if exists {
		return models.Team{}, errors.New("TEAM_EXISTS")
	}

	var teamID int
	err = tx.QueryRowContext(ctx, "INSERT INTO teams (name) VALUES ($1) RETURNING id", team.Name).Scan(&teamID)
	if err != nil {
		return models.Team{}, fmt.Errorf("insert team: %w", err)
	}

	for _, member := range team.Members {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO users (user_id, username, is_active)
			VALUES ($1, $2, $3)
			ON CONFLICT (user_id) DO UPDATE SET username = EXCLUDED.username, is_active = EXCLUDED.is_active
		`, member.ID, member.Username, member.IsActive)
		if err != nil {
			return models.Team{}, fmt.Errorf("upsert user %s: %w", member.ID, err)
		}

		_, err = tx.ExecContext(ctx, `
			INSERT INTO team_members (team_id, user_id)
			VALUES ($1, $2)
			ON CONFLICT DO NOTHING
		`, teamID, member.ID)
		if err != nil {
			return models.Team{}, fmt.Errorf("add team member %s: %w", member.ID, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return models.Team{}, fmt.Errorf("commit transaction: %w", err)
	}

	team.ID = teamID
	return team, nil
}

func (p *PGTeamStorage) GetTeamByName(ctx context.Context, name string) (models.Team, error) {
	slog.Debug("Getting team by name in PG", "name", name)

	var team models.Team
	err := p.DB.GetContext(ctx, &team, "SELECT id, name FROM teams WHERE name = $1", name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.Team{}, errors.New("NOT_FOUND")
		}
		return models.Team{}, fmt.Errorf("get team: %w", err)
	}

	err = p.DB.SelectContext(ctx, &team.Members, `
		SELECT u.user_id, u.username, u.is_active
		FROM users u
		INNER JOIN team_members tm ON u.user_id = tm.user_id
		WHERE tm.team_id = $1
	`, team.ID)
	if err != nil {
		return models.Team{}, fmt.Errorf("get team members: %w", err)
	}

	return team, nil
}

func (p *PGTeamStorage) GetRandomReviewers(ctx context.Context, teamID int, authorID string, limit int) ([]string, error) {
	slog.Debug("Getting random reviewers in PG", "teamID", teamID, "authorID", authorID, "limit", limit)

	var reviewers []string
	err := p.DB.SelectContext(ctx, &reviewers, `
  SELECT u.user_id
  FROM users u
  INNER JOIN team_members tm ON u.user_id = tm.user_id
  WHERE tm.team_id = $1
   AND u.is_active = true
   AND u.user_id != $2
  ORDER BY RANDOM()
  LIMIT $3
 `, teamID, authorID, limit)
	if err != nil {
		slog.Error("SQL get random reviewers error", "err", err)
		return nil, fmt.Errorf("failed to get random reviewers: %w", err)
	}

	return reviewers, nil
}

func (p *PGTeamStorage) GetReplacementCandidate(ctx context.Context, teamID int, excludeUserIDs []string) (string, error) {
	slog.Debug("Getting replacement candidate in PG", "teamID", teamID, "excludeIDs", excludeUserIDs)

	query := `
  SELECT u.user_id
  FROM users u
  INNER JOIN team_members tm ON u.user_id = tm.user_id
  WHERE tm.team_id = $1
   AND u.is_active = true
   AND u.user_id != ALL($2)
  ORDER BY RANDOM()
  LIMIT 1
 `

	var candidateID string
	err := p.DB.GetContext(ctx, &candidateID, query, teamID, excludeUserIDs)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", errors.New("NO_CANDIDATE")
		}
		return "", fmt.Errorf("failed to get replacement candidate: %w", err)
	}

	return candidateID, nil
}
