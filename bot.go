package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
	// "github.com/go-co-op/gocron/v2"
)

var (
	BotToken       = flag.String("token", "", "Bot Token")
	GeneralChannel = "522472817283956745"
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

	dg.AddHandler(func(s *discordgo.Session, event *discordgo.Ready) {
		s.ChannelMessageSend(GeneralChannel, "FARTS WE GO IN")
	})
	dg.AddHandler(messageCreate)

	// Receive message events
	dg.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildMessages

	// Open websocket connection to discord
	err = dg.Open()
	if err != nil {
		fmt.Println("YOU RUINED IT: ", err)
		return
	}

	// Wait for ctrl+c to close app
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
	dg.Close()

}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Content == "hi" {
		s.ChannelMessageSend("522472817283956745", "Hello World")
	}
	fmt.Println("MESSAGE: ", m.Content)
	fmt.Println("AUTHOR: ", m.Author.ID)
	fmt.Println("CHANNEL: ", m.ChannelID)
	fmt.Println("GUILD?: ", m.GuildID)
}
