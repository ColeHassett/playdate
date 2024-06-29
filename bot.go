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

	startAPI()

	// Wait for ctrl+c to close app
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
	dg.Close()

}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	fmt.Println("MESSAGE: ", m.Content)
	fmt.Println("AUTHOR: ", m.Author.ID)
	fmt.Println("CHANNEL: ", m.ChannelID)
	fmt.Println("GUILD?: ", m.GuildID)

	if strings.HasPrefix(m.Content, "!groups") {
		c, err := s.State.Channel(m.ChannelID)
		if err != nil {
			return
		}

		resp, err := http.Get("http://localhost:8080/groups")
		if err != nil {
			log.Fatalln(err)
			return
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Fatalln(err)
			return
		}

		groups := string(body)

		s.ChannelMessageSend(c.ID, groups)

	} else if strings.HasPrefix(m.Content, "!group") {
		c, err := s.State.Channel(m.ChannelID)
		if err != nil {
			return
		}

		params := strings.Fields(m.Content)
		id := params[1]

		resp, err := http.Get("http://localhost:8080/group/" + id)
		if err != nil {
			log.Fatalln(err)
			return
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Fatalln(err)
			return
		}

		group := string(body)

		s.ChannelMessageSend(c.ID, group)
	}
}
