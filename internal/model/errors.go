package model

import "fmt"

const (
	CodeTeamExists  = "TEAM_EXISTS"
	CodePRExists    = "PR_EXISTS"
	CodePRMerged    = "PR_MERGED"
	CodeNotAssigned = "NOT_ASSIGNED"
	CodeNoCandidate = "NO_CANDIDATE"
	CodeNotFound    = "NOT_FOUND"
	CodeEmptyField  = "EMPTY_FIELD"

	MsgTeamExists  = "team_name already exists"
	MsgPRExists    = "PR id already exists"
	MsgPRMerged    = "cannot reassign on merged PR"
	MsgNotAssigned = "reviewer is not assigned to this PR"
	MsgNoCandidate = "no active replacement candidate in team"
	MsgNotFound    = "resource not found"
	MsgEmptyField  = "field is empty"
)

type PRError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *PRError) Error() string {
	return e.Message
}

func NewTeamExistsError() *PRError {
	return &PRError{
		Code:    CodeTeamExists,
		Message: MsgTeamExists,
	}
}

func NewPRExistsError() *PRError {
	return &PRError{
		Code:    CodePRExists,
		Message: MsgPRExists,
	}
}

func NewPRMergedsError() *PRError {
	return &PRError{
		Code:    CodePRMerged,
		Message: MsgPRMerged,
	}
}

func NewNotAssignedError() *PRError {
	return &PRError{
		Code:    CodeNotAssigned,
		Message: MsgNotAssigned,
	}
}

func NewNoCandidateError() *PRError {
	return &PRError{
		Code:    CodeNoCandidate,
		Message: MsgNoCandidate,
	}
}

func NewNotFoundError() *PRError {
	return &PRError{
		Code:    CodeNotFound,
		Message: MsgNotFound,
	}
}

func NewEmptyFieldError(field string) *PRError {
	return &PRError{
		Code:    CodeEmptyField,
		Message: fmt.Sprintf("%s %s", field, MsgEmptyField),
	}
}
