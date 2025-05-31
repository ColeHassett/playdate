package main

import (
	"database/sql"
	"embed"
	"fmt"
	"log"
	"os"
	"uc181discord/games/bot/internal"

	"github.com/pressly/goose/v3"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

//go:embed migrations/*
var embedMigrations embed.FS

func main() {
	// note that this is path referenced from within the docker container
	banner, err := os.ReadFile("banner.txt")
	if err != nil {
		log.Fatal("Failed to read banner text file. err=", err)
	}
	log.Println(string(banner))

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", internal.Config.PostgresUser, internal.Config.PostgresPassword, internal.Config.PostgresHost, internal.Config.PostgresPort, internal.Config.PostgresDatabase)
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
	db := bun.NewDB(sqldb, pgdialect.New())
	defer db.Close()

	// Check the connection
	if err := db.Ping(); err != nil {
		log.Fatalf("failed to connect to PostgreSQL. url=%s err=%v", dsn, err)
	}
	log.Println("Connected to PostgreSQL successfully!")

	// run database migrations automatically
	goose.SetBaseFS(embedMigrations)
	if err := goose.SetDialect("postgres"); err != nil {
		panic(err)
	}
	if err := goose.Up(sqldb, "migrations"); err != nil {
		panic(err)
	}

	internal.StartAPI(db) // start webserver on subprocess
	// startDiscordBot(db) // start discord bot on main process
}
