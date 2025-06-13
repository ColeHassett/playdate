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

type PlayDateStatus string

const (
	PlayDateStatusPending PlayDateStatus = "pending"
	PlayDateStatusDone    PlayDateStatus = "done"
)

type Attendance string

const (
	AttendanceNo    Attendance = "no"
	AttendanceMaybe Attendance = "maybe"
	AttendanceYes   Attendance = "yes"
)

func AttendanceFrom(s string) Attendance {
	switch s {
	case string(AttendanceYes):
		return AttendanceYes
	case string(AttendanceMaybe):
		return AttendanceMaybe
	default:
		return AttendanceNo
	}
}

type PlayDate struct {
	bun.BaseModel `bun:"table:playdate"`

	ID          int            `bun:",pk,autoincrement" json:"id"`
	CreatedDate time.Time      `bun:"created_date,nullzero,default:CURRENT_TIMESTAMP" json:"created_date"`
	Game        string         `bun:"game,notnull" json:"game"`
	Date        time.Time      `bun:"date,nullzero" json:"date"`
	Status      PlayDateStatus `bun:"status,notnull,default:'pending',type:playdate_status"`
	OwnerId     int            `bun:"owner_id,notnull"`

	// just relationship fields for bun to utilize
	Players []*Player `bun:"m2m:playdate_player,join:PlayDate=Player"`
	Owner   *Player   `bun:"rel:belongs-to,join:owner_id=id"`
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

	PlayDateID int        `bun:"playdate_id,pk"`
	PlayerID   int        `bun:"player_id,pk"`
	Attending  Attendance `bun:"attending,notnull,default:'no',type:attendance"`

	// just relationship fields for bun to utilize
	PlayDate *PlayDate `bun:"rel:belongs-to,join:playdate_id=id"`
	Player   *Player   `bun:"rel:belongs-to,join:player_id=id"`
}
