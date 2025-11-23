package integration

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/karambo3a/avito_test_task/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreatePR(t *testing.T) {
	client := NewClient("http://localhost:" + os.Getenv("TEST_SERVICE_PORT"))
	dbVerifier := setupDBVerifier(t)
	defer dbVerifier.Close()

	timestamp := time.Now().UnixNano()
	teamName := fmt.Sprintf("pr-test-team-%d", timestamp)
	authorID := fmt.Sprintf("author-test-%d", timestamp)
	nonExistentAuthorID := fmt.Sprintf("non-existent-author-%d", timestamp)

	team := &model.Team{
		TeamName: teamName,
		Members: []model.TeamMember{
			{
				UserID:   authorID,
				Username: "Author User",
				IsActive: true,
			},
			{
				UserID:   fmt.Sprintf("reviewer-test-%d-1", timestamp),
				Username: "Reviewer 1",
				IsActive: true,
			},
			{
				UserID:   fmt.Sprintf("reviewer-test-%d-2", timestamp),
				Username: "Reviewer 2",
				IsActive: true,
			},
			{
				UserID:   fmt.Sprintf("reviewer-test-%d-3", timestamp),
				Username: "Reviewer 3",
				IsActive: false,
			},
		},
	}

	_, statusCode, err := client.AddTeam(team)
	require.NoError(t, err, "Adding team should not fail")
	require.Equal(t, http.StatusCreated, statusCode, "Team creation should succeed")

	testCases := []struct {
		name           string
		prID           string
		prName         string
		authorID       string
		expectedStatus int
		errorCode      string
	}{
		{
			name:           "Valid PR",
			prID:           fmt.Sprintf("pr-%d-1", timestamp),
			prName:         "Test PR 1",
			authorID:       authorID,
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "Duplicate PR ID",
			prID:           fmt.Sprintf("pr-%d-1", timestamp),
			prName:         "Test PR Duplicate",
			authorID:       authorID,
			expectedStatus: http.StatusConflict,
			errorCode:      model.CodePRExists,
		},
		{
			name:           "Non-existent author",
			prID:           fmt.Sprintf("pr-%d-2", timestamp),
			prName:         "Test PR 2",
			authorID:       nonExistentAuthorID,
			expectedStatus: http.StatusNotFound,
			errorCode:      model.CodeNotFound,
		},
		{
			name:           "Empty PR ID",
			prID:           "",
			prName:         "Test PR Empty ID",
			authorID:       authorID,
			expectedStatus: http.StatusBadRequest,
			errorCode:      model.CodeEmptyField,
		},
		{
			name:           "Empty PR name",
			prID:           fmt.Sprintf("pr-%d-3", timestamp),
			prName:         "",
			authorID:       authorID,
			expectedStatus: http.StatusBadRequest,
			errorCode:      "EMPTY_FIELD",
		},
		{
			name:           "Empty author ID",
			prID:           fmt.Sprintf("pr-%d-4", timestamp),
			prName:         "Test PR 4",
			authorID:       "",
			expectedStatus: http.StatusBadRequest,
			errorCode:      model.CodeEmptyField,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, statusCode, err := client.CreatePR(tc.prID, tc.prName, tc.authorID)
			require.NoError(t, err, "API call should not fail")
			assert.Equal(t, tc.expectedStatus, statusCode, "Status code should match expected")

			switch statusCode {
			case http.StatusCreated:
				respData, ok := resp.(map[string]interface{})
				require.True(t, ok, "Response should be a map")

				prObj, ok := respData["pr"]
				require.True(t, ok, "Response should have 'pr' field")

				prMap, ok := prObj.(map[string]interface{})
				require.True(t, ok, "PR data should be a map")

				assert.Equal(t, tc.prID, prMap["pull_request_id"], "PR ID should match")
				assert.Equal(t, tc.prName, prMap["pull_request_name"], "PR name should match")
				assert.Equal(t, tc.authorID, prMap["author_id"], "Author ID should match")
				assert.Equal(t, "OPEN", prMap["status"], "Status should be OPEN")

				reviewers, ok := prMap["assigned_reviewers"].([]interface{})
				require.True(t, ok, "Assigned reviewers should be an array")
				assert.LessOrEqual(t, len(reviewers), 2, "Should have at most 2 reviewers")

				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				exists, err := dbVerifier.VerifyPullRequestExists(ctx, tc.prID)
				assert.NoError(t, err, "Database verification should not fail")
				assert.True(t, exists, "PR should exist in database")

				dbPR, err := dbVerifier.GetPullRequest(ctx, tc.prID)
				assert.NoError(t, err, "Getting PR from database should not fail")
				assert.Equal(t, tc.prID, dbPR.PullRequestID, "PR ID in database should match")
				assert.Equal(t, tc.prName, dbPR.PullRequestName, "PR name in database should match")
				assert.Equal(t, tc.authorID, dbPR.AuthorID, "Author ID in database should match")
				assert.Equal(t, "OPEN", dbPR.Status, "Status in database should be OPEN")
				assert.LessOrEqual(t, len(dbPR.AssignedReviewers), 2, "Should have at most 2 reviewers in database")

			case http.StatusNotFound, http.StatusBadRequest, http.StatusConflict:
				respData, ok := resp.(map[string]interface{})
				require.True(t, ok, "Response should be a map")

				errorObj, ok := respData["error"]
				require.True(t, ok, "Error response should have 'error' field")

				if tc.errorCode != "" {
					errorMap, ok := errorObj.(map[string]interface{})
					if ok {
						if code, exists := errorMap["code"]; exists {
							assert.Equal(t, tc.errorCode, code, "Error code should match expected")
						}
					}
				}

				if tc.prID != "" && statusCode != http.StatusConflict {
					ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer cancel()

					exists, err := dbVerifier.VerifyPullRequestExists(ctx, tc.prID)
					assert.NoError(t, err, "Database verification should not fail")
					assert.False(t, exists, "PR should not exist in database")
				}
			}
		})
	}
}

func TestMergePR(t *testing.T) {
	client := NewClient("http://localhost:" + os.Getenv("TEST_SERVICE_PORT"))
	dbVerifier := setupDBVerifier(t)
	defer dbVerifier.Close()

	timestamp := time.Now().UnixNano()
	teamName := fmt.Sprintf("merge-test-team-%d", timestamp)
	authorID := fmt.Sprintf("merge-author-%d", timestamp)
	prID := fmt.Sprintf("merge-pr-%d", timestamp)
	prName := "Test Merge PR"

	team := &model.Team{
		TeamName: teamName,
		Members: []model.TeamMember{
			{
				UserID:   authorID,
				Username: "Merge Author",
				IsActive: true,
			},
			{
				UserID:   fmt.Sprintf("merge-reviewer-%d", timestamp),
				Username: "Merge Reviewer",
				IsActive: true,
			},
		},
	}

	_, statusCode, err := client.AddTeam(team)
	require.NoError(t, err, "Adding team should not fail")
	require.Equal(t, http.StatusCreated, statusCode, "Team creation should succeed")

	_, statusCode, err = client.CreatePR(prID, prName, authorID)
	require.NoError(t, err, "Creating PR should not fail")
	require.Equal(t, http.StatusCreated, statusCode, "PR creation should succeed")

	testCases := []struct {
		name           string
		prID           string
		expectedStatus int
		errorCode      string
	}{
		{
			name:           "Merge existing PR",
			prID:           prID,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Merge already merged PR (idempotent)",
			prID:           prID,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Merge non-existent PR",
			prID:           "non-existent-pr",
			expectedStatus: http.StatusNotFound,
			errorCode:      model.CodeNotFound,
		},
		{
			name:           "Empty PR ID",
			prID:           "",
			expectedStatus: http.StatusBadRequest,
			errorCode:      model.CodeEmptyField,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, statusCode, err := client.MergePR(tc.prID)
			require.NoError(t, err, "API call should not fail")
			assert.Equal(t, tc.expectedStatus, statusCode, "Status code should match expected")

			switch statusCode {
			case http.StatusOK:
				respData, ok := resp.(map[string]interface{})
				require.True(t, ok, "Response should be a map")

				prObj, ok := respData["pr"]
				require.True(t, ok, "Response should have 'pr' field")

				prMap, ok := prObj.(map[string]interface{})
				require.True(t, ok, "PR data should be a map")

				assert.Equal(t, tc.prID, prMap["pull_request_id"], "PR ID should match")
				assert.Equal(t, "MERGED", prMap["status"], "Status should be MERGED")
				assert.NotNil(t, prMap["mergedAt"], "MergedAt should not be nil")

				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				dbPR, err := dbVerifier.GetPullRequest(ctx, tc.prID)
				assert.NoError(t, err, "Getting PR from database should not fail")
				assert.Equal(t, tc.prID, dbPR.PullRequestID, "PR ID in database should match")
				assert.Equal(t, "MERGED", dbPR.Status, "Status in database should be MERGED")
				assert.NotEmpty(t, dbPR.MergedAt, "MergedAt in database should not be empty")

			case http.StatusNotFound, http.StatusBadRequest:
				respData, ok := resp.(map[string]interface{})
				require.True(t, ok, "Response should be a map")

				errorObj, ok := respData["error"]
				require.True(t, ok, "Error response should have 'error' field")

				if tc.errorCode != "" {
					errorMap, ok := errorObj.(map[string]interface{})
					if ok {
						if code, exists := errorMap["code"]; exists {
							assert.Equal(t, tc.errorCode, code, "Error code should match expected")
						}
					}
				}
			}
		})
	}
}

func TestReassignPR(t *testing.T) {
	client := NewClient("http://localhost:" + os.Getenv("TEST_SERVICE_PORT"))
	dbVerifier := setupDBVerifier(t)
	defer dbVerifier.Close()

	timestamp := time.Now().UnixNano()
	teamName := fmt.Sprintf("reassign-test-team-%d", timestamp)
	authorID := fmt.Sprintf("reassign-author-%d", timestamp)
	reviewer1ID := fmt.Sprintf("reassign-reviewer1-%d", timestamp)
	reviewer2ID := fmt.Sprintf("reassign-reviewer2-%d", timestamp)
	reviewer3ID := fmt.Sprintf("reassign-reviewer3-%d", timestamp)
	prID := fmt.Sprintf("reassign-pr-%d", timestamp)
	prName := "Test Reassign PR"
	mergedPRID := fmt.Sprintf("reassign-merged-pr-%d", timestamp)

	team := &model.Team{
		TeamName: teamName,
		Members: []model.TeamMember{
			{
				UserID:   authorID,
				Username: "Reassign Author",
				IsActive: true,
			},
			{
				UserID:   reviewer1ID,
				Username: "Reassign Reviewer 1",
				IsActive: true,
			},
			{
				UserID:   reviewer2ID,
				Username: "Reassign Reviewer 2",
				IsActive: true,
			},
			{
				UserID:   reviewer3ID,
				Username: "Reassign Reviewer 3",
				IsActive: true,
			},
		},
	}

	_, statusCode, err := client.AddTeam(team)
	require.NoError(t, err, "Adding team should not fail")
	require.Equal(t, http.StatusCreated, statusCode, "Team creation should succeed")

	_, statusCode, err = client.CreatePR(prID, prName, authorID)
	require.NoError(t, err, "Creating PR should not fail")
	require.Equal(t, http.StatusCreated, statusCode, "PR creation should succeed")

	_, statusCode, err = client.CreatePR(mergedPRID, "Merged PR", authorID)
	require.NoError(t, err, "Creating merged PR should not fail")
	require.Equal(t, http.StatusCreated, statusCode, "Merged PR creation should succeed")

	_, statusCode, err = client.MergePR(mergedPRID)
	require.NoError(t, err, "Merging PR should not fail")
	require.Equal(t, http.StatusOK, statusCode, "PR merge should succeed")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dbPR, err := dbVerifier.GetPullRequest(ctx, prID)
	require.NoError(t, err, "Getting PR from database should not fail")
	require.NotEmpty(t, dbPR.AssignedReviewers, "PR should have assigned reviewers")

	oldReviewerID := dbPR.AssignedReviewers[0]

	testCases := []struct {
		name           string
		prID           string
		oldUserID      string
		expectedStatus int
		errorCode      string
	}{
		{
			name:           "Reassign valid reviewer",
			prID:           prID,
			oldUserID:      oldReviewerID,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Reassign non-assigned reviewer",
			prID:           prID,
			oldUserID:      authorID,
			expectedStatus: http.StatusConflict,
			errorCode:      model.CodeNotAssigned,
		},
		{
			name:           "Reassign on merged PR",
			prID:           mergedPRID,
			oldUserID:      oldReviewerID,
			expectedStatus: http.StatusConflict,
			errorCode:      model.CodePRMerged,
		},
		{
			name:           "Reassign with non-existent PR",
			prID:           "non-existent-pr",
			oldUserID:      oldReviewerID,
			expectedStatus: http.StatusNotFound,
			errorCode:      model.CodeNotFound,
		},
		{
			name:           "Empty PR ID",
			prID:           "",
			oldUserID:      oldReviewerID,
			expectedStatus: http.StatusBadRequest,
			errorCode:      model.CodeEmptyField,
		},
		{
			name:           "Empty old user ID",
			prID:           prID,
			oldUserID:      "",
			expectedStatus: http.StatusBadRequest,
			errorCode:      model.CodeEmptyField,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, statusCode, err := client.ReassignPR(tc.prID, tc.oldUserID)
			require.NoError(t, err, "API call should not fail")
			assert.Equal(t, tc.expectedStatus, statusCode, "Status code should match expected")

			switch statusCode {
			case http.StatusOK:
				respData, ok := resp.(map[string]interface{})
				require.True(t, ok, "Response should be a map")

				prObj, ok := respData["pr"]
				require.True(t, ok, "Response should have 'pr' field")

				prMap, ok := prObj.(map[string]interface{})
				require.True(t, ok, "PR data should be a map")

				replacedBy, ok := respData["replaced_by"]
				require.True(t, ok, "Response should have 'replaced_by' field")
				assert.NotEqual(t, tc.oldUserID, replacedBy, "Replaced by should be different from old user ID")

				reviewers, ok := prMap["assigned_reviewers"].([]interface{})
				require.True(t, ok, "Assigned reviewers should be an array")

				for _, reviewer := range reviewers {
					assert.NotEqual(t, tc.oldUserID, reviewer, "Old reviewer should not be in the list")
				}

				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				dbPR, err := dbVerifier.GetPullRequest(ctx, tc.prID)
				assert.NoError(t, err, "Getting PR from database should not fail")

				for _, reviewer := range dbPR.AssignedReviewers {
					assert.NotEqual(t, tc.oldUserID, reviewer, "Old reviewer should not be in the database")
				}

				newReviewerFound := false
				for _, reviewer := range dbPR.AssignedReviewers {
					if reviewer == replacedBy {
						newReviewerFound = true
						break
					}
				}
				assert.True(t, newReviewerFound, "New reviewer should be in the database")

			case http.StatusNotFound, http.StatusBadRequest, http.StatusConflict:
				respData, ok := resp.(map[string]interface{})
				require.True(t, ok, "Response should be a map")

				errorObj, ok := respData["error"]
				require.True(t, ok, "Error response should have 'error' field")

				if tc.errorCode != "" {
					errorMap, ok := errorObj.(map[string]interface{})
					if ok {
						if code, exists := errorMap["code"]; exists {
							assert.Equal(t, tc.errorCode, code, "Error code should match expected")
						}
					}
				}
			}
		})
	}
}
