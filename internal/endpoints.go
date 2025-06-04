package internal

import (
	"context"
	"encoding/json"
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
	router.StaticFile("custom-colors.css", fmt.Sprintf("%s/custom-colors.css", Config.TemplateDirectory))
	// Templated endpoints
	router.GET("/", api.index)
	router.GET("/home", api.home)
	router.POST("/register", api.registerUserTemplate)
	router.GET("/playdate", api.showPlayDateForm)
	router.POST("/playdate", api.createPlayDateTemplate)

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

	router.Run()
}

func renderPartialHtml(c *gin.Context, filepath string, state any) {
	tmpl := template.Must(template.ParseFiles(fmt.Sprintf("%s/%s", Config.TemplateDirectory, filepath)))
	err := tmpl.ExecuteTemplate(c.Writer, filepath, state)
	if err != nil {
		fmt.Printf("Error rendering %s: %v\n", filepath, err)
		c.Status(http.StatusInternalServerError)
		return
	}
}

type Api struct {
	db  *bun.DB
	ctx context.Context
}

func (a *Api) index(c *gin.Context) {
	cookie, _ := c.Cookie("playdate")
	type IndexState struct {
		SignedUp  bool
		PlayDates []*PlayDate
	}
	state := &IndexState{}
	if cookie == "" {
		c.HTML(http.StatusOK, "index.html", state)
		return
	}
	// extra processing
	state.SignedUp = true
	state.PlayDates = []*PlayDate{}
	err := a.db.NewSelect().Model(&state.PlayDates).Scan(a.ctx)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}
	log.Println(state)

	c.HTML(http.StatusOK, "index.html", state)
}

func (a *Api) showPlayDateForm(c *gin.Context) {
	c.HTML(http.StatusOK, "playdate-form.html", gin.H{})
}

func (a *Api) home(c *gin.Context) {
	type HomeState struct {
		Playdates []PlayDate
	}

	state := &HomeState{Playdates: []PlayDate{}}
	err := a.db.NewSelect().Model(&state.Playdates).Scan(a.ctx)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	renderPartialHtml(c, "home.html", state)
}

func (a *Api) createPlayDateTemplate(c *gin.Context) {
	inputGame := c.PostForm("game")
	inputDatetime := c.PostForm("date")

	type FormData struct {
		Game   string
		Date   string
		Errors map[string]string
	}
	formData := &FormData{Game: inputGame, Date: inputDatetime, Errors: make(map[string]string)}
	if inputGame == "" {
		formData.Errors["game"] = "game is required"
	}
	if inputDatetime == "" {
		formData.Errors["date"] = "date is required"
	}
	// NOTE: that this date is not random but instead hardcoded into the standard
	// libary to code layouts against.
	expectedTimeLayout := "2006-01-02T15:04"
	parsedDatetime, err := time.Parse(expectedTimeLayout, inputDatetime)
	if err != nil {
		formData.Errors["date"] = "invalid format for date/time, please use layout 2025-01-01T12:00"
	}
	if len(formData.Errors) > 0 {
		renderPartialHtml(c, "playdate-form.html", formData)
		return
	}

	playdate := PlayDate{Game: inputGame, Date: parsedDatetime}
	_, err = a.db.NewInsert().Model(&playdate).Exec(a.ctx)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.Header("HX-Location", "/")
}

func (a *Api) registerUserTemplate(c *gin.Context) {
	name := c.PostForm("name")
	discID := c.PostForm("discID")

	type FormData struct {
		Name   string
		DiscID string
		Errors map[string]string
	}
	formData := &FormData{Name: name, DiscID: discID, Errors: make(map[string]string)}
	if name == "" {
		formData.Errors["name"] = "name is required"
	}
	if discID == "" {
		formData.Errors["discID"] = "discID is required"
	}
	if len(formData.Errors) > 0 {
		renderPartialHtml(c, "register.html", formData)
		return
	}

	player := Player{Name: name, DiscordID: discID}
	err := a.db.NewSelect().Model(&player).Where("discord_id = ?", player.DiscordID).Scan(a.ctx)
	if err != nil {
		_, err := a.db.NewInsert().Model(&player).Exec(a.ctx)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}
	}

	val, _ := json.Marshal(player)
	c.SetCookie("playdate", string(val), 0, "/", "", false, true)
	c.Status(http.StatusCreated)
	c.Header("HX-Location", "/")
	// renderPartialHtml(c, "home.html", nil)
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
