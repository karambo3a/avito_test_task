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

func (h *Handler) SetUserIsActive(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	var request struct {
		UserID   string `json:"user_id"`
		IsActive bool   `json:"is_active"`
	}
	if err := c.BindJSON(&request); err != nil {
		log.Printf("BindJSON error: %v", err)
		c.Status(http.StatusInternalServerError)
		return
	}

	user, err := h.service.SetUserIsActive(ctx, request.UserID, request.IsActive)
	if err != nil {
		var prError *model.PRError
		if errors.As(err, &prError) {
			switch prError.Code {
			case model.CodeNotFound:
				log.Println("handler: user not found")
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

	log.Printf("user status updates: %s", request.UserID)
	c.JSON(http.StatusOK, gin.H{
		"user": user,
	})
}

func (h *Handler) GetUserReview(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	userID := c.Query("user_id")

	pullRequests, err := h.service.GetUserReview(ctx, userID)
	if err != nil {
		var prError *model.PRError
		if errors.As(err, &prError) {
			switch prError.Code {
			case model.CodeNotFound:
				log.Println("handler: user not found")
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

	log.Printf("pull requests: %v", pullRequests)
	c.JSON(http.StatusOK, gin.H{
		"user_id":       userID,
		"pull_requests": pullRequests,
	})
}
