package integration

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/karambo3a/avito_test_task/internal/model"
)

type DBVerifier struct {
	db *sql.DB
}

func NewDBVerifier(host, port, user, password, dbname string) (*DBVerifier, error) {
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		user, password, host, port, dbname, "disable")

	db, err := sql.Open("pgx", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DBVerifier{db: db}, nil
}

func (v *DBVerifier) Close() error {
	return v.db.Close()
}

func setupDBVerifier(t *testing.T) *DBVerifier {
	dbVerifier, err := NewDBVerifier(
		os.Getenv("LOCALHOST"),
		os.Getenv("TEST_DATABASE_PORT"),
		os.Getenv("TEST_DATABASE_USER"),
		os.Getenv("TEST_DATABASE_PASSWORD"),
		os.Getenv("TEST_DATABASE_NAME"),
	)
	if err != nil {
		t.Fatalf("failed to create DB verifier: %v", err)
	}
	return dbVerifier
}

func (v *DBVerifier) VerifyTeamExists(ctx context.Context, teamName string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM team WHERE team_name = $1)`

	err := v.db.QueryRowContext(ctx, query, teamName).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to verify team existence: %w", err)
	}

	return exists, nil
}

func (v *DBVerifier) GetTeam(ctx context.Context, teamName string) (*model.Team, error) {
	exists, err := v.VerifyTeamExists(ctx, teamName)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("team not found: %s", teamName)
	}

	query := `
		SELECT u.user_id, u.username, u.is_active
		FROM users u
		WHERE u.team_name = $1
	`

	rows, err := v.db.QueryContext(ctx, query, teamName)
	if err != nil {
		return nil, fmt.Errorf("failed to query team members: %w", err)
	}
	defer rows.Close()

	var members []model.TeamMember
	for rows.Next() {
		var member model.TeamMember
		if err := rows.Scan(&member.UserID, &member.Username, &member.IsActive); err != nil {
			return nil, fmt.Errorf("failed to scan team member: %w", err)
		}
		members = append(members, member)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating team members: %w", err)
	}

	return &model.Team{
		TeamName: teamName,
		Members:  members,
	}, nil
}

func (v *DBVerifier) VerifyUserExists(ctx context.Context, userID string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE user_id = $1)`

	err := v.db.QueryRowContext(ctx, query, userID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to verify user existence: %w", err)
	}

	return exists, nil
}

func (v *DBVerifier) GetUser(ctx context.Context, userID string) (*model.User, error) {
	query := `SELECT user_id, username, team_name, is_active FROM users WHERE user_id = $1`

	var user model.User
	err := v.db.QueryRowContext(ctx, query, userID).Scan(
		&user.UserID, &user.Username, &user.TeamName, &user.IsActive,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user %s not found", userID)
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

func (v *DBVerifier) VerifyPullRequestExists(ctx context.Context, prID string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM pr WHERE pr_id = $1)`

	err := v.db.QueryRowContext(ctx, query, prID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to verify PR existence: %w", err)
	}

	return exists, nil
}

func (v *DBVerifier) GetPullRequest(ctx context.Context, prID string) (*model.PullRequest, error) {
	prQuery := `
		SELECT pr_id, pr_name, author_id, status, created_at, merged_at
		FROM pr
		WHERE pr_id = $1
	`

	var pr model.PullRequest
	var createdAt, mergedAt sql.NullString

	err := v.db.QueryRowContext(ctx, prQuery, prID).Scan(
		&pr.PullRequestID, &pr.PullRequestName, &pr.AuthorID, &pr.Status, &createdAt, &mergedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("pull request %s not found", prID)
		}
		return nil, fmt.Errorf("failed to get pull request: %w", err)
	}

	if createdAt.Valid {
		pr.CreatedAt = createdAt.String
	}

	if mergedAt.Valid {
		pr.MergedAt = mergedAt.String
	}

	reviewersQuery := `
		SELECT user_id FROM reviewer_x_pr WHERE pr_id = $1
	`

	rows, err := v.db.QueryContext(ctx, reviewersQuery, prID)
	if err != nil {
		return nil, fmt.Errorf("failed to query reviewers: %w", err)
	}
	defer rows.Close()

	var reviewers []string
	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err != nil {
			return nil, fmt.Errorf("failed to scan reviewer: %w", err)
		}
		reviewers = append(reviewers, userID)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating reviewers: %w", err)
	}

	pr.AssignedReviewers = reviewers

	return &pr, nil
}

func (v *DBVerifier) GetUserStatistics(ctx context.Context, userID string) (*model.UserStatistics, error) {
	exists, err := v.VerifyUserExists(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("user not found: %s", userID)
	}

	userQuery := `
		SELECT user_id, username, team_name FROM users WHERE user_id = $1
	`

	var stats model.UserStatistics
	err = v.db.QueryRowContext(ctx, userQuery, userID).Scan(
		&stats.UserID, &stats.Username, &stats.TeamName,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	authoredQuery := `
		SELECT COUNT(*) FROM pr WHERE author_id = $1
	`

	err = v.db.QueryRowContext(ctx, authoredQuery, userID).Scan(&stats.AuthoredPRsCount)
	if err != nil {
		return nil, fmt.Errorf("failed to count authored PRs: %w", err)
	}

	assignedQuery := `
		SELECT COUNT(*) FROM reviewer_x_pr WHERE user_id = $1
	`

	err = v.db.QueryRowContext(ctx, assignedQuery, userID).Scan(&stats.AssignedReviewsCount)
	if err != nil {
		return nil, fmt.Errorf("failed to count assigned reviews: %w", err)
	}

	return &stats, nil
}

func (v *DBVerifier) GetTeamStatistics(ctx context.Context, teamName string) (*model.TeamStatistics, error) {
	exists, err := v.VerifyTeamExists(ctx, teamName)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("team not found: %s", teamName)
	}

	var stats model.TeamStatistics
	stats.TeamName = teamName

	membersQuery := `
		SELECT user_id FROM users WHERE team_name = $1
	`

	rows, err := v.db.QueryContext(ctx, membersQuery, teamName)
	if err != nil {
		return nil, fmt.Errorf("failed to query team members: %w", err)
	}
	defer rows.Close()

	var memberIDs []string
	for rows.Next() {
		var userID string
		if err = rows.Scan(&userID); err != nil {
			return nil, fmt.Errorf("failed to scan member ID: %w", err)
		}
		memberIDs = append(memberIDs, userID)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating member IDs: %w", err)
	}

	if len(memberIDs) == 0 {
		return &stats, nil
	}

	totalQuery := `
		SELECT COUNT(*) FROM pr WHERE author_id IN (
			SELECT user_id FROM users WHERE team_name = $1
		)
	`

	err = v.db.QueryRowContext(ctx, totalQuery, teamName).Scan(&stats.TotalPRs)
	if err != nil {
		return nil, fmt.Errorf("failed to count total PRs: %w", err)
	}

	mergedQuery := `
		SELECT COUNT(*) FROM pr
		WHERE author_id IN (
			SELECT user_id FROM users WHERE team_name = $1
		) AND status = 'MERGED'
	`

	err = v.db.QueryRowContext(ctx, mergedQuery, teamName).Scan(&stats.MergedPRs)
	if err != nil {
		return nil, fmt.Errorf("failed to count merged PRs: %w", err)
	}

	openQuery := `
		SELECT COUNT(*) FROM pr
		WHERE author_id IN (
			SELECT user_id FROM users WHERE team_name = $1
		) AND status = 'OPEN'
	`

	err = v.db.QueryRowContext(ctx, openQuery, teamName).Scan(&stats.OpenPRs)
	if err != nil {
		return nil, fmt.Errorf("failed to count open PRs: %w", err)
	}

	return &stats, nil
}
