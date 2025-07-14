package internal

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/rs/zerolog/log"
	"github.com/uptrace/bun"
)

func SendPatchNotes(dg *discordgo.Session) {
	// get release notes
	release := getGithubReleaseNotes()

	log.Debug().Any("Release", release).Msg("Github Release")

	releaseBody := strings.ReplaceAll(release.Body, "@ColeHassett", "<@108736074557239296>")
	releaseBody = strings.ReplaceAll(releaseBody, "@colinthatcher", "<@128629520742744065>")

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("🤯 New PlayDate Release: %s 🤯", release.Name),
		Description: getFirstNRunes(releaseBody, 4096),
		Color:       0xfadde6,
		Timestamp:   release.PublishedAt.Format(time.RFC3339), // Discord expects ISO 8601 for timestamp
	}

	_, err := dg.ChannelMessageSendEmbed(Config.DiscordConfig.ChannelID, embed)
	if err != nil {
		log.Err(err).Msg("Failed to send patch notes embed")
	}
}

func extractPlayerFromDiscord(i *discordgo.InteractionCreate, db *bun.DB) *Player {
	if strings.Compare(i.Member.User.ID, "") == 0 {
		log.Error().Any("interaction", i).Msg("received empty string for discord user id")
	}

	// get or create new reference to the player
	var player Player
	err := db.NewSelect().Model(&player).Scan(context.Background())
	if err != nil {
		player = Player{
			Name: i.Member.User.ID,
		}
		_, err := db.NewInsert().Model(&player).Exec(context.Background())
		if err != nil {
			log.Err(err).Msg("failed to create empty player")
			return nil
		}
	}
	return &player
}

func getUserId(s *discordgo.Session, i *discordgo.InteractionCreate, db *bun.DB) {
	player := extractPlayerFromDiscord(i, db)
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Your Discord User ID is %s", player.Name),
		},
	})
}

func setPlayDateAttendenceFromDisc(db *bun.DB, dg *discordgo.Session, r *discordgo.MessageReaction) {
	ctx := context.TODO() // TODO: This should be a real context with like a 5s timeout?

	if r.UserID == "1252426978313633812" {
		log.Debug().Msg("Reaction created by bot")
		return
	}

	log.Info().Any("Reaction", r).Msg("Setting player attendance")
	discId := r.UserID
	react := r.Emoji.Name
	msg, err := dg.ChannelMessage(Config.DiscordConfig.ChannelID, r.MessageID)
	if err != nil {
		log.Err(err).Msg("failed to get reaction message")
		return
	}
	if msg.Author.ID != "1252426978313633812" {
		log.Debug().Msg("Not a bot message")
		return
	}

	attendance := AttendanceFrom(react) // parse input attendence action to internal enum
	msgSplit := strings.Split(msg.Content, "/")
	if len(msgSplit) <= 1 {
		log.Debug().Msg("Not a playdate")
		return
	}
	pId, err := strconv.Atoi(msgSplit[len(msgSplit)-1])
	if err != nil {
		log.Err(err).Msg("failed to parse given playdate id")
		return
	}

	playdate := &PlayDate{ID: pId}
	err = db.NewSelect().Model(playdate).WherePK().Scan(ctx)
	if err != nil {
		log.Err(err).Int("playdateID", pId).Msg("failed to find playdate")
		return
	}
	if playdate.Status != PlayDateStatusPending {
		log.Debug().Msg("PlayDate already happened")
		return
	}

	log.Info().Int("playdateID", playdate.ID).Str("discordId", discId).Any("action", attendance).Msg("attempting to set playdate attendance")
	player := &Player{DiscordID: discId}
	err = db.NewSelect().Model(player).Where("discord_id = ?", player.DiscordID).Scan(ctx)
	if err != nil {
		log.Err(err).Str("discID", discId).Msg("failed to find player")
		dg.ChannelMessageSend(Config.DiscordConfig.ChannelID, "Please go here to make an account: https://playdate.colinthatcher.dev/discord/login")
		err = dg.MessageReactionRemove(Config.DiscordConfig.ChannelID, msg.ID, r.Emoji.APIName(), discId)
		if err != nil {
			log.Err(err).Msg("Failed to remove reaction on anon user")
		}
		return
	}
	rel := &PlayDateToPlayer{PlayDateID: playdate.ID, PlayerID: player.ID, Attending: attendance}
	_, err = db.NewInsert().Model(rel).On("CONFLICT (playdate_id, player_id) DO UPDATE").Set("attending = EXCLUDED.attending").Exec(ctx)
	if err != nil {
		// send error back to user within the players-table.html
		log.Error().Err(err).Interface("relation", rel).Msg("failed to insert playdate to player relation")
	} else {
		log.Info().Interface("relation", rel).Msg("successfully inserted playdate to player relation")
	}

	playdatePlayers := []*PlayDateToPlayer{}
	err = db.NewSelect().Model(&playdatePlayers).Relation("Player").Where("playdate_id = ?", playdate.ID).Scan(ctx)
	if err != nil {
		// report error back to user, but just render the page like normal
		log.Err(err).Any("playdate", playdate).Msg("failed to find related players to playdate")
	}

	if r.Emoji.APIName() != "👍" {
		err = dg.MessageReactionRemove(Config.DiscordConfig.ChannelID, msg.ID, "👍", discId)
		if err != nil {
			log.Err(err).Str("Reaction", "👍").Msg("Failed to remove reaction")
		}
	}
	if r.Emoji.APIName() != "🤔" {
		err = dg.MessageReactionRemove(Config.DiscordConfig.ChannelID, msg.ID, "🤔", discId)
		if err != nil {
			log.Err(err).Str("Reaction", "🤔").Msg("Failed to remove reaction")
		}
	}
	if r.Emoji.APIName() != "👎" {
		err = dg.MessageReactionRemove(Config.DiscordConfig.ChannelID, msg.ID, "👎", discId)
		if err != nil {
			log.Err(err).Str("Reaction", "👎").Msg("Failed to remove reaction")
		}
	}
}
