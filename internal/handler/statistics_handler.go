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

func (h *Handler) GetUserStatistics(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	userID := c.Query("user_id")

	stats, err := h.service.GetUserStatistics(ctx, userID)
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
			log.Printf("handler: server error: %v", err)
			c.Status(http.StatusInternalServerError)
		}
		return
	}

	log.Printf("user statistics: %+v", stats)
	c.JSON(http.StatusOK, stats)
}

func (h *Handler) GetTeamStatistics(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	teamName := c.Query("team_name")

	stats, err := h.service.GetTeamStatistics(ctx, teamName)
	if err != nil {
		var prError *model.PRError
		if errors.As(err, &prError) {
			switch prError.Code {
			case model.CodeNotFound:
				log.Println("handler: team not found")
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
			log.Printf("handler: server error: %v", err)
			c.Status(http.StatusInternalServerError)
		}
		return
	}

	log.Printf("team statistics: %+v", stats)
	c.JSON(http.StatusOK, stats)
}
