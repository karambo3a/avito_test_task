package model

type UserStatistics struct {
	UserID               string `json:"user_id"`
	Username             string `json:"username"`
	TeamName             string `json:"team_name"`
	AssignedReviewsCount int    `json:"assigned_reviews_count"`
	AuthoredPRsCount     int    `json:"authored_prs_count"`
}

type TeamStatistics struct {
	TeamName  string `json:"team_name"`
	TotalPRs  int    `json:"total_prs"`
	MergedPRs int    `json:"merged_prs"`
	OpenPRs   int    `json:"open_prs"`
}
