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

	commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate, db *bun.DB){
		"idme": getUserId,
	}
)

func SetupDiscordHandlers(db *bun.DB, dg *discordgo.Session) {
	log.Debug().Msg("Adding Bot handlers.")
	dg.AddHandler(func(session *discordgo.Session, interaction *discordgo.InteractionCreate) {
		if handler, ok := commandHandlers[interaction.ApplicationCommandData().Name]; ok {
			handler(session, interaction, db)
		}
	})

	// Start discord reaction handlers
	dg.AddHandler(func(s *discordgo.Session, r *discordgo.MessageReactionAdd) {
		setPlayDateAttendenceFromDisc(db, dg, r.MessageReaction)
	})

	_, err := dg.ApplicationCommandBulkOverwrite(dg.State.User.ID, Config.DiscordConfig.GuildID, commands)
	if err != nil {
		log.Err(err).Msg("failed to create Discord commands")
	}
}

func InitAttendanceReactions(a *Api, msg *discordgo.Message) {
	log.Info().Msg("Adding Reactions to playdate")
	addMessageReaction(a.dg, Config.DiscordConfig.ChannelID, msg.ID, "👍")
	addMessageReaction(a.dg, Config.DiscordConfig.ChannelID, msg.ID, "🤔")
	addMessageReaction(a.dg, Config.DiscordConfig.ChannelID, msg.ID, "👎")
}

func DeleteDiscordCommands(dg *discordgo.Session) error {
	log.Info().Msg("Removing Discord Commands..")
	registeredCommands, err := dg.ApplicationCommands(dg.State.User.ID, Config.DiscordConfig.GuildID)
	if err != nil {
		return err
	}
	for _, v := range registeredCommands {
		err := dg.ApplicationCommandDelete(dg.State.User.ID, Config.DiscordConfig.GuildID, v.ID)
		if err != nil {
			log.Err(err).Any("Command", v).Msg("Could not delete Discord command")
			return err
		}
	}
	log.Info().Msg("Done Removing Discord Commands!")
	return nil
}

/* Start DiscordGo Wrapper Methods */

// sendChannelMessage wraps the discordgo.ChannelMessageSend method around the Config.DiscordEnabled setting. Logs a warning if discord is not enabled.
func sendChannelMessage(dg *discordgo.Session, channelID string, msg string) (*discordgo.Message, error) {
	if !Config.DiscordEnabled {
		log.Warn().Msgf("sendChannelMessage: channelID=%s msg=%s", channelID, msg)
		return &discordgo.Message{
			ID:        "fake-message-id",
			ChannelID: "fake-channel-id",
		}, nil
	}
	return dg.ChannelMessageSend(channelID, msg)
}

// createUserChannel wraps the discordgo.UserChannelCreate method around the Config.DiscordEnabled setting. Returns a dummy discordgo.Channel if discord is not enabled.
func createUserChannel(dg *discordgo.Session, discordID string) (*discordgo.Channel, error) {
	if !Config.DiscordEnabled {
		return &discordgo.Channel{
			ID:   "fake-channel-id",
			Name: "fake-channel-name",
		}, nil
	}
	return dg.UserChannelCreate(discordID)
}

// addMessageReaction wraps the discordgo.MessageReactionAdd method around the Config.DiscordEnabled setting. Logs a warning if discord is not enabled.
func addMessageReaction(dg *discordgo.Session, channelID string, msgID string, reaction string) error {
	if !Config.DiscordEnabled {
		log.Warn().Msgf("addMessageReaction: channelID=%s msgID=%s reaction=%s", channelID, msgID, reaction)
		return nil
	}
	return dg.MessageReactionAdd(channelID, msgID, reaction)
}

/* End DiscordGo Wrapper Methods */
