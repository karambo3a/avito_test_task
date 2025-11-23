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

func TestAddTeam(t *testing.T) {
	client := NewClient("http://localhost:" + os.Getenv("TEST_SERVICE_PORT"))
	dbVerifier := setupDBVerifier(t)
	defer dbVerifier.Close()

	timestamp := time.Now().UnixNano()
	uniqueTeamName := fmt.Sprintf("backend-team-%d", timestamp)
	duplicateTeamName := uniqueTeamName

	testCases := []struct {
		name           string
		team           *model.Team
		expectedStatus int
		errorCode      string
	}{
		{
			name: "Valid team",
			team: &model.Team{
				TeamName: uniqueTeamName,
				Members: []model.TeamMember{
					{
						UserID:   "user-backend-001",
						Username: "Alice Backend Developer",
						IsActive: true,
					},
					{
						UserID:   "user-backend-002",
						Username: "Bob Backend Engineer",
						IsActive: true,
					},
					{
						UserID:   "user-backend-003",
						Username: "Charlie Backend Architect",
						IsActive: true,
					},
				},
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "Team with duplicate name",
			team: &model.Team{
				TeamName: duplicateTeamName,
				Members: []model.TeamMember{
					{
						UserID:   "user-backend-004",
						Username: "Dave Backend Developer",
						IsActive: true,
					},
				},
			},
			expectedStatus: http.StatusBadRequest,
			errorCode:      model.CodeTeamExists,
		},
		{
			name: "Team with empty name",
			team: &model.Team{
				TeamName: "",
				Members: []model.TeamMember{
					{
						UserID:   "user-backend-005",
						Username: "Eve Backend Developer",
						IsActive: true,
					},
				},
			},
			expectedStatus: http.StatusBadRequest,
			errorCode:      model.CodeEmptyField,
		},
		{
			name: "Team with no members",
			team: &model.Team{
				TeamName: fmt.Sprintf("empty-team-%d", timestamp),
				Members:  []model.TeamMember{},
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "Team with member missing required fields",
			team: &model.Team{
				TeamName: fmt.Sprintf("invalid-member-team-%d", timestamp),
				Members: []model.TeamMember{
					{
						UserID:   "",
						Username: "Missing UserID",
						IsActive: true,
					},
				},
			},
			expectedStatus: http.StatusBadRequest,
			errorCode:      model.CodeEmptyField,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, statusCode, err := client.AddTeam(tc.team)
			require.NoError(t, err, "API call should not fail")
			assert.Equal(t, tc.expectedStatus, statusCode, "Status code should match expected")

			switch statusCode {
			case http.StatusCreated:
				respData, ok := resp.(map[string]interface{})
				require.True(t, ok, "Response should be a map")

				team, ok := respData["team"]
				require.True(t, ok, "Response should have 'team' field")

				teamMap, ok := team.(map[string]interface{})
				require.True(t, ok, "Team data should be a map")

				assert.Equal(t, tc.team.TeamName, teamMap["team_name"], "Team name should match")

				members, ok := teamMap["members"].([]interface{})
				require.True(t, ok, "Members should be an array")
				assert.Len(t, members, len(tc.team.Members), "Member count should match")

				for i, expectedMember := range tc.team.Members {
					if i < len(members) {
						member, ok := members[i].(map[string]interface{})
						require.True(t, ok, "Member should be a map")

						assert.Equal(t, expectedMember.UserID, member["user_id"], "UserID should match")
						assert.Equal(t, expectedMember.Username, member["username"], "Username should match")
						assert.Equal(t, expectedMember.IsActive, member["is_active"], "IsActive should match")
					}
				}

				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				exists, err := dbVerifier.VerifyTeamExists(ctx, tc.team.TeamName)
				assert.NoError(t, err, "Database verification should not fail")
				assert.True(t, exists, "Team should exist in database")

				dbTeam, err := dbVerifier.GetTeam(ctx, tc.team.TeamName)
				assert.NoError(t, err, "Getting team from database should not fail")
				assert.Equal(t, tc.team.TeamName, dbTeam.TeamName, "Team name in database should match")

				expectedMembers := make(map[string]model.TeamMember)
				for _, m := range tc.team.Members {
					if m.UserID != "" {
						expectedMembers[m.UserID] = m
					}
				}

				actualMembers := make(map[string]model.TeamMember)
				for _, m := range dbTeam.Members {
					actualMembers[m.UserID] = m
				}

				for userID, expectedMember := range expectedMembers {
					actualMember, exists := actualMembers[userID]
					assert.True(t, exists, "Member should exist in database")
					assert.Equal(t, expectedMember.Username, actualMember.Username, "Username should match")
					assert.Equal(t, expectedMember.IsActive, actualMember.IsActive, "IsActive should match")
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

				if tc.team.TeamName == "" {
					ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer cancel()

					exists, err := dbVerifier.VerifyTeamExists(ctx, tc.team.TeamName)
					assert.NoError(t, err, "Database verification should not fail")
					assert.False(t, exists, "Team with empty name should not exist in database")
				}
			}
		})
	}
}

func TestGetTeam(t *testing.T) {
	client := NewClient("http://localhost:" + os.Getenv("TEST_SERVICE_PORT"))
	dbVerifier := setupDBVerifier(t)
	defer dbVerifier.Close()

	timestamp := time.Now().UnixNano()
	teamName := fmt.Sprintf("frontend-team-%d", timestamp)

	team := &model.Team{
		TeamName: teamName,
		Members: []model.TeamMember{
			{
				UserID:   fmt.Sprintf("user-frontend-%d-001", timestamp),
				Username: "Frank Frontend Developer",
				IsActive: true,
			},
			{
				UserID:   fmt.Sprintf("user-frontend-%d-002", timestamp),
				Username: "Grace Frontend Engineer",
				IsActive: false,
			},
		},
	}

	_, statusCode, err := client.AddTeam(team)
	require.NoError(t, err, "Adding team should not fail")
	require.Equal(t, http.StatusCreated, statusCode, "Team creation should succeed")

	testCases := []struct {
		name           string
		teamName       string
		expectedStatus int
		errorCode      string
	}{
		{
			name:           "Existing team",
			teamName:       teamName,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Non-existent team",
			teamName:       "non-existent-team",
			expectedStatus: http.StatusNotFound,
			errorCode:      "NOT_FOUND",
		},
		{
			name:           "Empty team name",
			teamName:       "",
			expectedStatus: http.StatusBadRequest,
			errorCode:      "EMPTY_FIELD",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, statusCode, err := client.GetTeam(tc.teamName)
			require.NoError(t, err, "API call should not fail")
			assert.Equal(t, tc.expectedStatus, statusCode, "Status code should match expected")

			switch statusCode {
			case http.StatusOK:
				teamResp, ok := resp.(model.Team)
				require.True(t, ok, "Response should be a Team object")
				assert.Equal(t, tc.teamName, teamResp.TeamName, "Team name should match")
				assert.NotEmpty(t, teamResp.Members, "Team should have members")

				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				dbTeam, err := dbVerifier.GetTeam(ctx, tc.teamName)
				assert.NoError(t, err, "Getting team from database should not fail")
				assert.Equal(t, teamResp.TeamName, dbTeam.TeamName, "Team name should match database")
				assert.Len(t, teamResp.Members, len(dbTeam.Members), "Member count should match database")

				apiMembers := make(map[string]model.TeamMember)
				for _, m := range teamResp.Members {
					apiMembers[m.UserID] = m
				}

				dbMembers := make(map[string]model.TeamMember)
				for _, m := range dbTeam.Members {
					dbMembers[m.UserID] = m
				}

				for userID, apiMember := range apiMembers {
					dbMember, exists := dbMembers[userID]
					assert.True(t, exists, "Member should exist in database")
					assert.Equal(t, apiMember.Username, dbMember.Username, "Username should match")
					assert.Equal(t, apiMember.IsActive, dbMember.IsActive, "IsActive should match")
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

				if statusCode == http.StatusNotFound {
					ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer cancel()

					exists, err := dbVerifier.VerifyTeamExists(ctx, tc.teamName)
					assert.NoError(t, err, "Database verification should not fail")
					assert.False(t, exists, "Team should not exist in database")
				}
			}
		})
	}
}
