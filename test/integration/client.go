package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/karambo3a/avito_test_task/internal/model"
)

type Client struct {
	baseURL string
	client  *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		client:  &http.Client{Timeout: 20 * time.Second},
	}
}

func (c *Client) doRequest(method, path string, queryParams url.Values, body interface{}) ([]byte, int, error) {
	reqURL := c.baseURL + path
	if len(queryParams) > 0 {
		reqURL += "?" + queryParams.Encode()
	}

	var req *http.Request
	var err error
	if body != nil {
		var bodyReader io.Reader
		bodyReader, err = toReader(body)
		if err != nil {
			return nil, -1, fmt.Errorf("failed to serialize request body: %w", err)
		}
		req, err = http.NewRequest(method, reqURL, bodyReader)
	} else {
		req, err = http.NewRequest(method, reqURL, nil)
	}

	if err != nil {
		return nil, -1, fmt.Errorf("failed to create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.client.Do(req)
	if err != nil {
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			return nil, -1, fmt.Errorf("request timeout")
		}
		return nil, -1, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("failed to read response body: %w", err)
	}

	return respBody, resp.StatusCode, nil
}

// Team endpoints

func (c *Client) AddTeam(team *model.Team) (any, int, error) {
	respBody, statusCode, err := c.doRequest(http.MethodPost, "/team/add", nil, team)
	if err != nil {
		return nil, statusCode, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, statusCode, fmt.Errorf("failed to parse response: %w", err)
	}

	return result, statusCode, nil
}

func (c *Client) GetTeam(teamName string) (any, int, error) {
	params := url.Values{}
	params.Add("team_name", teamName)

	respBody, statusCode, err := c.doRequest(http.MethodGet, "/team/get", params, nil)
	if err != nil {
		return nil, statusCode, err
	}

	if statusCode != http.StatusOK {
		var errorResp map[string]interface{}
		if err := json.Unmarshal(respBody, &errorResp); err != nil {
			return nil, statusCode, fmt.Errorf("failed to parse error response: %w", err)
		}
		return errorResp, statusCode, nil
	}

	var team model.Team
	if err := json.Unmarshal(respBody, &team); err != nil {
		return nil, statusCode, fmt.Errorf("failed to parse team response: %w", err)
	}

	return team, statusCode, nil
}

// User endpoints

func (c *Client) SetUserIsActive(userID string, isActive bool) (any, int, error) {
	reqBody := map[string]interface{}{
		"user_id":   userID,
		"is_active": isActive,
	}

	respBody, statusCode, err := c.doRequest(http.MethodPost, "/users/setIsActive", nil, reqBody)
	if err != nil {
		return nil, statusCode, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, statusCode, fmt.Errorf("failed to parse response: %w", err)
	}

	return result, statusCode, nil
}

func (c *Client) GetUserReview(userID string) (any, int, error) {
	params := url.Values{}
	params.Add("user_id", userID)

	respBody, statusCode, err := c.doRequest(http.MethodGet, "/users/getReview", params, nil)
	if err != nil {
		return nil, statusCode, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, statusCode, fmt.Errorf("failed to parse response: %w", err)
	}

	return result, statusCode, nil
}

// Pull Request endpoints

func (c *Client) CreatePR(pullRequestID, pullRequestName, authorID string) (any, int, error) {
	reqBody := map[string]interface{}{
		"pull_request_id":   pullRequestID,
		"pull_request_name": pullRequestName,
		"author_id":         authorID,
	}

	respBody, statusCode, err := c.doRequest(http.MethodPost, "/pullRequest/create", nil, reqBody)
	if err != nil {
		return nil, statusCode, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, statusCode, fmt.Errorf("failed to parse response: %w", err)
	}

	return result, statusCode, nil
}

func (c *Client) MergePR(pullRequestID string) (any, int, error) {
	reqBody := map[string]interface{}{
		"pull_request_id": pullRequestID,
	}

	respBody, statusCode, err := c.doRequest(http.MethodPost, "/pullRequest/merge", nil, reqBody)
	if err != nil {
		return nil, statusCode, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, statusCode, fmt.Errorf("failed to parse response: %w", err)
	}

	return result, statusCode, nil
}

func (c *Client) ReassignPR(pullRequestID, oldUserID string) (any, int, error) {
	reqBody := map[string]interface{}{
		"pull_request_id": pullRequestID,
		"old_user_id":     oldUserID,
	}

	respBody, statusCode, err := c.doRequest(http.MethodPost, "/pullRequest/reassign", nil, reqBody)
	if err != nil {
		return nil, statusCode, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, statusCode, fmt.Errorf("failed to parse response: %w", err)
	}

	return result, statusCode, nil
}

// Statistics endpoints

func (c *Client) GetUserStatistics(userID string) (any, int, error) {
	params := url.Values{}
	params.Add("user_id", userID)

	respBody, statusCode, err := c.doRequest(http.MethodGet, "/statistics/user", params, nil)
	if err != nil {
		return nil, statusCode, err
	}

	if statusCode != http.StatusOK {
		var errorResp map[string]interface{}
		if err := json.Unmarshal(respBody, &errorResp); err != nil {
			return nil, statusCode, fmt.Errorf("failed to parse error response: %w", err)
		}
		return errorResp, statusCode, nil
	}

	var stats model.UserStatistics
	if err := json.Unmarshal(respBody, &stats); err != nil {
		return nil, statusCode, fmt.Errorf("failed to parse response: %w", err)
	}

	return stats, statusCode, nil
}

func (c *Client) GetTeamStatistics(teamName string) (any, int, error) {
	params := url.Values{}
	params.Add("team_name", teamName)

	respBody, statusCode, err := c.doRequest(http.MethodGet, "/statistics/team", params, nil)
	if err != nil {
		return nil, statusCode, err
	}

	if statusCode != http.StatusOK {
		var errorResp map[string]interface{}
		if err := json.Unmarshal(respBody, &errorResp); err != nil {
			return nil, statusCode, fmt.Errorf("failed to parse error response: %w", err)
		}
		return errorResp, statusCode, nil
	}

	var stats model.TeamStatistics
	if err := json.Unmarshal(respBody, &stats); err != nil {
		return nil, statusCode, fmt.Errorf("failed to parse response: %w", err)
	}

	return stats, statusCode, nil
}

func toReader(data any) (io.Reader, error) {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(jsonBytes), nil
}
