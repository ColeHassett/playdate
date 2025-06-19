package internal

import (
	"os"

	"github.com/rs/zerolog/log"
)

var Config *AppConfig

type DiscordConfig struct {
	APIKey    string
	ChannelID string
	GuildID   string
	// OAuth Config
	ClientID     string
	ClientSecret string
	RedirectURI  string
	AuthURL      string
	TokenURL     string
	UserAPIURL   string
	Scopes       string
}

type AppConfig struct {
	PostgresHost      string
	PostgresPort      string
	PostgresDatabase  string
	PostgresUser      string
	PostgresPassword  string
	TemplateDirectory string
	DiscordConfig     *DiscordConfig
}

func init() {
	Config = newAppConfig()
	log.Info().Any("config", Config).Msg("successfully loaded application configuration")
}

func getOrDefault(name string, defaultValue string) string {
	value, present := os.LookupEnv(name)
	if !present {
		return defaultValue
	}
	return value
}

func newAppConfig() *AppConfig {
	discordConfig := &DiscordConfig{
		APIKey:       getOrDefault("DISCORD_API_KEY", "fake-discord-api-key"),
		ChannelID:    getOrDefault("DISCORD_CHANNEL_ID", "fake-discord-channel-id"),
		GuildID:      getOrDefault("DISCORD_GUILD_ID", "fake-discord-guild-id"),
		ClientID:     getOrDefault("DISCORD_CLIENT_ID", "fake-discord-client-id"),
		ClientSecret: getOrDefault("DISCORD_CLIENT_SECRET", "fake-discord-client-secret"),
		RedirectURI:  getOrDefault("DISCORD_REDIRECT_URI", "http://localhost:8080/discord/callback"),
		AuthURL:      getOrDefault("DISCORD_AUTH_URL", "https://discord.com/api/oauth2/authorize"),
		TokenURL:     getOrDefault("DISCORD_TOKEN_URL", "https://discord.com/api/oauth2/token"),
		UserAPIURL:   getOrDefault("DISCORD_USER_API_URL", "https://discord.com/api/users/@me"),
		Scopes:       getOrDefault("DISCORD_SCOPES", "identify"),
	}
	config := &AppConfig{
		PostgresHost:      getOrDefault("POSTGRES_HOST", "localhost"),
		PostgresPort:      getOrDefault("POSTGRES_PORT", "5432"),
		PostgresDatabase:  getOrDefault("POSTGRES_DB", "postgres"),
		PostgresUser:      getOrDefault("POSTGRES_USER", "postgres"),
		PostgresPassword:  getOrDefault("POSTGRES_PASSWORD", "postgres"),
		TemplateDirectory: getOrDefault("TEMPLATE_DIRECTORY", "templates/"),
		DiscordConfig:     discordConfig,
	}
	return config
}
