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

func TestSetUserIsActive(t *testing.T) {
	client := NewClient("http://localhost:" + os.Getenv("TEST_SERVICE_PORT"))
	dbVerifier := setupDBVerifier(t)
	defer dbVerifier.Close()

	timestamp := time.Now().UnixNano()
	teamName := fmt.Sprintf("user-test-team-%d", timestamp)
	userID := fmt.Sprintf("user-test-%d", timestamp)
	username := fmt.Sprintf("Test User %d", timestamp)

	team := &model.Team{
		TeamName: teamName,
		Members: []model.TeamMember{
			{
				UserID:   userID,
				Username: username,
				IsActive: true,
			},
		},
	}

	_, statusCode, err := client.AddTeam(team)
	require.NoError(t, err, "Adding team should not fail")
	require.Equal(t, http.StatusCreated, statusCode, "Team creation should succeed")

	testCases := []struct {
		name           string
		userID         string
		isActive       bool
		expectedStatus int
		errorCode      string
	}{
		{
			name:           "Set existing user to inactive",
			userID:         userID,
			isActive:       false,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Set existing user back to active",
			userID:         userID,
			isActive:       true,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Set non-existent user",
			userID:         "non-existent-user",
			isActive:       true,
			expectedStatus: http.StatusNotFound,
			errorCode:      model.CodeNotFound,
		},
		{
			name:           "Empty user ID",
			userID:         "",
			isActive:       true,
			expectedStatus: http.StatusBadRequest,
			errorCode:      model.CodeEmptyField,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, statusCode, err := client.SetUserIsActive(tc.userID, tc.isActive)
			require.NoError(t, err, "API call should not fail")
			assert.Equal(t, tc.expectedStatus, statusCode, "Status code should match expected")

			switch statusCode {
			case http.StatusOK:
				respData, ok := resp.(map[string]interface{})
				require.True(t, ok, "Response should be a map")

				userObj, ok := respData["user"]
				require.True(t, ok, "Response should have 'user' field")

				userMap, ok := userObj.(map[string]interface{})
				require.True(t, ok, "User data should be a map")

				assert.Equal(t, tc.userID, userMap["user_id"], "User ID should match")
				assert.Equal(t, tc.isActive, userMap["is_active"], "IsActive should match")

				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				dbUser, err := dbVerifier.GetUser(ctx, tc.userID)
				assert.NoError(t, err, "Getting user from database should not fail")
				assert.Equal(t, tc.userID, dbUser.UserID, "User ID in database should match")
				assert.Equal(t, tc.isActive, dbUser.IsActive, "IsActive in database should match")

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

				if statusCode == http.StatusNotFound && tc.userID != "" {
					ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer cancel()

					exists, err := dbVerifier.VerifyUserExists(ctx, tc.userID)
					assert.NoError(t, err, "Database verification should not fail")
					assert.False(t, exists, "User should not exist in database")
				}
			}
		})
	}
}

func TestGetUserReview(t *testing.T) {
	client := NewClient("http://localhost:" + os.Getenv("TEST_SERVICE_PORT"))
	dbVerifier := setupDBVerifier(t)
	defer dbVerifier.Close()

	timestamp := time.Now().UnixNano()
	teamName := fmt.Sprintf("review-test-team-%d", timestamp)
	authorID := fmt.Sprintf("author-test-%d", timestamp)
	reviewerID := fmt.Sprintf("reviewer-test-%d", timestamp)
	prID := fmt.Sprintf("pr-test-%d", timestamp)
	prName := fmt.Sprintf("Test PR %d", timestamp)

	team := &model.Team{
		TeamName: teamName,
		Members: []model.TeamMember{
			{
				UserID:   authorID,
				Username: "Author User",
				IsActive: true,
			},
			{
				UserID:   reviewerID,
				Username: "Reviewer User",
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
		userID         string
		expectedStatus int
		errorCode      string
		shouldHavePRs  bool
	}{
		{
			name:           "Get reviews for reviewer",
			userID:         reviewerID,
			expectedStatus: http.StatusOK,
			shouldHavePRs:  true,
		},
		{
			name:           "Get reviews for author (should be empty)",
			userID:         authorID,
			expectedStatus: http.StatusOK,
			shouldHavePRs:  false,
		},
		{
			name:           "Get reviews for non-existent user",
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
			resp, statusCode, err := client.GetUserReview(tc.userID)
			require.NoError(t, err, "API call should not fail")
			assert.Equal(t, tc.expectedStatus, statusCode, fmt.Sprintf("Status code should match expected: %v (want) != %v (expected)", tc.expectedStatus, statusCode))

			switch statusCode {
			case http.StatusOK:
				respData, ok := resp.(map[string]interface{})
				require.True(t, ok, "Response should be a map")

				userIDFromResp, ok := respData["user_id"]
				require.True(t, ok, "Response should have 'user_id' field")
				assert.Equal(t, tc.userID, userIDFromResp, "User ID should match")

				pullRequests, ok := respData["pull_requests"]
				require.True(t, ok, "Response should have 'pull_requests' field")

				prs, ok := pullRequests.([]interface{})
				require.True(t, ok, "Pull requests should be an array")

				if tc.shouldHavePRs {
					assert.NotEmpty(t, prs, "User should have assigned PRs")

					if len(prs) > 0 {
						found := false
						for _, pr := range prs {
							prMap, ok := pr.(map[string]interface{})
							if !ok {
								continue
							}

							if extractedPRID, ok := prMap["pull_request_id"]; ok && extractedPRID == prID {
								found = true
								assert.Equal(t, prName, prMap["pull_request_name"], "PR name should match")
								assert.Equal(t, authorID, prMap["author_id"], "Author ID should match")
								break
							}
						}
						assert.True(t, found, "Created PR should be in the list")
					}
				} else {
					assert.Empty(t, prs, "User should not have assigned PRs")
				}

			case http.StatusBadRequest:
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
