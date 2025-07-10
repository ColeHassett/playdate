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
	a.dg.MessageReactionAdd(Config.DiscordConfig.ChannelID, msg.ID, "👍")
	a.dg.MessageReactionAdd(Config.DiscordConfig.ChannelID, msg.ID, "🤔")
	a.dg.MessageReactionAdd(Config.DiscordConfig.ChannelID, msg.ID, "👎")
}
