package main

import (
	"log"
	"os"
)

var Config *AppConfig

type AppConfig struct {
	DiscordAPIKey    string
	DiscordChannelID string
	DiscordGuildID   string
	PostgresHost     string
	PostgresPort     string
	PostgresDatabase string
	PostgresUser     string
	PostgresPassword string
}

func init() {
	Config = newAppConfig()
	log.Printf("Created global AppConfig. Config=%+v\n", Config)
}

func getOrDefault(name string, defaultValue string) string {
	value, present := os.LookupEnv(name)
	if !present {
		return defaultValue
	}
	return value
}

func newAppConfig() *AppConfig {
	config := &AppConfig{
		DiscordAPIKey:    getOrDefault("DISCORD_API_KEY", "fake-discord-api-key"),
		DiscordChannelID: getOrDefault("DISCORD_CHANNEL_ID", ""),
		DiscordGuildID:   getOrDefault("DISCORD_GUILD_ID", ""),
		PostgresHost:     getOrDefault("POSTGRES_HOST", "postgres"),
		PostgresPort:     getOrDefault("POSTGRES_PORT", "5432"),
		PostgresDatabase: getOrDefault("POSTGRES_DATABASE", "postgres"),
		PostgresUser:     getOrDefault("POSTGRES_USER", "postgres"),
		PostgresPassword: getOrDefault("POSTGRES_PASSWORD", "postgres"),
	}
	return config
}
