package internal

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"
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

	commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate, botContext *BotContext){
		"hi":    sayHello,
		"bye":   sayBye,
		"games": getGames,
		"join":  joinGame,
		"leave": leaveGame,
		"add":   addGame,
	}
)

type BotContext struct {
	player *Player
	db     *bun.DB
}

func createDiscordCommands(dg *discordgo.Session) ([]*discordgo.ApplicationCommand, error) {
	registeredCommands := make([]*discordgo.ApplicationCommand, len(commands))
	for i, v := range commands {
		cmd, err := dg.ApplicationCommandCreate(dg.State.User.ID, Config.DiscordGuildID, v)
		if err != nil {
			return nil, err
		}
		registeredCommands[i] = cmd
	}
	return registeredCommands, nil
}

func deleteDiscordCommands(dg *discordgo.Session, registeredCommands []*discordgo.ApplicationCommand) error {
	for _, v := range registeredCommands {
		err := dg.ApplicationCommandDelete(dg.State.User.ID, Config.DiscordGuildID, v.ID)
		if err != nil {
			return err
		}
	}
	return nil
}

func createDiscordBot(errChan chan error, db *bun.DB) {
	log.Println("Attempting to start Discord Bot.")
	dg, err := discordgo.New("Bot " + Config.DiscordAPIKey)
	if err != nil {
		errChan <- err
	}

	log.Println("Adding Bot handlers.")
	dg.AddHandler(func(session *discordgo.Session, interaction *discordgo.InteractionCreate) {
		if handler, ok := commandHandlers[interaction.ApplicationCommandData().Name]; ok {
			botContext := &BotContext{
				player: extractPlayerFromDiscord(interaction, db),
				db:     db,
			}
			handler(session, interaction, botContext)
		}
	})

	dg.AddHandler(func(s *discordgo.Session, event *discordgo.Ready) {
		s.ChannelMessageSend(Config.DiscordChannelID, "Let's Play!")
	})

	// Open websocket connection to discord
	log.Println("Opening websocket to Discord...")
	err = dg.Open()
	if err != nil {
		errChan <- err
	}
	commands, err := createDiscordCommands(dg)
	if err != nil {
		errChan <- err
	}
	defer dg.Close()
	defer deleteDiscordCommands(dg, commands)

	// Wait for ctrl+c to close app
	log.Println("Started Discord Bot. Waiting on commands")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	errChan <- nil
}

func startDiscordBot(db *bun.DB) {
	errorChan := make(chan error)
	go createDiscordBot(errorChan, db)
	errors := <-errorChan
	if errors != nil {
		log.Fatal(errors)
	}
}
