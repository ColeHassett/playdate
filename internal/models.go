package internal

import (
	"time"

	"github.com/uptrace/bun"
)

// source: https://bun.uptrace.dev/guide/relations.html#many-to-many-relation
func InitializeManyToManyRelationships(db *bun.DB) {
	// Register many to many model so bun can better recognize m2m relation.
	// This should be done before you use the model for the first time.
	db.RegisterModel((*PlayDateToPlayer)(nil))
}

type PlayDate struct {
	bun.BaseModel `bun:"table:playdate"`

	ID          int       `bun:",pk,autoincrement" json:"id"`
	CreatedDate time.Time `bun:"created_date,nullzero,default:CURRENT_TIMESTAMP" json:"created_date"`
	Game        string    `bun:"game,notnull" json:"game"`
	Date        time.Time `bun:"date,nullzero" json:"date"`
	Players     []*Player `bun:"m2m:playdate_player,join:PlayDate=Player"`
}

type Player struct {
	bun.BaseModel `bun:"table:player"`

	ID          int       `bun:",pk,autoincrement" json:"id"`
	CreatedDate time.Time `bun:"created_date,nullzero,default:CURRENT_TIMESTAMP" json:"created_date"`
	Name        string    `bun:"name,notnull" json:"name"`
	DiscordID   string    `bun:"discord_id,notnull" json:"discord_id"`
}

type PlayDateToPlayer struct {
	bun.BaseModel `bun:"table:playdate_player"`

	PlayDateID int       `bun:"playdate_id,pk"`
	PlayDate   *PlayDate `bun:"rel:belongs-to,join:playdate_id=id"`
	PlayerID   int       `bun:"player_id,pk"`
	Player     *Player   `bun:"rel:belongs-to,join:player_id=id"`
	Attending  string    `bun:"attending,notnull"`
}
