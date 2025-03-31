package main

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

var (
	commands = []*discordgo.ApplicationCommand{
		{
			Name:        "hi",
			Description: "Say hello!",
		},
		{
			Name:        "bye",
			Description: "Say bye!",
		},
		{
			Name:        "games",
			Description: "See list of games",
		},
		{
			Name:        "join",
			Description: "Join a game to be notified when it's being played",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "name",
					Description: "Name of the game to join",
					Required:    true,
				},
			},
		},
		{
			Name:        "leave",
			Description: "Leave the list of users linked to specified game",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "name",
					Description: "Name of the game to leave",
					Required:    true,
				},
			},
		},
		{
			Name:        "add",
			Description: "Add a game to the game list",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "name",
					Description: "Name of the game to add",
					Required:    true,
				},
			},
		},
	}

	commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate, player *Player){
		"hi":    sayHello,
		"bye":   sayBye,
		"games": getGames,
		"join":  joinGame,
		"leave": leaveGame,
		"add":   addGame,
	}
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
