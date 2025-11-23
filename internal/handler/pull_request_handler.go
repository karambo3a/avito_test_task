package handler

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/karambo3a/avito_test_task/internal/model"
)

func (h *Handler) CreatePR(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	var request struct {
		PullRequestID   string `json:"pull_request_id"`
		PullRequestName string `json:"pull_request_name"`
		AuthorID        string `json:"author_id"`
	}

	if err := c.BindJSON(&request); err != nil {
		log.Printf("BindJSON error: %v", err)
		c.Status(http.StatusInternalServerError)
		return
	}

	pr, err := h.service.CreatePR(ctx, request.PullRequestID, request.PullRequestName, request.AuthorID)
	if err != nil {
		var prError *model.PRError
		if errors.As(err, &prError) {
			switch prError.Code {
			case model.CodeNotFound:
				log.Println("handler: author not found")
				c.JSON(http.StatusNotFound, gin.H{
					"error": err,
				})
			case model.CodePRExists:
				log.Println("handler: pr already exists")
				c.JSON(http.StatusConflict, gin.H{
					"error": err,
				})
			case model.CodeEmptyField:
				log.Println("handler: empty field")
				c.JSON(http.StatusBadRequest, gin.H{
					"error": err,
				})
			}
		} else {
			log.Println("handler: server error")
			c.Status(http.StatusInternalServerError)
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"pr": map[string]interface{}{
			"pull_request_id":    pr.PullRequestID,
			"pull_request_name":  pr.PullRequestName,
			"author_id":          pr.AuthorID,
			"status":             pr.Status,
			"assigned_reviewers": pr.AssignedReviewers,
		},
	})
}

func (h *Handler) MergePR(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	var req struct {
		PullRequestID string `json:"pull_request_id"`
	}
	if err := c.BindJSON(&req); err != nil {
		log.Printf("BindJSON error: %v", err)
		c.Status(http.StatusInternalServerError)
		return
	}

	pr, err := h.service.MergePR(ctx, req.PullRequestID)
	if err != nil {
		var prError *model.PRError
		if errors.As(err, &prError) {
			switch prError.Code {
			case model.CodeNotFound:
				log.Println("handler: pr not found")
				c.JSON(http.StatusNotFound, gin.H{
					"error": err,
				})
			case model.CodeEmptyField:
				log.Println("handler: empty field")
				c.JSON(http.StatusBadRequest, gin.H{
					"error": err,
				})
			}
		} else {
			log.Println("handler: server error")
			c.Status(http.StatusInternalServerError)
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"pr": map[string]interface{}{
			"pull_request_id":    pr.PullRequestID,
			"pull_request_name":  pr.PullRequestName,
			"author_id":          pr.AuthorID,
			"status":             pr.Status,
			"assigned_reviewers": pr.AssignedReviewers,
			"mergedAt":           pr.MergedAt,
		},
	})
}

func (h *Handler) ReassignPR(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	var req struct {
		PullRequestID string `json:"pull_request_id"`
		OldUserID     string `json:"old_user_id"`
	}
	if err := c.BindJSON(&req); err != nil {
		log.Printf("BindJSON error: %v", err)
		c.Status(http.StatusInternalServerError)
		return
	}

	pr, replacedBy, err := h.service.ReassignPR(ctx, req.PullRequestID, req.OldUserID)
	if err != nil {
		var prError *model.PRError
		if errors.As(err, &prError) {
			switch prError.Code {
			case model.CodeNotFound:
				log.Println("handler: pr not found")
				c.JSON(http.StatusNotFound, gin.H{
					"error": err,
				})
			case model.CodePRMerged:
				log.Println("handler: pr merged")
				c.JSON(http.StatusConflict, gin.H{
					"error": err,
				})
			case model.CodeNotAssigned:
				log.Println("handler: user not assigned")
				c.JSON(http.StatusConflict, gin.H{
					"error": err,
				})
			case model.CodeNoCandidate:
				log.Println("handler: no active candidates for reviewers")
				c.JSON(http.StatusConflict, gin.H{
					"error": err,
				})
			case model.CodeEmptyField:
				log.Println("handler: empty field")
				c.JSON(http.StatusBadRequest, gin.H{
					"error": err,
				})
			}
		} else {
			log.Println("handler: server error")
			c.Status(http.StatusInternalServerError)
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"pr": map[string]any{
			"pull_request_id":    pr.PullRequestID,
			"pull_request_name":  pr.PullRequestName,
			"author_id":          pr.AuthorID,
			"status":             pr.Status,
			"assigned_reviewers": pr.AssignedReviewers,
		},
		"replaced_by": replacedBy,
	})
}
