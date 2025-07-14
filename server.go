package main

import (
	"database/sql"
	"embed"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"uc181discord/games/bot/internal"

	"github.com/bwmarrin/discordgo"
	"github.com/pressly/goose/v3"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/extra/bundebug"
)

//go:embed migrations/*
var embedMigrations embed.FS

func main() {
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})

	// capture process kill signals into channel
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	printBanner()
	db := setupPostgresClient()
	dg := setupDiscordClient()
	internal.SetupDiscordHandlers(db, dg)
	internal.SendPatchNotes(dg)
	go internal.StartAPI(db, dg) // start webserver on subprocess

	// listen for kill signals from OS
	<-quit
	shutdownServer(db, dg)
}

func shutdownServer(db *bun.DB, dg *discordgo.Session) {
	log.Info().Msg("Starting server shutdown...")
	err := db.Close()
	if err != nil {
		log.Err(err).Msg("failed to close postgres client")
	}
	err = internal.DeleteDiscordCommands(dg)
	if err != nil {
		log.Err(err).Msg("failed to delete existing discord commands")
	}
	err = dg.Close()
	if err != nil {
		log.Err(err).Msg("failed to close discord client")
	}
	log.Info().Msg("Successfully shutdown server")
}

func printBanner() {
	banner, err := os.ReadFile("banner.txt")
	if err != nil {
		log.Err(err).Msg("Failed to read banner text file")
	}
	fmt.Print(string(banner))
}

func setupPostgresClient() *bun.DB {
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", internal.Config.PostgresUser, internal.Config.PostgresPassword, internal.Config.PostgresHost, internal.Config.PostgresPort, internal.Config.PostgresDatabase)
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
	db := bun.NewDB(sqldb, pgdialect.New())
	db.AddQueryHook(bundebug.NewQueryHook(bundebug.WithVerbose(true)))

	// init m2m relationships the bun way
	internal.InitializeManyToManyRelationships(db)

	// Check the connection
	if err := db.Ping(); err != nil {
		log.Err(err).Str("url", dsn).Msg("failed to connect to PostgreSQL")
	}
	log.Debug().Msg("Connected to PostgreSQL successfully!")

	// run database migrations automatically
	goose.SetBaseFS(embedMigrations)
	if err := goose.SetDialect("postgres"); err != nil {
		log.Panic().Err(err).Msg("failed to set dialect for goose migrations")
	}
	if err := goose.Up(sqldb, "migrations"); err != nil {
		log.Panic().Err(err).Msg("failed to run goose migrations")
	}

	log.Debug().Msg("successfully started postgres client")
	return db
}

func setupDiscordClient() *discordgo.Session {
	dg, err := discordgo.New("Bot " + internal.Config.DiscordConfig.APIKey)
	if err != nil {
		log.Err(err).Msg("failed to create Discord client")
	}

	err = dg.Open()
	if err != nil {
		log.Err(err).Msg("failed to open websocket to Discord")
	}

	log.Debug().Msg("successfully started discord clinet")
	return dg
}
