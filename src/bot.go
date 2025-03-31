package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

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

func main() {
	// note that this is path referenced from within the docker container
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
