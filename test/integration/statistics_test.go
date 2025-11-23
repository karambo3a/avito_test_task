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

func TestGetUserStatistics(t *testing.T) {
	client := NewClient("http://localhost:" + os.Getenv("TEST_SERVICE_PORT"))
	dbVerifier := setupDBVerifier(t)
	defer dbVerifier.Close()

	timestamp := time.Now().UnixNano()
	teamName := fmt.Sprintf("stats-team-%d", timestamp)
	userID := fmt.Sprintf("stats-user-%d", timestamp)
	username := fmt.Sprintf("Stats User %d", timestamp)
	authorID := fmt.Sprintf("stats-author-%d", timestamp)

	team := &model.Team{
		TeamName: teamName,
		Members: []model.TeamMember{
			{
				UserID:   userID,
				Username: username,
				IsActive: true,
			},
			{
				UserID:   authorID,
				Username: "Stats Author",
				IsActive: true,
			},
		},
	}

	_, statusCode, err := client.AddTeam(team)
	require.NoError(t, err, "Adding team should not fail")
	require.Equal(t, http.StatusCreated, statusCode, "Team creation should succeed")

	for i := 0; i < 3; i++ {
		prID := fmt.Sprintf("stats-pr-%d-%d", timestamp, i)
		prName := fmt.Sprintf("Stats PR %d", i)

		_, statusCode, err = client.CreatePR(prID, prName, authorID)
		require.NoError(t, err, "Creating PR should not fail")
		require.Equal(t, http.StatusCreated, statusCode, "PR creation should succeed")

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		var dbPR *model.PullRequest
		dbPR, err = dbVerifier.GetPullRequest(ctx, prID)
		cancel()
		require.NoError(t, err, "Getting PR from database should not fail")

		userAssigned := false
		for _, reviewer := range dbPR.AssignedReviewers {
			if reviewer == userID {
				userAssigned = true
				break
			}
		}

		if !userAssigned && len(dbPR.AssignedReviewers) > 0 {
			_, statusCode, err = client.ReassignPR(prID, dbPR.AssignedReviewers[0])
			require.NoError(t, err, "Reassigning PR should not fail")
			require.Equal(t, http.StatusOK, statusCode, "PR reassignment should succeed")
		}
	}

	for i := 0; i < 2; i++ {
		prID := fmt.Sprintf("stats-user-pr-%d-%d", timestamp, i)
		prName := fmt.Sprintf("User Stats PR %d", i)

		_, statusCode, err = client.CreatePR(prID, prName, userID)
		require.NoError(t, err, "Creating PR should not fail")
		require.Equal(t, http.StatusCreated, statusCode, "PR creation should succeed")
	}

	testCases := []struct {
		name           string
		userID         string
		expectedStatus int
		errorCode      string
	}{
		{
			name:           "Get statistics for existing user",
			userID:         userID,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Get statistics for non-existent user",
			userID:         "non-existent-user",
			expectedStatus: http.StatusNotFound,
			errorCode:      model.CodeNotFound,
		},
		{
			name:           "Empty user ID",
			userID:         "",
			expectedStatus: http.StatusBadRequest,
			errorCode:      model.CodeEmptyField,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, statusCode, err := client.GetUserStatistics(tc.userID)
			require.NoError(t, err, "API call should not fail")
			assert.Equal(t, tc.expectedStatus, statusCode, "Status code should match expected")

			switch statusCode {
			case http.StatusOK:
				stats, ok := resp.(model.UserStatistics)
				require.True(t, ok, "Response should be a UserStatistics object")

				assert.Equal(t, tc.userID, stats.UserID, "User ID should match")
				assert.Equal(t, teamName, stats.TeamName, "Team name should match")

				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				dbStats, err := dbVerifier.GetUserStatistics(ctx, tc.userID)
				assert.NoError(t, err, "Getting user statistics from database should not fail")
				assert.Equal(t, stats.UserID, dbStats.UserID, "User ID in database should match")
				assert.Equal(t, stats.Username, dbStats.Username, "Username in database should match")
				assert.Equal(t, stats.TeamName, dbStats.TeamName, "Team name in database should match")
				assert.Equal(t, stats.AssignedReviewsCount, dbStats.AssignedReviewsCount, "Assigned reviews count in database should match")
				assert.Equal(t, stats.AuthoredPRsCount, dbStats.AuthoredPRsCount, "Authored PRs count in database should match")

				if tc.userID == userID {
					assert.GreaterOrEqual(t, stats.AuthoredPRsCount, 2, "User should have authored at least 2 PRs")
					assert.GreaterOrEqual(t, stats.AssignedReviewsCount, 1, "User should have been assigned at least 1 PR for review")
				}

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

func TestGetTeamStatistics(t *testing.T) {
	client := NewClient("http://localhost:" + os.Getenv("TEST_SERVICE_PORT"))
	dbVerifier := setupDBVerifier(t)
	defer dbVerifier.Close()

	timestamp := time.Now().UnixNano()
	teamName := fmt.Sprintf("team-stats-%d", timestamp)
	authorID1 := fmt.Sprintf("team-stats-author1-%d", timestamp)
	authorID2 := fmt.Sprintf("team-stats-author2-%d", timestamp)

	team := &model.Team{
		TeamName: teamName,
		Members: []model.TeamMember{
			{
				UserID:   authorID1,
				Username: "Team Stats Author 1",
				IsActive: true,
			},
			{
				UserID:   authorID2,
				Username: "Team Stats Author 2",
				IsActive: true,
			},
			{
				UserID:   fmt.Sprintf("team-stats-reviewer-%d", timestamp),
				Username: "Team Stats Reviewer",
				IsActive: true,
			},
		},
	}

	_, statusCode, err := client.AddTeam(team)
	require.NoError(t, err, "Adding team should not fail")
	require.Equal(t, http.StatusCreated, statusCode, "Team creation should succeed")

	for i := 0; i < 3; i++ {
		prID := fmt.Sprintf("team-stats-pr1-%d-%d", timestamp, i)
		prName := fmt.Sprintf("Team Stats PR 1-%d", i)

		_, statusCode, err = client.CreatePR(prID, prName, authorID1)
		require.NoError(t, err, "Creating PR should not fail")
		require.Equal(t, http.StatusCreated, statusCode, "PR creation should succeed")

		if i == 0 {
			_, statusCode, err = client.MergePR(prID)
			require.NoError(t, err, "Merging PR should not fail")
			require.Equal(t, http.StatusOK, statusCode, "PR merge should succeed")
		}
	}

	for i := 0; i < 2; i++ {
		prID := fmt.Sprintf("team-stats-pr2-%d-%d", timestamp, i)
		prName := fmt.Sprintf("Team Stats PR 2-%d", i)

		_, statusCode, err = client.CreatePR(prID, prName, authorID2)
		require.NoError(t, err, "Creating PR should not fail")
		require.Equal(t, http.StatusCreated, statusCode, "PR creation should succeed")

		if i == 0 {
			_, statusCode, err = client.MergePR(prID)
			require.NoError(t, err, "Merging PR should not fail")
			require.Equal(t, http.StatusOK, statusCode, "PR merge should succeed")
		}
	}

	testCases := []struct {
		name           string
		teamName       string
		expectedStatus int
		errorCode      string
	}{
		{
			name:           "Get statistics for existing team",
			teamName:       teamName,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Get statistics for non-existent team",
			teamName:       "non-existent-team",
			expectedStatus: http.StatusNotFound,
			errorCode:      model.CodeNotFound,
		},
		{
			name:           "Empty team name",
			teamName:       "",
			expectedStatus: http.StatusBadRequest,
			errorCode:      model.CodeEmptyField,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, statusCode, err := client.GetTeamStatistics(tc.teamName)
			require.NoError(t, err, "API call should not fail")
			assert.Equal(t, tc.expectedStatus, statusCode, "Status code should match expected")

			switch statusCode {
			case http.StatusOK:
				stats, ok := resp.(model.TeamStatistics)
				require.True(t, ok, "Response should be a TeamStatistics object")

				assert.Equal(t, tc.teamName, stats.TeamName, "Team name should match")

				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				dbStats, err := dbVerifier.GetTeamStatistics(ctx, tc.teamName)
				assert.NoError(t, err, "Getting team statistics from database should not fail")
				assert.Equal(t, stats.TeamName, dbStats.TeamName, "Team name in database should match")
				assert.Equal(t, stats.TotalPRs, dbStats.TotalPRs, "Total PRs in database should match")
				assert.Equal(t, stats.MergedPRs, dbStats.MergedPRs, "Merged PRs in database should match")
				assert.Equal(t, stats.OpenPRs, dbStats.OpenPRs, "Open PRs in database should match")

				if tc.teamName == teamName {
					assert.Equal(t, 5, stats.TotalPRs, "Team should have 5 PRs in total")
					assert.Equal(t, 2, stats.MergedPRs, "Team should have 2 merged PRs")
					assert.Equal(t, 3, stats.OpenPRs, "Team should have 3 open PRs")
				}

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
