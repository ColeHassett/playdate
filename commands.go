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
	}

	commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"hi": sayHello,
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
