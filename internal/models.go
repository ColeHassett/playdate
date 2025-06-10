package internal

import (
	"time"

	"github.com/uptrace/bun"
)

type PlayDate struct {
	bun.BaseModel `bun:"table:playdate"`

	ID          int       `bun:",pk,autoincrement" json:"id"`
	CreatedDate time.Time `bun:"created_date,nullzero,default:CURRENT_TIMESTAMP" json:"created_date"`
	Game        string    `bun:"game,notnull" json:"game"`
	Date        time.Time `bun:"date,nullzero" json:"date"`
}

type Player struct {
	bun.BaseModel `bun:"table:player"`

	ID          int       `bun:",pk,autoincrement" json:"id"`
	CreatedDate time.Time `bun:"created_date,nullzero,default:CURRENT_TIMESTAMP" json:"created_date"`
	Name        string    `bun:"name,notnull" json:"name"`
	DiscordID   string    `bun:"discord_id,notnull" json:"discord_id"`
}
