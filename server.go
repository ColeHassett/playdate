package main

import (
	"database/sql"
	"embed"
	"fmt"
	"os"
	"uc181discord/games/bot/internal"

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

	// note that this is path referenced from within the docker container
	banner, err := os.ReadFile("banner.txt")
	if err != nil {
		log.Err(err).Msg("Failed to read banner text file")
	}
	fmt.Print(string(banner))

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", internal.Config.PostgresUser, internal.Config.PostgresPassword, internal.Config.PostgresHost, internal.Config.PostgresPort, internal.Config.PostgresDatabase)
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
	db := bun.NewDB(sqldb, pgdialect.New())
	defer db.Close()
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

	internal.StartAPI(db) // start webserver on subprocess
	// startDiscordBot(db) // start discord bot on main process
}
