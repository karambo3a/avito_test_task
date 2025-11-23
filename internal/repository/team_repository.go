package repository

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"github.com/karambo3a/avito_test_task/internal/model"
)

type TeamPostgresRepository struct {
	db *sql.DB
}

func NewTeamPostgresRepository(db *sql.DB) *TeamPostgresRepository {
	return &TeamPostgresRepository{db: db}
}

func (r *TeamPostgresRepository) TeamExists(ctx context.Context, tx *sql.Tx, teamName string) (bool, error) {
	result := tx.QueryRowContext(ctx, `
		SELECT team_name
		FROM team
		WHERE team_name = $1`,
		teamName)

	var name string
	if err := result.Scan(&name); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		log.Printf("scan error: %v", err)
		return false, fmt.Errorf("scan error: %w", err)
	}

	return true, nil
}

func (r *TeamPostgresRepository) AddTeam(ctx context.Context, team model.Team) (*model.Team, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("begin transaction error: %v", err)
		return nil, fmt.Errorf("begin transaction error: %w", err)
	}
	defer func() {
		if err = tx.Rollback(); err != nil && err != sql.ErrTxDone {
			log.Printf("rollback transaction error: %v", err)
		}
	}()

	ok, err := r.TeamExists(ctx, tx, team.TeamName)
	if err != nil {
		return nil, err
	}
	if ok {
		log.Printf("team exists: %v", team.TeamName)
		return nil, model.NewTeamExistsError()
	}

	_, err = tx.Exec(`
			INSERT INTO team
			(team_name)
			VALUES ($1)
			`, team.TeamName)
	if err != nil {
		log.Printf("exec error: %v", err)
		return nil, fmt.Errorf("exec error: %w", err)
	}

	for _, teamMember := range team.Members {
		_, err := tx.Exec(`
			INSERT INTO users
			(user_id, username, team_name, is_active)
			VALUES ($1, $2, $3, $4)
			`, teamMember.UserID, teamMember.Username, team.TeamName, teamMember.IsActive)
		if err != nil {
			log.Printf("exec error: %v", err)
			return nil, fmt.Errorf("exec error: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		log.Printf("commit transaction error: %v", err)
		return nil, fmt.Errorf("commit transaction error: %w", err)
	}
	return &team, nil
}

func (r *TeamPostgresRepository) GetTeam(ctx context.Context, teamName string) (*model.Team, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("begin transaction error: %v", err)
		return nil, fmt.Errorf("begin transaction error: %w", err)
	}
	defer func() {
		if err = tx.Rollback(); err != nil && err != sql.ErrTxDone {
			log.Printf("rollback transaction error: %v", err)
		}
	}()

	ok, err := r.TeamExists(ctx, tx, teamName)
	if err != nil {
		return nil, err
	}
	if !ok {
		log.Printf("team doesn't exist: %s", teamName)
		return nil, model.NewNotFoundError()
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT user_id, username, is_active
		FROM users
		WHERE team_name = $1
		`, teamName)
	if err != nil {
		log.Printf("query error: %v", err)
		return nil, fmt.Errorf("query error: %w", err)
	}
	defer rows.Close()

	members := []model.TeamMember{}
	for rows.Next() {
		var member model.TeamMember
		err := rows.Scan(&member.UserID, &member.Username, &member.IsActive)
		if err != nil {
			log.Printf("scan error: %v", err)
			return nil, fmt.Errorf("scan error: %w", err)
		}
		members = append(members, member)
	}

	if err := tx.Commit(); err != nil {
		log.Printf("commit transaction error: %v", err)
		return nil, fmt.Errorf("commit transaction error: %w", err)
	}

	return &model.Team{
		TeamName: teamName,
		Members:  members,
	}, nil
}
