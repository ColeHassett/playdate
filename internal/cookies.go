package internal

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"github.com/uptrace/bun"
)

var ErrMissingCookie = errors.New("missing cookie")
var ErrPlayerFromCookieNotFound = errors.New("player from cookie not found")
var ErrPlayerNotFoundInContext = errors.New("failed to find player in gin context")

// Middleware to check request cookies and gracefully handle errors that could occur
func CookiesMiddleware(api *Api) gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Debug().Msg("starting middleware checking the users cookies...")
		player, err := FindPlayerFromPlayDateCookie(api.db, c)
		switch err {
		case ErrMissingCookie:
			log.Warn().Msg("request missing cookie")
			c.Redirect(http.StatusFound, "/")
			return
		case ErrPlayerFromCookieNotFound:
			log.Error().Err(err).Msg(err.Error())
			api.userLogout(c)
			return
		}
		log.Debug().Msg("finished middleware checking the users cookies, player added to gin context!")
		c.Set("player", player)

		// release request to next function in chain
		c.Next()
	}
}

// Given a request context return the player found. The player is verified to exist in the database and
// the sessionId is verified againt the database aswell.
func FindPlayerFromPlayDateCookie(db *bun.DB, c *gin.Context) (*Player, error) {
	cookie, _ := c.Cookie("playdate")
	if cookie == "" {
		return nil, ErrMissingCookie
	}
	// render actual home page since user has a cookie
	player, err := getPlayerFromCookieString(db, cookie, c.Request.Context())
	if err != nil {
		return nil, ErrPlayerFromCookieNotFound
	}
	return player, nil
}

// Retrieve the player from the gin context
func GetPlayerFromContext(c *gin.Context) (*Player, error) {
	playerFromContext, ok := c.Get("player")
	if !ok {
		return nil, ErrPlayerNotFoundInContext
	}
	player := playerFromContext.(*Player)
	return player, nil
}

// Given a cookie string find the associated player within the database
func getPlayerFromCookieString(db *bun.DB, cookie string, ctx context.Context) (*Player, error) {
	player := &Player{SessionId: cookie}
	err := db.NewSelect().Model(player).Where("session_id = ?", player.SessionId).Scan(ctx)
	if err != nil {
		// if the given player id doesn't exist just return the called to the home page
		msg := "failed to find the player from their cookie"
		log.Err(err).Str("cookie", cookie).Any("player", player).Msg(msg)
		return nil, errors.New(msg)
	}
	return player, nil
}
