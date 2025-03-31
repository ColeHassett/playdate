package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func sayHello(s *discordgo.Session, i *discordgo.InteractionCreate, player *Player) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Hello <@!%s>!", player.name),
		},
	})
}

func sayBye(s *discordgo.Session, i *discordgo.InteractionCreate, player *Player) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Goodbye <@!%s>!", player.name),
		},
	})
}

// print out list of games available
func getGames(s *discordgo.Session, i *discordgo.InteractionCreate, player *Player) {
	// TODO: retrieve list of registered games from db
	games := []string{"No games being played"}
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: strings.Join(games, ", "),
		},
	})
}

// add user to group associated with specified game
func joinGame(s *discordgo.Session, i *discordgo.InteractionCreate, player *Player) {
	game := i.ApplicationCommandData().Options[0].Value

	// TODO: add user to group for specified game in db

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("<@!%s> is signed up for %s!", player.name, game),
		},
	})
}

// add user to group associated with specified game
func leaveGame(s *discordgo.Session, i *discordgo.InteractionCreate, player *Player) {
	game := i.ApplicationCommandData().Options[0].Value

	// TODO: remove user from group for specified game in db

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("<@!%s> now hates anyone playing %s!", player.name, game),
		},
	})
}

// add a game to the game list
func addGame(s *discordgo.Session, i *discordgo.InteractionCreate, player *Player) {
	game := fmt.Sprintf("%v", i.ApplicationCommandData().Options[0].Value)

	games := []string{game}
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("%s added to game list games list. games=%v", game, games),
		},
	})
}

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
