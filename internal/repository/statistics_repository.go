package repository

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"github.com/karambo3a/avito_test_task/internal/model"
)

type StatisticsPostgresRepository struct {
	db *sql.DB
}

func NewStatisticsPostgresRepository(db *sql.DB) *StatisticsPostgresRepository {
	return &StatisticsPostgresRepository{db: db}
}

func (r *StatisticsPostgresRepository) GetUserStatistics(ctx context.Context, userID string) (*model.UserStatistics, error) {
	ok, err := r.UserExists(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, model.NewNotFoundError()
	}

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

	var stats model.UserStatistics
	err = tx.QueryRowContext(ctx, `
		SELECT user_id, username, team_name
		FROM users
		WHERE user_id = $1
	`, userID).Scan(&stats.UserID, &stats.Username, &stats.TeamName)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	err = tx.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM pr
		WHERE author_id = $1
	`, userID).Scan(&stats.AuthoredPRsCount)
	if err != nil {
		log.Printf("query row error: %v", err)
		return nil, fmt.Errorf("query row error: %w", err)
	}

	err = tx.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM reviewer_x_pr
		WHERE user_id = $1
	`, userID).Scan(&stats.AssignedReviewsCount)
	if err != nil {
		log.Printf("query row error: %v", err)
		return nil, fmt.Errorf("query row error: %w", err)
	}

	if err = tx.Commit(); err != nil {
		log.Printf("commit transaction error: %v", err)
		return nil, fmt.Errorf("commit transaction error: %w", err)
	}

	return &stats, nil
}

func (r *StatisticsPostgresRepository) GetTeamStatistics(ctx context.Context, teamName string) (*model.TeamStatistics, error) {
	ok, err := r.TeamExists(ctx, teamName)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, model.NewNotFoundError()
	}

	stats := model.TeamStatistics{TeamName: teamName}

	err = r.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*) as total_prs,
			SUM(CASE WHEN status = 'MERGED' THEN 1 ELSE 0 END) as merged_prs,
			SUM(CASE WHEN status = 'OPEN' THEN 1 ELSE 0 END) as open_prs
		FROM pr
		JOIN users as u ON pr.author_id = u.user_id
		WHERE u.team_name = $1
	`, teamName).Scan(&stats.TotalPRs, &stats.MergedPRs, &stats.OpenPRs)
	if err != nil {
		log.Printf("scan error: %v", err)
		return nil, fmt.Errorf("scan error: %w", err)
	}

	return &stats, nil
}

func (r *StatisticsPostgresRepository) UserExists(ctx context.Context, userID string) (bool, error) {
	result := r.db.QueryRowContext(ctx, `
		SELECT user_id
		FROM users
		WHERE user_id = $1`,
		userID)

	log.Printf("userID: %s", userID)
	var id string
	if err := result.Scan(&id); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		log.Printf("scan error: %v", err)
		return false, fmt.Errorf("scan error: %w", err)
	}
	log.Printf("id: %s", id)

	return true, nil
}

func (r *StatisticsPostgresRepository) TeamExists(ctx context.Context, teamName string) (bool, error) {
	result := r.db.QueryRowContext(ctx, `
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
