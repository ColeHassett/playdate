package main

import (
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func extractPlayerFromDiscord(i *discordgo.InteractionCreate) *Player {
	if &i == nil || &i.Member == nil || &i.Member.User == nil {
		log.Panicf("Received invalid discord interaction. interaction=%+v", i)
	}
	if strings.Compare(i.Member.User.ID, "") == 0 {
		log.Panicf("Received empty string for discord user id. interaction=%+v", i)
	}

	// get or create new reference to the player
	// TODO: Attempt to find given player in the database or save the new player to the database
	player := Player{
		name: i.Member.User.ID,
	}
	return &player
}
