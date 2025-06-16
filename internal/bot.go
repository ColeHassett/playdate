package internal

import (
	"github.com/bwmarrin/discordgo"
	"github.com/rs/zerolog/log"
	"github.com/uptrace/bun"
)

var (
	commands = []*discordgo.ApplicationCommand{
		// {
		// 	Name:        "hi",
		// 	Description: "Say hello!",
		// },
		// {
		// 	Name:        "bye",
		// 	Description: "Say bye!",
		// },
		// {
		// 	Name:        "games",
		// 	Description: "See list of games",
		// },
		// {
		// 	Name:        "join",
		// 	Description: "Join a game to be notified when it's being played",
		// 	Options: []*discordgo.ApplicationCommandOption{
		// 		{
		// 			Type:        discordgo.ApplicationCommandOptionString,
		// 			Name:        "name",
		// 			Description: "Name of the game to join",
		// 			Required:    true,
		// 		},
		// 	},
		// },
		// {
		// 	Name:        "leave",
		// 	Description: "Leave the list of users linked to specified game",
		// 	Options: []*discordgo.ApplicationCommandOption{
		// 		{
		// 			Type:        discordgo.ApplicationCommandOptionString,
		// 			Name:        "name",
		// 			Description: "Name of the game to leave",
		// 			Required:    true,
		// 		},
		// 	},
		// },
		// {
		// 	Name:        "add",
		// 	Description: "Add a game to the game list",
		// 	Options: []*discordgo.ApplicationCommandOption{
		// 		{
		// 			Type:        discordgo.ApplicationCommandOptionString,
		// 			Name:        "name",
		// 			Description: "Name of the game to add",
		// 			Required:    true,
		// 		},
		// 	},
		// },
		{
			Name:        "idme",
			Description: "Get your Discord User ID",
		},
	}

	commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate, botContext *BotContext){
		// "hi":    sayHello,
		// "bye":   sayBye,
		// "games": getGames,
		// "join":  joinGame,
		// "leave": leaveGame,
		// "add":   addGame,
		"idme": getUserId,
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

func deleteDiscordCommands(dg *discordgo.Session) error {
	registeredCommands, err := dg.ApplicationCommands(dg.State.User.ID, Config.DiscordGuildID)
	if err != nil {
		return err
	}
	for _, v := range registeredCommands {
		err := dg.ApplicationCommandDelete(dg.State.User.ID, Config.DiscordGuildID, v.ID)
		if err != nil {
			log.Err(err).Any("Command", v).Msg("Could not delete discord command")
			return err
		}
	}
	return nil
}

func createDiscordBot(errChan chan error, db *bun.DB) (dg *discordgo.Session) {
	log.Info().Msg("Attempting to start Discord Bot.")
	dg, err := discordgo.New("Bot " + Config.DiscordAPIKey)
	if err != nil {
		log.Err(err).Msg("failed to create discord client")
		errChan <- err
	}

	log.Info().Msg("Adding Bot handlers.")
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
	log.Info().Msg("Opening websocket to Discord...")
	err = dg.Open()
	if err != nil {
		errChan <- err
	}

	// deleteDiscordCommands(dg)
	log.Info().Msg("Creating Discord Commands...")
	_, err = createDiscordCommands(dg)
	if err != nil {
		errChan <- err
	}
	// defer dg.AddHandler(func(s *discordgo.Session, event *discordgo.Ready) {
	// 	s.ChannelMessageSend(Config.DiscordChannelID, "Nap time....😴")
	// })
	// defer dg.Close()
	// defer deleteDiscordCommands(dg)

	// Wait for ctrl+c to close app
	// log.Println("Started Discord Bot. Waiting on commands")
	// sc := make(chan os.Signal, 1)
	// signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM)
	// <-sc
	// errChan <- nil

	log.Info().Msg("Discord Bot Started")
	return dg

}

func StartDiscordBot(db *bun.DB) (dg *discordgo.Session) {
	errorChan := make(chan error)
	return createDiscordBot(errorChan, db)
	// errors := <-errorChan
	// if errors != nil {
	// 	log.Err(errors)
	// }
}

func StopDiscordBot(dg *discordgo.Session) {
	log.Info().Msg("Removing Discord Commands..")
	_, err := dg.ApplicationCommandBulkOverwrite(dg.State.User.ID, "", []*discordgo.ApplicationCommand{})
	if err != nil {
		log.Err(err).Msg("Could not delete discord commands")
	}
	log.Info().Msg("Done Removing Discord Commands!")
	dg.ChannelMessageSend(Config.DiscordChannelID, "Nap time....😴")
	dg.Close()
	log.Info().Msg("Discord Bot Successfully Shutdown")
}
