package main

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"text/template"

	"github.com/gin-gonic/gin"
	"github.com/uptrace/bun"
)

type Todo struct {
	ID   int
	Text string
}

// Global variable to simulate a database for simplicity
var (
	todos    = []Todo{}
	nextID   = 1
	todosMux = sync.Mutex{} // Mutex for concurrent access to todos slice
)

func startAPI(db *bun.DB) {
	api := Api{db: db, ctx: context.Background()}

	router := gin.Default()

	// Template Endpoints
	router.LoadHTMLGlob("templates/*")
	// Render the initial page
	router.GET("/", func(c *gin.Context) {
		todosMux.Lock()
		defer todosMux.Unlock()
		c.HTML(http.StatusOK, "index.html", gin.H{
			"Todos": todos,
		})
	})

	// Handle adding a new todo (HTMX POST request)
	router.POST("/todos", func(c *gin.Context) {
		newTodoText := c.PostForm("newTodo")
		if newTodoText == "" {
			c.Status(http.StatusBadRequest)
			return
		}

		todosMux.Lock()
		newTodo := Todo{ID: nextID, Text: newTodoText}
		todos = append(todos, newTodo)
		nextID++
		todosMux.Unlock()

		// Render only the new todo item as a partial HTML response
		// This HTML will be inserted by HTMX into the #todo-list
		tmpl := template.Must(template.ParseFiles("templates/todo-item.html"))
		err := tmpl.ExecuteTemplate(c.Writer, "todo-item.html", newTodo)
		if err != nil {
			fmt.Printf("Error rendering todo-item: %v\n", err)
			c.Status(http.StatusInternalServerError)
			return
		}
	})

	// Handle deleting a todo (HTMX DELETE request)
	router.DELETE("/todos/:id", func(c *gin.Context) {
		idParam := c.Param("id")
		id, err := strconv.Atoi(idParam)
		if err != nil {
			c.Status(http.StatusBadRequest)
			return
		}

		todosMux.Lock()
		defer todosMux.Unlock()

		found := false
		for i, todo := range todos {
			if todo.ID == id {
				todos = append(todos[:i], todos[i+1:]...) // Remove the todo
				found = true
				break
			}
		}

		if !found {
			c.Status(http.StatusNotFound)
			return
		}

		// HTMX expects an empty response (or a specific HTML if you want to replace it with something else)
		// for `outerHTML` swap to remove the element.
		c.Status(http.StatusOK) // Or http.StatusNoContent (204)
	})

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
