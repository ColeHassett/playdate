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
	case string(AttendanceYes), "👍":
		return AttendanceYes
	case string(AttendanceMaybe), "🤔":
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
	Players     []*Player           `bun:"m2m:playdate_player,join:PlayDate=Player"`
	Owner       *Player             `bun:"rel:belongs-to,join:owner_id=id"`
	Attendances []*PlayDateToPlayer `bun:"rel:has-many,join:id=playdate_id"`
}

type Player struct {
	bun.BaseModel `bun:"table:player"`

	ID               int       `bun:",pk,autoincrement" json:"id"`
	CreatedDate      time.Time `bun:"created_date,nullzero,default:CURRENT_TIMESTAMP" json:"created_date"`
	Name             string    `bun:"name,notnull,unique" json:"name"`
	Password         string    `bun:"password,notnull"`
	DiscordID        string    `bun:"discord_id,notnull,unique" json:"discord_id"`
	VerificationCode string    `bun:"verification_code,notnull" json:"verification_code"`
	SessionId        string    `bun:"session_id,notnull"`
	OAuthToken       string    `bun:"oauth_token"`

	// just relationship fields for bun to utilize
	Attendances []*PlayDateToPlayer `bun:"rel:has-many,join:id=player_id"`
	PlayDates   []*PlayDate         `bun:"m2m:playdate_player,join:Player=PlayDate"`
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
