package main

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

type group struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Members []string `json:"members"`
	Games   []string `json:"games"`
}

var groups = []group{
	{ID: "1", Name: "Group 1", Members: []string{"Member A", "Member B", "Member C"}, Games: []string{"Fortnite", "Roblox", "Elden Ring", "Cornhole"}},
}

func startAPI() {
	router := gin.Default()
	router.GET("/groups", getGroups)
	router.GET("/group/:id", getGroupById)

	router.Run("localhost:8080")
}

func getGroups(c *gin.Context) {
	c.IndentedJSON(http.StatusOK, groups)
}

func getGroupById(c *gin.Context) {
	id := c.Param("id")

	for _, g := range groups {
		if g.ID == id {
			c.IndentedJSON(http.StatusOK, g)
			return
		}
	}
	c.IndentedJSON(http.StatusNotFound, gin.H{"message": fmt.Sprintf("Group with ID %s not found", id)})
}
