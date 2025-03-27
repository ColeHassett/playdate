package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

func main() {
	banner, err := os.ReadFile("banner.txt")
	if err != nil {
		log.Fatal("Failed to read banner text file. err=", err)
	}
	log.Println(string(banner))

	dg, err := discordgo.New("Bot " + Config.DiscordAPIKey)
	if err != nil {
		log.Panic("YOU RUINED IT: ", err)
		return
	}

	dg.AddHandler(func(session *discordgo.Session, interaction *discordgo.InteractionCreate) {
		if handler, ok := commandHandlers[interaction.ApplicationCommandData().Name]; ok {
			player := extractPlayerFromDiscord(interaction)
			handler(session, interaction, player)
		}
	})

	dg.AddHandler(func(s *discordgo.Session, event *discordgo.Ready) {
		s.ChannelMessageSend(Config.DiscordChannelID, "Let's Play!")
	})

	// Open websocket connection to discord
	err = dg.Open()
	if err != nil {
		log.Fatalf("Failed to open connection: %v", err)
		return
	}

	registeredCommands := make([]*discordgo.ApplicationCommand, len(commands))
	for i, v := range commands {
		cmd, err := dg.ApplicationCommandCreate(dg.State.User.ID, Config.DiscordGuildID, v)
		if err != nil {
			log.Panicf("Cannot create [%v]: %v", v.Name, err)
		}
		registeredCommands[i] = cmd
	}

	// Wait for ctrl+c to close app
	log.Println("Bot successfully started. Waiting for commands...")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	for _, v := range registeredCommands {
		err := dg.ApplicationCommandDelete(dg.State.User.ID, Config.DiscordGuildID, v.ID)
		if err != nil {
			log.Panicf("Cannot delete [%v]: %v", v.Name, err)
		}
	}

	dg.Close()
}
