package internal

import (
	"github.com/bwmarrin/discordgo"
	"github.com/rs/zerolog/log"
	"github.com/uptrace/bun"
)

var (
	commands = []*discordgo.ApplicationCommand{
		{
			Name:        "idme",
			Description: "Get your Discord User ID",
		},
	}

	commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate, botContext *BotContext){
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
	log.Info().Msg("Removing Discord Commands..")
	registeredCommands, err := dg.ApplicationCommands(dg.State.User.ID, Config.DiscordGuildID)
	if err != nil {
		return err
	}
	for _, v := range registeredCommands {
		err := dg.ApplicationCommandDelete(dg.State.User.ID, Config.DiscordGuildID, v.ID)
		if err != nil {
			log.Err(err).Any("Command", v).Msg("Could not delete Discord command")
			return err
		}
	}
	log.Info().Msg("Done Removing Discord Commands!")
	return nil
}

func InitAttendanceReactions(a *Api, msg *discordgo.Message) {
	log.Info().Msg("Adding Reactions to playdate")
	a.dg.MessageReactionAdd(Config.DiscordChannelID, msg.ID, "👍")
	a.dg.MessageReactionAdd(Config.DiscordChannelID, msg.ID, "🤔")
	a.dg.MessageReactionAdd(Config.DiscordChannelID, msg.ID, "👎")
}

func handleReactionAdd(dg *discordgo.Session, event *discordgo.MessageReactionAdd) {

}

func handleReactionRemove(dg *discordgo.Session, event *discordgo.MessageReactionRemove) {

}

func createDiscordBot(db *bun.DB) (dg *discordgo.Session) {
	log.Info().Msg("Attempting to start Discord Bot.")
	dg, err := discordgo.New("Bot " + Config.DiscordAPIKey)
	if err != nil {
		log.Err(err).Msg("failed to create Discord client")
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
	dg.AddHandler(func(s *discordgo.Session, event *discordgo.MessageReactionAdd) {
		handleReactionAdd(s, event)
	})
	dg.AddHandler(func(s *discordgo.Session, event *discordgo.MessageReactionRemove) {
		handleReactionRemove(s, event)
	})
	dg.AddHandler(func(s *discordgo.Session, event *discordgo.Ready) {
		s.ChannelMessageSend(Config.DiscordChannelID, "Let's Play!")
	})

	// Open websocket connection to discord
	log.Info().Msg("Opening websocket to Discord...")
	err = dg.Open()
	if err != nil {
		log.Err(err).Msg("failed to open websocket to Discord")
	}

	log.Info().Msg("Creating Discord Commands...")
	_, err = createDiscordCommands(dg)
	if err != nil {
		log.Err(err).Msg("failed to create Discord commands")
	}

	log.Info().Msg("Discord Bot Started")
	return dg

}

func StartDiscordBot(db *bun.DB) (dg *discordgo.Session) {
	return createDiscordBot(db)
}

func StopDiscordBot(dg *discordgo.Session) {
	deleteDiscordCommands(dg)
	dg.ChannelMessageSend(Config.DiscordChannelID, "Nap time....😴")
	dg.Close()
	log.Info().Msg("Discord Bot Successfully Shutdown")
}
