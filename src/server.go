package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

func main() {
	// note that this is path referenced from within the docker container
	banner, err := os.ReadFile("banner.txt")
	if err != nil {
		log.Fatal("Failed to read banner text file. err=", err)
	}
	log.Println(string(banner))

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", Config.PostgresUser, Config.PostgresPassword, Config.PostgresHost, Config.PostgresPort, Config.PostgresDatabase)
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
	db := bun.NewDB(sqldb, pgdialect.New())
	defer db.Close()

	// Check the connection
	if err := db.Ping(); err != nil {
		log.Fatalf("failed to connect to PostgreSQL. url=%s err=%v", dsn, err)
	}
	log.Println("Connected to PostgreSQL successfully!")

	go startAPI(db)     // start webserver on subprocess
	startDiscordBot(db) // start discord bot on main process
}
