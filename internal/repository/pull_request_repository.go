package repository

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"github.com/karambo3a/avito_test_task/internal/model"
)

type PRPostgresRepository struct {
	db *sql.DB
}

func NewPRPostgresRepository(db *sql.DB) *PRPostgresRepository {
	return &PRPostgresRepository{db: db}
}

func (r *PRPostgresRepository) CreatePR(ctx context.Context, pullRequestID, pullRequestName, authorID string) (*model.PullRequest, error) {
	ok, err := r.AuthorExists(ctx, authorID)
	if err != nil {
		return nil, err
	}
	if !ok {
		log.Printf("author doesn't exist: %s", authorID)
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

	result := r.db.QueryRowContext(ctx, `
		SELECT pr_id
		FROM pr
		WHERE pr_id = $1`,
		pullRequestID)
	var id string
	if err = result.Scan(&id); err != nil {
		if err == sql.ErrNoRows {
			log.Printf("pr exists: %s", pullRequestID)
			return nil, model.NewPRExistsError()
		}
		log.Printf("scan error: %v", err)
		return nil, fmt.Errorf("scan error: %w", err)
	}

	reviewers, err := r.FindReviewers(ctx, tx, authorID)
	if err != nil {
		return nil, err
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO pr (pr_id, pr_name, author_id)
		VALUES ($1, $2, $3)
		`, pullRequestID, pullRequestName, authorID)
	if err != nil {
		log.Printf("exec error: %v", err)
		return nil, fmt.Errorf("exec error: %w", err)
	}

	for _, reviewer := range reviewers {
		_, err = tx.ExecContext(ctx, `
		INSERT INTO reviewer_x_pr (user_id, pr_id)
		VALUES ($1, $2)
		`, reviewer, pullRequestID)
		if err != nil {
			log.Printf("exec error: %v", err)
			return nil, fmt.Errorf("exec error: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		log.Printf("commit transaction error: %v", err)
		return nil, fmt.Errorf("commit transaction error: %w", err)
	}

	return &model.PullRequest{
		PullRequestShort: model.PullRequestShort{
			PullRequestID:   pullRequestID,
			PullRequestName: pullRequestName,
			AuthorID:        authorID,
			Status:          "OPEN",
		},
		AssignedReviewers: reviewers,
	}, nil
}

func (r *PRPostgresRepository) MergePR(ctx context.Context, pullRequestID string) (*model.PullRequest, error) {
	ok, err := r.PRExists(ctx, pullRequestID)
	if err != nil {
		return nil, err
	}
	if !ok {
		log.Printf("pr doesn't exist: %s", pullRequestID)
		return nil, model.NewNotFoundError()
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("begin transaction error: %v", err)
		return nil, fmt.Errorf("begin transaction error: %w", err)
	}
	defer func() {
		if err = tx.Rollback(); err != nil && err != sql.ErrTxDone {
			log.Printf("rollback transaction error in MergePR: %v", err)
		}
	}()

	status, err := r.GetStatus(ctx, tx, pullRequestID)
	if err != nil {
		return nil, err
	}
	if status == "MERGED" {
		var pr *model.PullRequest
		pr, err = r.GetPR(ctx, tx, pullRequestID)
		if err != nil {
			return nil, err
		}
		if err = tx.Commit(); err != nil {
			log.Printf("commit transaction error: %v", err)
			return nil, fmt.Errorf("commit transaction error: %w", err)
		}
		return pr, nil
	}

	row := tx.QueryRowContext(ctx, `
	UPDATE pr
	SET status = 'MERGED', merged_at = CURRENT_TIMESTAMP
	WHERE pr_id = $1
	RETURNING pr_id, pr_name, author_id, status, merged_at
	`, pullRequestID)

	var pr model.PullRequest
	if err = row.Scan(&pr.PullRequestID, &pr.PullRequestName, &pr.AuthorID, &pr.Status, &pr.MergedAt); err != nil {
		log.Printf("pr doesn't exist: %s", pullRequestID)
		return nil, model.NewNotFoundError()
	}

	reviewers, err := r.GetReviewers(ctx, tx, pullRequestID)
	if err != nil {
		return nil, err
	}

	pr.AssignedReviewers = reviewers

	if err := tx.Commit(); err != nil {
		log.Printf("commit transaction error: %v", err)
		return nil, fmt.Errorf("commit transaction error: %w", err)
	}
	return &pr, nil
}

func (r *PRPostgresRepository) ReassignPR(ctx context.Context, pullRequestID, oldReviewerID string) (*model.PullRequest, string, error) {
	ok, err := r.PRExists(ctx, pullRequestID)
	if err != nil {
		return nil, "", err
	}
	if !ok {
		log.Printf("pr doesn't exist: %s", pullRequestID)
		return nil, "", model.NewNotFoundError()
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("begin transaction error: %v", err)
		return nil, "", fmt.Errorf("begin transaction error: %w", err)
	}
	defer func() {
		if err = tx.Rollback(); err != nil && err != sql.ErrTxDone {
			log.Printf("rollback transaction error in ReassignPR: %v", err)
		}
	}()

	status, err := r.GetStatus(ctx, tx, pullRequestID)
	if err != nil {
		return nil, "", err
	}
	if status == "MERGED" {
		log.Printf("pr merged: %s", pullRequestID)
		return nil, "", model.NewPRMergedsError()
	}

	exists, err := r.IsReviewer(ctx, tx, pullRequestID, oldReviewerID)
	if err != nil {
		return nil, "", err
	}
	if !exists {
		log.Printf("user is not a reviewer: %s", oldReviewerID)
		return nil, "", model.NewNotAssignedError()
	}

	newReviewerID, err := r.FindNewReviewer(ctx, tx, pullRequestID, oldReviewerID)
	if err != nil {
		return nil, "", err
	}
	if newReviewerID == "" {
		log.Printf("no new reviewer")
		return nil, "", model.NewNoCandidateError()
	}

	_, err = tx.ExecContext(ctx,
		`DELETE FROM reviewer_x_pr
		WHERE pr_id = $1 AND user_id = $2`,
		pullRequestID, oldReviewerID,
	)
	if err != nil {
		log.Printf("exec error: %v", err)
		return nil, "", fmt.Errorf("exec error: %w", err)
	}

	_, err = tx.ExecContext(ctx,
		`INSERT INTO reviewer_x_pr (pr_id, user_id)
		VALUES ($1, $2)`,
		pullRequestID, newReviewerID,
	)
	if err != nil {
		log.Printf("exec error: %v", err)
		return nil, "", fmt.Errorf("exec error: %w", err)
	}

	var pr model.PullRequest
	err = tx.QueryRowContext(ctx,
		`SELECT pr_id, pr_name, author_id, status
		FROM pr
		WHERE pr_id = $1`,
		pullRequestID,
	).Scan(&pr.PullRequestID, &pr.PullRequestName, &pr.AuthorID, &pr.Status)
	if err != nil {
		log.Printf("scan error: %v", err)
		return nil, "", fmt.Errorf("scan error: %w", err)
	}

	log.Printf("pr: %v", pr)
	reviewers, err := r.GetReviewers(ctx, tx, pullRequestID)
	if err != nil {
		return nil, "", err
	}

	pr.AssignedReviewers = reviewers

	if err := tx.Commit(); err != nil {
		log.Printf("commit transaction error: %v", err)
		return nil, "", fmt.Errorf("commit transaction error: %w", err)
	}
	return &pr, newReviewerID, nil
}

func (r *PRPostgresRepository) AuthorExists(ctx context.Context, authorID string) (bool, error) {
	result := r.db.QueryRowContext(ctx, `
		SELECT user_id
		FROM users
		WHERE user_id = $1`,
		authorID)

	var id string
	if err := result.Scan(&id); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		log.Printf("scan error: %v", err)
		return false, fmt.Errorf("scan error: %w", err)
	}

	return true, nil
}

func (r *PRPostgresRepository) PRExists(ctx context.Context, pullRequestID string) (bool, error) {
	result := r.db.QueryRowContext(ctx, `
		SELECT pr_id
		FROM pr
		WHERE pr_id = $1`,
		pullRequestID)

	var id string
	if err := result.Scan(&id); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		log.Printf("scan error: %v", err)
		return false, fmt.Errorf("scan error: %w", err)
	}

	return true, nil
}

func (r *PRPostgresRepository) FindReviewers(ctx context.Context, tx *sql.Tx, authorID string) ([]string, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT user_id FROM users
		WHERE team_name = (
			SELECT team_name
			FROM users
			WHERE user_id = $1)
		AND is_active = true
		AND user_id != $1
		ORDER BY RANDOM()
		LIMIT 2
		`, authorID)
	if err != nil {
		log.Printf("query error: %v", err)
		return nil, fmt.Errorf("query error: %w", err)
	}
	defer rows.Close()

	reviewers := []string{}
	for rows.Next() {
		var s string
		err := rows.Scan(&s)
		if err != nil {
			log.Printf("scan error: %v", err)
			return nil, fmt.Errorf("scan error: %w", err)
		}

		reviewers = append(reviewers, s)
	}

	return reviewers, nil
}

func (r *PRPostgresRepository) GetStatus(ctx context.Context, tx *sql.Tx, pullRequestID string) (string, error) {
	var status string
	err := tx.QueryRowContext(ctx, `
	SELECT status
	FROM pr
	WHERE pr_id = $1
	`, pullRequestID).Scan(&status)
	if err != nil {
		log.Printf("query row error: %v", err)
		return "", fmt.Errorf("query row error: %w", err)
	}

	return status, nil
}

func (r *PRPostgresRepository) IsReviewer(ctx context.Context, tx *sql.Tx, pullRequestID, userID string) (bool, error) {
	var exists bool
	err := tx.QueryRowContext(ctx,
		`SELECT EXISTS(
			SELECT 1
			FROM reviewer_x_pr
			WHERE pr_id = $1 AND user_id = $2)`,
		pullRequestID, userID,
	).Scan(&exists)
	if err != nil {
		log.Printf("scan error: %v", err)
		return false, fmt.Errorf("scan error: %w", err)
	}

	return exists, nil
}

func (r *PRPostgresRepository) FindNewReviewer(ctx context.Context, tx *sql.Tx, pullRequestID, oldReviewerID string) (string, error) {
	var newReviewerID string
	findReviewerQuery := `
        SELECT user_id
        FROM users
        WHERE team_name = (SELECT team_name FROM users WHERE user_id = $1)
        AND user_id != $1
        AND user_id != (SELECT author_id FROM pr WHERE pr_id = $2)
        AND is_active = true
        AND user_id NOT IN (
            SELECT user_id FROM reviewer_x_pr WHERE pr_id = $2
        )
        ORDER BY RANDOM()
        LIMIT 1
    `

	err := tx.QueryRowContext(ctx, findReviewerQuery, oldReviewerID, pullRequestID).Scan(&newReviewerID)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Println("no new reviewer")
			return "", nil
		}
		log.Printf("query row error: %v", err)
		return "", fmt.Errorf("query row error: %w", err)
	}

	return newReviewerID, nil
}

func (r *PRPostgresRepository) GetReviewers(ctx context.Context, tx *sql.Tx, pullRequestID string) ([]string, error) {
	rows, err := tx.QueryContext(ctx, `
	SELECT user_id
	FROM reviewer_x_pr
	WHERE pr_id = $1`,
		pullRequestID)
	if err != nil {
		log.Printf("query error: %v", err)
		return nil, fmt.Errorf("query error: %w", err)
	}
	defer rows.Close()

	reviewers := []string{}
	for rows.Next() {
		var reviewer string
		if err := rows.Scan(&reviewer); err != nil {
			log.Printf("scan error: %v", err)
			return nil, fmt.Errorf("scan error: %w", err)
		}
		reviewers = append(reviewers, reviewer)
	}

	return reviewers, nil
}

func (r *PRPostgresRepository) GetPR(ctx context.Context, tx *sql.Tx, pullRequestID string) (*model.PullRequest, error) {
	row := tx.QueryRowContext(ctx, `
		SELECT pr_name, author_id, status, merged_at
		FROM pr
		WHERE pr_id = $1
		`, pullRequestID)

	var pr model.PullRequest
	if err := row.Scan(&pr.PullRequestName, &pr.AuthorID, &pr.Status, &pr.MergedAt); err != nil {
		log.Printf("scan error: %v", err)
		return nil, fmt.Errorf("scan error: %w", err)
	}

	reviewers, err := r.GetReviewers(ctx, tx, pullRequestID)
	if err != nil {
		return nil, err
	}
	pr.AssignedReviewers = reviewers
	pr.PullRequestID = pullRequestID

	return &pr, nil
}
