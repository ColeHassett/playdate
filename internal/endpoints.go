package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/gin-contrib/logger"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/uptrace/bun"
)

func StartAPI(db *bun.DB, dg *discordgo.Session) {
	api := Api{db: db, dg: dg, ctx: context.Background()}

	router := gin.New()        // NOTE: Not using Default to avoid the wrong logger being used?
	router.Use(gin.Recovery()) // handle panics (aka unhandled exceptions)
	gin.SetMode(gin.DebugMode) // adds additional debugging features for the gin http server.

	// Add the zerolog to gin's middleware
	router.Use(logger.SetLogger(
		logger.WithUTC(true),                            // Log time in UTC
		logger.WithDefaultLevel(zerolog.InfoLevel),      // Default log level
		logger.WithClientErrorLevel(zerolog.WarnLevel),  // Level for 4xx errors
		logger.WithServerErrorLevel(zerolog.ErrorLevel), // Level for 5xx errors
	))

	// Template Endpoints
	router.LoadHTMLGlob(fmt.Sprintf("%s/**/*.html", Config.TemplateDirectory))
	router.StaticFile("custom-colors.css", fmt.Sprintf("%s/custom-colors.css", Config.TemplateDirectory))
	// Templated endpoints
	router.GET("/", api.index)
	router.POST("/register", api.registerUserTemplate)
	router.GET("/playdate", api.showPlayDateForm)
	router.POST("/playdate", api.createPlayDateTemplate)
	router.GET("/playdate/:id", api.getPlayDateTemplate)
	router.POST("/playdate/:id/yes", api.setPlayDateAttendence)
	router.POST("/playdate/:id/maybe", api.setPlayDateAttendence)
	router.POST("/playdate/:id/no", api.setPlayDateAttendence)

	go api.watchDog()
	router.Run()
}

type BaseTemplateData struct {
	ServerError string
	Errors      map[string]string
}

type Api struct {
	db  *bun.DB
	dg  *discordgo.Session
	ctx context.Context
}

func (a *Api) watchDog() {
	log.Info().Msg("Watching for PlayDates..")
	ticker := time.NewTicker(time.Minute)

	go func() {
		for {
			select {
			case <-ticker.C:
				a.fetchPoppedDates()
			}
		}
	}()
}

func (a *Api) index(c *gin.Context) {
	cookie, _ := c.Cookie("playdate")
	log.Debug().Str("Cookie", cookie).Str("HX-Request", c.Request.Header.Get("HX-Request")).Msg("rendering index")

	state := gin.H{"Errors": map[string]string{}}
	if cookie == "" {
		c.HTML(http.StatusOK, "pages/register.html", gin.H{})
		return
	}

	// render actual home page since user has a cookie
	state["SignedUp"] = true
	playdates := []*PlayDate{}
	err := a.db.NewSelect().Model(&playdates).Scan(a.ctx)
	if err != nil {
		log.Error().Err(err).Msg("DB error in index")
		state["ServerError"] = "Failed to retrieve playdates due to a server error. Please try again later."
	}
	state["PlayDates"] = playdates
	c.HTML(http.StatusOK, "pages/home.html", state)
}

func (a *Api) showPlayDateForm(c *gin.Context) {
	c.HTML(http.StatusOK, "partials/playdate-form.html", gin.H{})
}

func (a *Api) createPlayDateTemplate(c *gin.Context) {
	inputGame := c.PostForm("game")
	inputDatetime := c.PostForm("date")

	formData := gin.H{"Game": inputGame, "Date": inputDatetime}
	errors := map[string]string{}
	if inputGame == "" {
		errors["game"] = "game is required"
	}
	if inputDatetime == "" {
		errors["date"] = "date is required"
	}
	// NOTE: that this date is not random but instead hardcoded into the standard
	// libary to code layouts against.
	expectedTimeLayout := "2006-01-02T15:04"
	parsedDatetime, err := time.Parse(expectedTimeLayout, inputDatetime)
	if err != nil {
		errors["date"] = "invalid format for date/time, please use layout 2025-01-01T12:00"
	}
	formData["Errors"] = errors
	if len(errors) > 0 {
		c.HTML(http.StatusOK, "partials/playdate-form.html", formData)
		return
	}

	playdate := PlayDate{Game: inputGame, Date: parsedDatetime}
	_, err = a.db.NewInsert().Model(&playdate).Exec(a.ctx)
	if err != nil {
		log.Err(err).Any("playdate", playdate).Msg("failed to insert new playdate")
		formData["ServerError"] = err
		c.HTML(http.StatusOK, "partials/playdate-form.html", formData)
		return
	}

	// redirect the user back to the index router (i.e. the homepage)
	c.Header("HX-Location", "/")
}

func (a *Api) getPlayDateTemplate(c *gin.Context) {
	state := gin.H{}
	errors := map[string]string{}
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		// redirect the user to the home page if they request with an improper id
		log.Err(err).Str("playdateID", c.Param("id")).Msg("failed to parse given playdate id")
		c.Redirect(http.StatusFound, "/")
		return
	}

	log.Info().Int("id", id).Msg("Querying for players related to playdate")
	playdate := &PlayDate{ID: id}
	err = a.db.NewSelect().Model(playdate).WherePK().Scan(c.Request.Context())
	if err != nil {
		// if the given id doesn't exist just return the called to the home page
		log.Err(err).Int("playdateID", id).Msg("failed to find playdate")
		c.Redirect(http.StatusFound, "/")
		return
	}
	playdatePlayers := []*PlayDateToPlayer{}
	err = a.db.NewSelect().Model(&playdatePlayers).Relation("Player").Where("id = ?", id).Scan(c.Request.Context())
	if err != nil {
		// report error back to user, but just render the page like normal
		log.Err(err).Any("playdate", playdate).Msg("failed to find related players to playdate")
		errors["PlayDatePlayers"] = err.Error()
	}

	log.Debug().Interface("playdatePlayers", playdatePlayers).Msg("Playdate players details")
	state["Errors"] = errors
	state["PlayDate"] = playdate
	state["PlayDatePlayers"] = playdatePlayers
	log.Debug().Interface("playdate", playdate).Msg("Playdate details")
	if c.Request.Header.Get("HX-Request") == "" {
		c.HTML(http.StatusOK, "pages/playdate.html", state)
	} else {
		c.HTML(http.StatusOK, "partials/playdate.html", state)
	}
}

func (a *Api) setPlayDateAttendence(c *gin.Context) {
	uriParts := strings.Split(c.Request.RequestURI, "/")
	action := uriParts[len(uriParts)-1]
	log.Debug().Str("action", action).Msg("received attendance action")
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		// redirect the user to the home page if they request with an improper id
		log.Err(err).Str("playdateID", c.Param("id")).Msg("failed to parse given playdate id")
		c.Redirect(http.StatusFound, "/")
		return
	}

	playdate := &PlayDate{ID: id}
	err = a.db.NewSelect().Model(playdate).WherePK().Scan(c.Request.Context())
	if err != nil {
		// if the given id doesn't exist just return the called to the home page
		log.Err(err).Int("playdateID", id).Msg("failed to find playdate")
		c.Redirect(http.StatusFound, "/")
		return
	}

	player := &Player{}
	cookie, err := c.Cookie("playdate")
	if err != nil {
		// redirect the user to the home page if they don't have a cookie
		log.Err(err).Msg("failed to parse user's cookie")
		c.Redirect(http.StatusFound, "/")
		return
	}

	err = json.Unmarshal([]byte(cookie), player)
	if err != nil {
		// if the given player id doesn't exist just return the called to the home page
		log.Err(err).Str("cookie", cookie).Int("playerID", id).Msg("failed to unmarshal cookie into player struct")
		c.Redirect(http.StatusFound, "/")
		return
	}
	err = a.db.NewSelect().Model(player).WherePK().Scan(c.Request.Context())
	if err != nil {
		// if the given player id doesn't exist just return the called to the home page
		log.Err(err).Any("player", player).Int("playerID", id).Msg("failed to find the player from their cookie")
		c.Redirect(http.StatusFound, "/")
		return
	}

	log.Info().Int("playdateID", playdate.ID).Int("playerID", player.ID).Str("action", action).Msg("attempting to set playdate attendance")
	errors := map[string]string{}
	rel := &PlayDateToPlayer{PlayDateID: playdate.ID, PlayerID: player.ID, Attending: action}
	_, err = a.db.NewInsert().Model(rel).On("CONFLICT (playdate_id, player_id) DO UPDATE").Set("attending = EXCLUDED.attending").Exec(a.ctx)
	if err != nil {
		// send error back to user within the players-table.html
		log.Error().Err(err).Interface("relation", rel).Msg("failed to insert playdate to player relation")
		errors["PlayDatePlayers"] = err.Error()
	} else {
		log.Info().Interface("relation", rel).Msg("successfully inserted playdate to player relation")
	}

	playdatePlayers := []*PlayDateToPlayer{}
	err = a.db.NewSelect().Model(&playdatePlayers).Relation("Player").Where("id = ?", player.ID).Scan(c.Request.Context())
	if err != nil {
		// report error back to user, but just render the page like normal
		log.Err(err).Any("playdate", playdate).Msg("failed to find related players to playdate")
		errors["PlayDatePlayers"] = err.Error()
	}

	state := gin.H{}
	state["Errors"] = errors
	state["PlayDatePlayers"] = playdatePlayers
	state["PlayDate"] = playdate
	c.HTML(http.StatusOK, "partials/players-table.html", state)
}

func (a *Api) registerUserTemplate(c *gin.Context) {
	name := c.PostForm("name")
	discID := c.PostForm("discID")

	formData := gin.H{"Name": name, "DiscID": discID}
	errors := map[string]string{}
	if name == "" {
		errors["name"] = "name is required"
	}
	if discID == "" {
		errors["discID"] = "discID is required"
	}
	if len(errors) > 0 {
		formData["Errors"] = errors
		log.Debug().Any("errors", errors).Msg("form data didn't meet validation")
		c.HTML(http.StatusOK, "partials/register.html", formData)
		return
	}

	player := Player{Name: name, DiscordID: discID}
	err := a.db.NewSelect().Model(&player).Where("discord_id = ?", player.DiscordID).Scan(a.ctx)
	if err != nil {
		_, err := a.db.NewInsert().Model(&player).Exec(a.ctx)
		if err != nil {
			log.Err(err).Msg("failed to create new player")
			formData["ServerError"] = err.Error()
			c.HTML(http.StatusOK, "partials/register.html", formData)
			return
		}
	}

	val, _ := json.Marshal(player)
	c.SetCookie("playdate", string(val), 0, "/", "", false, true)
	c.Status(http.StatusCreated)
	c.Header("HX-Location", "/")
}

func (a *Api) fetchPoppedDates() {
	log.Info().Msg("Any PlayDates??")
	state := gin.H{"Errors": map[string]string{}}

	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		log.Error().Err(err).Msg("Cannot load time location")
		state["ServerError"] = "Failed to load time location"
	}
	now := time.Now().In(loc)
	playdates := []*PlayDate{}
	err = a.db.NewSelect().Model(&playdates).Where("date = ?", now.Format("2006-01-02T15:04")).Scan(a.ctx)
	if err != nil {
		log.Error().Err(err).Msg("Watch is Kill")
		state["ServerError"] = "Watch Dead"
	}

	for _, p := range playdates {
		players := "<@108736074557239296>"
		for _, b := range p.Players {
			players = players + fmt.Sprintf("<@%s>", b.DiscordID)
		}
		a.dg.ChannelMessageSend(Config.DiscordChannelID, fmt.Sprintf("time to play: %s", players))
	}
}
