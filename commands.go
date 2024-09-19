package main

import (
	"fmt"

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
	}

	commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"hi":    sayHello,
		"bye":   sayBye,
		"games": getGames,
		"join":  joinGame,
		"leave": leaveGame,
	}
)

func sayHello(s *discordgo.Session, i *discordgo.InteractionCreate) {
	author := fmt.Sprintf("%v", i.Interaction.Member.User.ID)
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Hello <@!%s>!", author),
		},
	})
}

func sayBye(s *discordgo.Session, i *discordgo.InteractionCreate) {
	author := fmt.Sprintf("%v", i.Interaction.Member.User.ID)
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Goodbye <@!%s>!", author),
		},
	})
}

// print out list of games available
func getGames(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// TODO: retrieve list of registered games from db
}

// add user to group associated with specified game
func joinGame(s *discordgo.Session, i *discordgo.InteractionCreate) {
	authorId := fmt.Sprintf("%v", i.Interaction.Member.User.ID)
	game := i.ApplicationCommandData().Options[0].Value

	// TODO: add user to group for specified game in db

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("<@!%s> is signed up for %s!", authorId, game),
		},
	})
}

// add user to group associated with specified game
func leaveGame(s *discordgo.Session, i *discordgo.InteractionCreate) {
	authorId := fmt.Sprintf("%v", i.Interaction.Member.User.ID)
	game := i.ApplicationCommandData().Options[0].Value

	// TODO: remove user from group for specified game in db

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("<@!%s> now hates anyone playing %s!", authorId, game),
		},
	})
}
