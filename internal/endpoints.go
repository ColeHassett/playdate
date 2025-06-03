package internal

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"text/template"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/uptrace/bun"
)

func StartAPI(db *bun.DB) {
	api := Api{db: db, ctx: context.Background()}

	router := gin.Default()

	// Template Endpoints
	router.LoadHTMLGlob(fmt.Sprintf("%s/*", Config.TemplateDirectory))
	// router.Static("static/", "/app/static/")
	router.StaticFile("service-worker.js", "./static/service-worker.js")
	// Render the initial page
	router.GET("/", api.index)
	router.POST("/player", api.createPlayersTemplate)
	router.POST("/playdate", api.createPlayDateTemplate)

	// Browser Notifications
	// TODO: This would live in their own package for sure
	router.GET("/get-keys", GenerateKeys)
	router.POST("/subscribe", SubscribeHandler)
	// router.GET("/send-notification", TestPush)
	router.GET("/send-notification", SendNotificationHandler)

	// API Endpoints
	v1 := router.Group("/api/v1")
	// /player
	v1.GET("/player", api.getPlayers)
	v1.POST("/player", api.createPlayer)
	v1.GET("/player/:id", api.getPlayer)
	v1.PATCH("/player/:id", api.updatePlayer)
	v1.DELETE("/player/:id", api.deletePlayer)
	// /playdate
	v1.GET("/playdate", api.getPlayDates)
	v1.POST("/playdate", api.createPlayDate)
	v1.GET("/playdate/:id", api.getPlayDate)
	v1.PATCH("/playdate/:id", api.updatePlayDate)
	v1.DELETE("/playdate/:id", api.deletePlayDate)

	router.Run("0.0.0.0:8080")
}

type Api struct {
	db  *bun.DB
	ctx context.Context
}

func (a *Api) index(c *gin.Context) {
	// retrieve players for initial rendering
	var players []Player
	err := a.db.NewSelect().Model(&players).Scan(a.ctx)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}
	var playdates []PlayDate
	err = a.db.NewSelect().Model(&playdates).Scan(a.ctx)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.HTML(http.StatusOK, "index.html", gin.H{
		"Players":   players,
		"PlayDates": playdates,
	})
}

func (a *Api) createPlayersTemplate(c *gin.Context) {
	playerName := c.PostForm("name")
	if playerName == "" {
		c.Status(http.StatusBadRequest)
		return
	}

	player := Player{Name: playerName}
	_, err := a.db.NewInsert().Model(&player).Exec(a.ctx)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	// Render only the new todo item as a partial HTML response
	// This HTML will be inserted by HTMX into the #todo-list
	tmpl := template.Must(template.ParseFiles(fmt.Sprintf("%s/player-item.html", Config.TemplateDirectory)))
	err = tmpl.ExecuteTemplate(c.Writer, "player-item.html", player)
	if err != nil {
		fmt.Printf("Error rendering player-item: %v\n", err)
		c.Status(http.StatusInternalServerError)
		return
	}
}

func (a *Api) createPlayDateTemplate(c *gin.Context) {
	inputGame := c.PostForm("game")
	inputDatetime := c.PostForm("datetime")
	log.Default().Println(inputGame, inputDatetime)
	if inputGame == "" {
		c.Status(http.StatusBadRequest)
		return
	}
	if inputDatetime == "" {
		c.Status(http.StatusBadRequest)
		return
	}

	// NOTE: that this date is not random but instead hardcoded into the standard
	// libary to code layouts against.
	expectedTimeLayout := "2006-01-02T15:04"
	parsedDatetime, err := time.Parse(expectedTimeLayout, inputDatetime)
	if err != nil {
		fmt.Print(err)
		c.Status(http.StatusBadRequest)
		return
	}
	log.Default().Println(parsedDatetime)

	playdate := PlayDate{Game: inputGame, Date: parsedDatetime}
	_, err = a.db.NewInsert().Model(&playdate).Exec(a.ctx)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	// Render only the new todo item as a partial HTML response
	// This HTML will be inserted by HTMX into the #todo-list
	tmpl := template.Must(template.ParseFiles(fmt.Sprintf("%s/playdate-item.html", Config.TemplateDirectory)))
	err = tmpl.ExecuteTemplate(c.Writer, "playdate-item.html", playdate)
	if err != nil {
		fmt.Printf("Error rendering playdate-item: %v\n", err)
		c.Status(http.StatusInternalServerError)
		return
	}
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
	c.JSON(http.StatusOK, players)
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

func (a *Api) getPlayer(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing required parameter id"})
		return
	}
	var player Player
	err := a.db.NewSelect().Model(&player).Scan(a.ctx)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, player)
}

func (a *Api) updatePlayer(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing required parameter id"})
		return
	}
	var json Player
	err := c.ShouldBindJSON(&json)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}
	var player Player
	err = a.db.NewSelect().Model(&player).Scan(a.ctx)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}
	if json.Name != "" {
		player.Name = json.Name
	}
	_, err = a.db.NewUpdate().Model(&player).WherePK().Exec(a.ctx)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, player)
}

func (a *Api) deletePlayer(c *gin.Context) {
	inputId := c.Param("id")
	if inputId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing required parameter id"})
		return
	}
	id, err := strconv.Atoi(inputId)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id given"})
		return
	}
	player := Player{ID: id}
	_, err = a.db.NewDelete().Model(&player).WherePK().Exec(a.ctx)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}
	c.JSON(http.StatusNoContent, gin.H{})
}

func (a *Api) getPlayDates(c *gin.Context) {
	var playdates []PlayDate
	err := a.db.NewSelect().Model(&playdates).Scan(a.ctx)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, playdates)
}

func (a *Api) createPlayDate(c *gin.Context) {
	var json PlayDate
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

func (a *Api) getPlayDate(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing required parameter id"})
		return
	}
	var playdate PlayDate
	err := a.db.NewSelect().Model(&playdate).Scan(a.ctx)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, playdate)
}

func (a *Api) updatePlayDate(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing required parameter id"})
		return
	}
	var json PlayDate
	err := c.ShouldBindJSON(&json)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}
	var playdate PlayDate
	err = a.db.NewSelect().Model(&playdate).Scan(a.ctx)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}
	if json.Game != "" {
		playdate.Game = json.Game
	}
	if !json.Date.IsZero() {
		playdate.Date = json.Date
	}
	_, err = a.db.NewUpdate().Model(&playdate).WherePK().Exec(a.ctx)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, playdate)
}

func (a *Api) deletePlayDate(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing required parameter id"})
		return
	}
	_, err := a.db.NewDelete().Where("id = ?", id).Exec(a.ctx)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}
	c.JSON(http.StatusNoContent, gin.H{})
}
