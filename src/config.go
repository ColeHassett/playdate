package main

import (
	"log"
	"os"
)

var Config AppConfig

type AppConfig struct {
	DiscordAPIKey    string
	DiscordChannelID string
	DiscordGuildID   string
}

func init() {
	Config = newAppConfig()
	log.Printf("Created global AppConfig. Config=%+v\n", Config)
}

func newAppConfig() AppConfig {
	discordAPIKey, present := os.LookupEnv("DISCORD_API_KEY")
	if !present {
		discordAPIKey = token
	}
	discordChannelID, present := os.LookupEnv("DISCORD_CHANNEL_ID")
	if !present {
		discordChannelID = ""
	}
	discordGuildID, present := os.LookupEnv("DISCORD_GUILD_ID")
	if !present {
		discordGuildID = ""
	}
	return AppConfig{
		DiscordAPIKey:    discordAPIKey,
		DiscordChannelID: discordChannelID,
		DiscordGuildID:   discordGuildID,
	}
}
