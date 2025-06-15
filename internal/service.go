package internal

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/uptrace/bun"
)

func sayHello(s *discordgo.Session, i *discordgo.InteractionCreate, botContext *BotContext) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Hello <@!%s>!", botContext.player.Name),
		},
	})
}

func sayBye(s *discordgo.Session, i *discordgo.InteractionCreate, botContext *BotContext) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Goodbye <@!%s>!", botContext.player.Name),
		},
	})
}

// print out list of games available
func getGames(s *discordgo.Session, i *discordgo.InteractionCreate, botContext *BotContext) {
	// TODO: retrieve list of registered games from db
	games := []string{"No games being played"}
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: strings.Join(games, ", "),
		},
	})
}

// add user to group associated with specified game
func joinGame(s *discordgo.Session, i *discordgo.InteractionCreate, botContext *BotContext) {
	game := i.ApplicationCommandData().Options[0].Value

	// TODO: add user to group for specified game in db

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("<@!%s> is signed up for %s!", botContext.player.Name, game),
		},
	})
}

// add user to group associated with specified game
func leaveGame(s *discordgo.Session, i *discordgo.InteractionCreate, botContext *BotContext) {
	game := i.ApplicationCommandData().Options[0].Value

	// TODO: remove user from group for specified game in db

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("<@!%s> now hates anyone playing %s!", botContext.player.Name, game),
		},
	})
}

// add a game to the game list
func addGame(s *discordgo.Session, i *discordgo.InteractionCreate, botContext *BotContext) {
	game := fmt.Sprintf("%v", i.ApplicationCommandData().Options[0].Value)

	games := []string{game}
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("%s added to game list games list. games=%v", game, games),
		},
	})
}

func extractPlayerFromDiscord(i *discordgo.InteractionCreate, db *bun.DB) *Player {
	if strings.Compare(i.Member.User.ID, "") == 0 {
		log.Panicf("Received empty string for discord user id. interaction=%+v", i)
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
			log.Fatal(err)
			return nil
		}
	}
	return &player
}

func getUserId(s *discordgo.Session, i *discordgo.InteractionCreate, botContext *BotContext) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Your Discord User ID is %s", botContext.player.Name),
		},
	})
}
