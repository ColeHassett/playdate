package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
	// "github.com/go-co-op/gocron/v2"
)

var (
	BotToken         = flag.String("token", "", "Bot Token")
	GeneralChannelID = "522472817283956745"
	GuildID          = "522472816818520106"
)

func init() { flag.Parse() }

func main() {

	if *BotToken == "" {
		fmt.Println("Not token specified")
		return
	}

	dg, err := discordgo.New("Bot " + *BotToken)
	if err != nil {
		fmt.Println("YOU RUINED IT: ", err)
		return
	}

	dg.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if h, ok := commandHandlers[i.ApplicationCommandData().Name]; ok {
			h(s, i)
		}
	})

	dg.AddHandler(func(s *discordgo.Session, event *discordgo.Ready) {
		s.ChannelMessageSend(GeneralChannelID, "FARTS WE GO IN")
	})

	// Open websocket connection to discord
	err = dg.Open()
	if err != nil {
		log.Fatalf("Failed to open connection: %v", err)
		return
	}

	registeredCommands := make([]*discordgo.ApplicationCommand, len(commands))
	for i, v := range commands {
		cmd, err := dg.ApplicationCommandCreate(dg.State.User.ID, GuildID, v)
		if err != nil {
			log.Panicf("Cannot create [%v]: %v", v.Name, err)
		}
		registeredCommands[i] = cmd
	}

	// Wait for ctrl+c to close app
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	for _, v := range registeredCommands {
		err := dg.ApplicationCommandDelete(dg.State.User.ID, GuildID, v.ID)
		if err != nil {
			log.Panicf("Cannot delete [%v]: %v", v.Name, err)
		}
	}

	dg.Close()

}
