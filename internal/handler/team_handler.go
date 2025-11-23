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

func (h *Handler) AddTeam(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	var reqBody model.Team
	if err := c.BindJSON(&reqBody); err != nil {
		log.Printf("BindJSON error: %v", err)
		c.Status(http.StatusInternalServerError)
		return
	}

	log.Printf("Received team: %s with %d members", reqBody.TeamName, len(reqBody.Members))

	team, err := h.service.AddTeam(ctx, reqBody)
	if err != nil {
		var prError *model.PRError
		if errors.As(err, &prError) {
			switch prError.Code {
			case model.CodeTeamExists:
				log.Println("handler: team exists")
				c.JSON(http.StatusBadRequest, gin.H{
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

	log.Printf("team saved: %+v", team)
	c.JSON(http.StatusCreated, gin.H{
		"team": team,
	})
}

func (h *Handler) GetTeam(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	teamName := c.Query("team_name")

	team, err := h.service.GetTeam(ctx, teamName)
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
			log.Println("handler: server error")
			c.Status(http.StatusInternalServerError)
		}
		return
	}

	log.Printf("team: %+v", team)
	c.JSON(http.StatusOK, team)
}
