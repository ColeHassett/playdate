package main

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/uptrace/bun"
)

func startAPI(db *bun.DB) {
	api := Api{db: db, ctx: context.Background()}

	router := gin.Default()
	router.GET("/player", api.getPlayers)
	router.POST("/player", api.createPlayer)
	router.Run("localhost:8080")
}

type Api struct {
	db  *bun.DB
	ctx context.Context
}

func (a *Api) getPlayers(c *gin.Context) {
	var players []Player
	err := a.db.NewSelect().Model(&players).Scan(a.ctx)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": players,
	})
}

func (a *Api) createPlayer(c *gin.Context) {
	var json Player
	err := c.ShouldBindJSON(&json)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}
	_, err = a.db.NewInsert().Model(&json).Exec(a.ctx)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}
	c.JSON(http.StatusCreated, gin.H{})
}
