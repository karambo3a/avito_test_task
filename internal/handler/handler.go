package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/karambo3a/avito_test_task/internal/service"
)

type Handler struct {
	service *service.Service
}

func NewHandler(s *service.Service) *Handler {
	return &Handler{service: s}
}

func (h *Handler) InitRoutes() *gin.Engine {
	router := gin.Default()

	teamGroup := router.Group("/team")
	{
		teamGroup.POST("/add", h.AddTeam)
		teamGroup.GET("/get", h.GetTeam)
	}

	usersGroup := router.Group("/users")
	{
		usersGroup.POST("/setIsActive", h.SetUserIsActive)
		usersGroup.GET("/getReview", h.GetUserReview)
	}

	prGroup := router.Group("/pullRequest")
	{
		prGroup.POST("/create", h.CreatePR)
		prGroup.POST("/merge", h.MergePR)
		prGroup.POST("/reassign", h.ReassignPR)
	}

	statsGroup := router.Group("/statistics")
	{
		statsGroup.GET("/user", h.GetUserStatistics)
		statsGroup.GET("/team", h.GetTeamStatistics)
	}

	return router
}
