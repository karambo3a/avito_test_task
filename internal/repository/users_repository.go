package repository

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"github.com/karambo3a/avito_test_task/internal/model"
)

type UsersPostgresRepository struct {
	db *sql.DB
}

func NewUsersPostgresRepository(db *sql.DB) *UsersPostgresRepository {
	return &UsersPostgresRepository{db: db}
}

func (r *UsersPostgresRepository) SetUserIsActive(ctx context.Context, userID string, isActive bool) (*model.User, error) {
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

	result, err := tx.ExecContext(ctx, `
		UPDATE users
		SET is_active = $1
		WHERE user_id = $2
		`, isActive, userID)

	if err != nil {
		log.Printf("exec error: %v", err)
		return nil, fmt.Errorf("exec error: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("RowsAffected error: %v", err)
		return nil, fmt.Errorf("RowsAffected error: %w", err)
	}
	if rowsAffected == 0 {
		log.Printf("user not found: %v", userID)
		return nil, model.NewNotFoundError()
	}

	row := tx.QueryRowContext(ctx, `
		SELECT user_id, username, team_name, is_active
		FROM users
		WHERE user_id = $1
		`, userID)

	var user model.User
	if err := row.Scan(&user.UserID, &user.Username, &user.TeamName, &user.IsActive); err != nil {
		log.Printf("scan error: %v", err)
		return nil, fmt.Errorf("scan error: %w", err)
	}

	if err := tx.Commit(); err != nil {
		log.Printf("commit transaction error: %v", err)
		return nil, fmt.Errorf("commit transaction error: %w", err)
	}
	return &user, nil
}

func (r *UsersPostgresRepository) GetUserReview(ctx context.Context, userID string) ([]model.PullRequestShort, error) {
	log.Println(userID)
	ok, err := r.UserExists(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !ok {
		log.Printf("user not found: %v", userID)
		return nil, model.NewNotFoundError()
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT pr.pr_id, pr.pr_name, pr.author_id, pr.status
		FROM pr INNER JOIN reviewer_x_pr as rpr ON pr.pr_id = rpr.pr_id
		WHERE rpr.user_id = $1
		`, userID)
	if err != nil {
		log.Printf("query error: %v", err)
		return nil, fmt.Errorf("query error: %w", err)
	}
	defer rows.Close()

	prs := []model.PullRequestShort{}
	for rows.Next() {
		var pr model.PullRequestShort
		err := rows.Scan(&pr.PullRequestID, &pr.PullRequestName, &pr.AuthorID, &pr.Status)
		if err != nil {
			log.Printf("query error: %v", err)
			return nil, fmt.Errorf("query error: %w", err)
		}

		prs = append(prs, pr)
	}

	return prs, nil
}

func (r *UsersPostgresRepository) UserExists(ctx context.Context, userID string) (bool, error) {
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
