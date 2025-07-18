package internal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/gin-contrib/logger"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/uptrace/bun"
	"golang.org/x/crypto/bcrypt"
)

var (
	// NOTE: This is just a const for the entire server to provide easy access to convert timestamps into EST.
	// This would be better on the client as their browser could convert the timestamp to their local timezone.
	easternLocation, _ = time.LoadLocation("America/New_York")
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

	// custom template functions
	router.SetFuncMap(template.FuncMap{
		"formatTime":   FormatTime,
		"relativeTime": RelativeTime,
	})

	// Template Endpoints
	router.LoadHTMLGlob(fmt.Sprintf("%s/**/*.html", Config.TemplateDirectory))
	router.StaticFile("custom-colors.css", fmt.Sprintf("%s/custom-colors.css", Config.TemplateDirectory))

	// NOTE: Login/Registration Routes
	router.GET("/", api.index)
	router.POST("/login", api.userLogin)
	router.DELETE("/logout", api.userLogout)
	router.POST("/register", api.registerUserTemplate)
	router.POST("/verify", api.verifyPlayer)
	// Navigate around?
	router.GET("/register", api.goToRegisterUser)
	router.GET("/login", api.goToLogin)

	// NOTE: Discord OAuth Routes
	router.GET("/discord/login", api.handleOAuthLogin)
	router.GET("/discord/callback", api.handleOAuthCallback)

	// NOTE: Health Check
	router.GET("/health", api.healthCheck)

	// NOTE: Application Routes
	router.GET("/playdate", api.showPlayDateForm)
	router.POST("/playdate", api.createPlayDateTemplate)
	router.GET("/playdate/:id", api.getPlayDateTemplate)
	router.POST("/playdate/:id/yes", api.setPlayDateAttendence)
	router.POST("/playdate/:id/maybe", api.setPlayDateAttendence)
	router.POST("/playdate/:id/no", api.setPlayDateAttendence)

	// Start discord handlers
	dg.AddHandler(func(s *discordgo.Session, r *discordgo.MessageReactionAdd) {
		api.setPlayDateAttendenceFromDisc(r.MessageReaction)
	})
	api.sendPatchNotes()

	go api.watchDog()
	router.Run("0.0.0.0:8080")
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

type GitHubRelease struct {
	TagName     string    `json:"tag_name"`
	Name        string    `json:"name"`
	Body        string    `json:"body"`
	PreRelease  bool      `json:"prerelease"`
	PublishedAt time.Time `json:"published_at"`
}

func (a *Api) watchDog() {
	log.Info().Msg("Watching for PlayDates..")
	ticker := time.NewTicker(time.Minute / 2)

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
	player, err := a.findPlayerFromCookie(c)
	if err != nil {
		// delete the cookie that was on the request to avoid an infinite loop
		a.userLogout(c)
		return
	}

	// find playdates that are upcoming
	upcomingPlaydates := []*PlayDate{}
	err = a.db.NewSelect().
		Model(&upcomingPlaydates).
		Relation("Owner").
		Relation("Players").
		Where("play_date.status = ?", PlayDateStatusPending).
		Order("play_date.created_date asc").
		Scan(a.ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to query for upcoming playdates")
		state["ServerError"] = "Failed to retrieve upcoming playdates due to a server error. Please try again later."
	}

	// find playdates in the past
	pastPlaydates := []*PlayDate{}
	err = a.db.NewSelect().
		Model(&pastPlaydates).
		Relation("Owner").
		Relation("Players").
		Where("play_date.status = ?", PlayDateStatusDone).
		Order("play_date.created_date desc").
		Scan(a.ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to query for past playdates")
		state["ServerError"] = "Failed to retrieve past playdates due to a server error. Please try again later."
	}
	playdates := append(upcomingPlaydates, pastPlaydates...)

	// NOTE: manually convert timestamp to eastern
	for _, p := range playdates {
		p.Date = p.Date.In(easternLocation)
	}

	state["PlayDates"] = playdates
	state["Player"] = player

	c.HTML(http.StatusOK, "pages/home.html", state)
}

func (a *Api) showPlayDateForm(c *gin.Context) {
	c.HTML(http.StatusOK, "partials/playdate-form.html", gin.H{})
}

func (a *Api) createPlayDateTemplate(c *gin.Context) {
	player, err := a.findPlayerFromCookie(c)
	if err != nil {
		c.Redirect(http.StatusFound, "/")
		return
	}

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
	parsedDatetime, err := time.ParseInLocation(expectedTimeLayout, inputDatetime, easternLocation)
	now := time.Now().In(easternLocation)
	if err != nil {
		formData["Date"] = ""
		errors["date"] = "invalid format for date/time, please use layout 2025-01-01T12:00"
	} else if parsedDatetime.Before(now) {
		formData["Date"] = ""
		errors["date"] = fmt.Sprintf("can not make a playdate in the past, %v is before %v", parsedDatetime, now)
	}
	formData["Errors"] = errors
	if len(errors) > 0 {
		c.HTML(http.StatusOK, "partials/playdate-form.html", formData)
		return
	}
	log.Debug().Str("datetime", parsedDatetime.String()).Msg("*** Checking time prior to db")

	playdate := PlayDate{Game: inputGame, Date: parsedDatetime, OwnerId: player.ID}
	_, err = a.db.NewInsert().Model(&playdate).Exec(a.ctx)
	if err != nil {
		log.Err(err).Any("playdate", playdate).Msg("failed to insert new playdate")
		formData["ServerError"] = err
		c.HTML(http.StatusOK, "partials/playdate-form.html", formData)
		return
	}

	// send notification to configure channel to share the new playdate to the masses!
	msg := fmt.Sprintf("Playdate %s at %s by %s! Check it out here: https://playdate.colinthatcher.dev/playdate/%d", playdate.Game, FormatTime(&playdate.Date), player.Name, playdate.ID)
	dgMsg, err := a.dg.ChannelMessageSend(Config.DiscordConfig.ChannelID, msg)
	if err != nil {
		log.Err(err).Any("playdate", playdate).Msg("failed to send message for new playdate to discord")
	}
	InitAttendanceReactions(a, dgMsg)

	// redirect the user back to the index router (i.e. the homepage)
	c.Header("HX-Location", "/")
}

func (a *Api) getPlayDateTemplate(c *gin.Context) {
	_, err := a.findPlayerFromCookie(c)
	if err != nil {
		c.Redirect(http.StatusFound, "/")
		return
	}

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
	err = a.db.NewSelect().Model(playdate).Relation("Owner").WherePK().Scan(c.Request.Context())
	if err != nil {
		// if the given id doesn't exist just return the called to the home page
		log.Err(err).Int("playdateID", id).Msg("failed to find playdate")
		c.Redirect(http.StatusFound, "/")
		return
	}
	playdatePlayers := []*PlayDateToPlayer{}
	err = a.db.NewSelect().Model(&playdatePlayers).Relation("Player").Where("playdate_id = ?", id).Scan(c.Request.Context())
	if err != nil {
		// report error back to user, but just render the page like normal
		log.Err(err).Any("playdate", playdate).Msg("failed to find related players to playdate")
		errors["PlayDatePlayers"] = err.Error()
	}

	log.Debug().Interface("playdatePlayers", playdatePlayers).Msg("Playdate players details")
	// NOTE: Manually parse timestamp into eastern time
	playdate.Date = playdate.Date.In(easternLocation)
	playdate.CreatedDate = playdate.CreatedDate.In(easternLocation)
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
	player, err := a.findPlayerFromCookie(c)
	if err != nil {
		c.Redirect(http.StatusFound, "/")
		return
	}

	// TODO: This can probably be cleaned up somehow
	uriParts := strings.Split(c.Request.RequestURI, "/")
	inputAction := uriParts[len(uriParts)-1]
	attendance := AttendanceFrom(inputAction) // parse input attendence action to internal enum
	log.Debug().Any("action", attendance).Msg("received attendance action")
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

	log.Info().Int("playdateID", playdate.ID).Int("playerID", player.ID).Any("action", attendance).Msg("attempting to set playdate attendance")
	errors := map[string]string{}
	rel := &PlayDateToPlayer{PlayDateID: playdate.ID, PlayerID: player.ID, Attending: attendance}
	_, err = a.db.NewInsert().Model(rel).On("CONFLICT (playdate_id, player_id) DO UPDATE").Set("attending = EXCLUDED.attending").Exec(a.ctx)
	if err != nil {
		// send error back to user within the players-table.html
		log.Error().Err(err).Interface("relation", rel).Msg("failed to insert playdate to player relation")
		errors["PlayDatePlayers"] = err.Error()
	} else {
		log.Info().Interface("relation", rel).Msg("successfully inserted playdate to player relation")
	}

	playdatePlayers := []*PlayDateToPlayer{}
	err = a.db.NewSelect().Model(&playdatePlayers).Relation("Player").Where("playdate_id = ?", playdate.ID).Scan(c.Request.Context())
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

func (a *Api) setPlayDateAttendenceFromDisc(r *discordgo.MessageReaction) {
	if r.UserID == "1252426978313633812" {
		log.Debug().Msg("Reaction created by bot")
		return
	}

	log.Info().Any("Reaction", r).Msg("Setting player attendance")
	discId := r.UserID
	react := r.Emoji.Name
	msg, err := a.dg.ChannelMessage(Config.DiscordConfig.ChannelID, r.MessageID)
	if err != nil {
		log.Err(err).Msg("failed to get reaction message")
		return
	}
	if msg.Author.ID != "1252426978313633812" {
		log.Debug().Msg("Not a bot message")
		return
	}

	attendance := AttendanceFrom(react) // parse input attendence action to internal enum
	msgSplit := strings.Split(msg.Content, "/")
	if len(msgSplit) <= 1 {
		log.Debug().Msg("Not a playdate")
		return
	}
	pId, err := strconv.Atoi(msgSplit[len(msgSplit)-1])
	if err != nil {
		log.Err(err).Msg("failed to parse given playdate id")
		return
	}

	playdate := &PlayDate{ID: pId}
	err = a.db.NewSelect().Model(playdate).WherePK().Scan(a.ctx)
	if err != nil {
		log.Err(err).Int("playdateID", pId).Msg("failed to find playdate")
		return
	}
	if playdate.Status != PlayDateStatusPending {
		log.Debug().Msg("PlayDate already happened")
		return
	}

	log.Info().Int("playdateID", playdate.ID).Str("discordId", discId).Any("action", attendance).Msg("attempting to set playdate attendance")
	player := &Player{DiscordID: discId}
	err = a.db.NewSelect().Model(player).Where("discord_id = ?", player.DiscordID).Scan(a.ctx)
	if err != nil {
		log.Err(err).Str("discID", discId).Msg("failed to find player")
		a.dg.ChannelMessageSend(Config.DiscordConfig.ChannelID, "Please go here to make an account: https://playdate.colinthatcher.dev/discord/login")
		err = a.dg.MessageReactionRemove(Config.DiscordConfig.ChannelID, msg.ID, r.Emoji.APIName(), discId)
		if err != nil {
			log.Err(err).Msg("Failed to remove reaction on anon user")
		}
		return
	}
	rel := &PlayDateToPlayer{PlayDateID: playdate.ID, PlayerID: player.ID, Attending: attendance}
	_, err = a.db.NewInsert().Model(rel).On("CONFLICT (playdate_id, player_id) DO UPDATE").Set("attending = EXCLUDED.attending").Exec(a.ctx)
	if err != nil {
		// send error back to user within the players-table.html
		log.Error().Err(err).Interface("relation", rel).Msg("failed to insert playdate to player relation")
	} else {
		log.Info().Interface("relation", rel).Msg("successfully inserted playdate to player relation")
	}

	playdatePlayers := []*PlayDateToPlayer{}
	err = a.db.NewSelect().Model(&playdatePlayers).Relation("Player").Where("playdate_id = ?", playdate.ID).Scan(a.ctx)
	if err != nil {
		// report error back to user, but just render the page like normal
		log.Err(err).Any("playdate", playdate).Msg("failed to find related players to playdate")
	}

	if r.Emoji.APIName() != "👍" {
		err = a.dg.MessageReactionRemove(Config.DiscordConfig.ChannelID, msg.ID, "👍", discId)
		if err != nil {
			log.Err(err).Str("Reaction", "👍").Msg("Failed to remove reaction")
		}
	}
	if r.Emoji.APIName() != "🤔" {
		err = a.dg.MessageReactionRemove(Config.DiscordConfig.ChannelID, msg.ID, "🤔", discId)
		if err != nil {
			log.Err(err).Str("Reaction", "🤔").Msg("Failed to remove reaction")
		}
	}
	if r.Emoji.APIName() != "👎" {
		err = a.dg.MessageReactionRemove(Config.DiscordConfig.ChannelID, msg.ID, "👎", discId)
		if err != nil {
			log.Err(err).Str("Reaction", "👎").Msg("Failed to remove reaction")
		}
	}
}

func (a *Api) registerUserTemplate(c *gin.Context) {
	name := c.PostForm("name")
	discID := c.PostForm("discID")
	pass := c.PostForm("password")

	formData := gin.H{"Name": name, "DiscID": discID, "Password": pass}
	errors := map[string]string{}
	r, _ := regexp.Compile("[^a-zA-Z0-9]")
	if name == "" {
		errors["name"] = "name is required"
	} else if r.MatchString(name) {
		errors["password"] = "name must be alphanumeric"
	}
	if discID == "" {
		errors["discID"] = "discID is required"
	}
	if pass == "" {
		errors["password"] = "password is required"
	} else if r.MatchString(pass) {
		errors["password"] = "password must be alphanumeric"
	}

	player := Player{Name: name}
	duplicate, err := a.db.NewSelect().Model(&player).Where("name = ?", player.Name).Exists(a.ctx)
	if err != nil {
		log.Err(err).Msg("Failed to check DB for existing username")
		formData["ServerError"] = "Not your fault, server is cooked. Try again?"
		c.HTML(http.StatusOK, "partials/register.html", formData)
		return
	}
	if duplicate {
		errors["name"] = "name is taken"
	}
	if len(errors) > 0 {
		formData["Errors"] = errors
		log.Debug().Any("errors", errors).Msg("form data didn't meet validation")
		c.HTML(http.StatusOK, "partials/register.html", formData)
		return
	}

	bytes, err := bcrypt.GenerateFromPassword([]byte(c.PostForm("password")), bcrypt.DefaultCost)
	if err != nil {
		log.Err(err).Msg("failed to hash password")
		formData["ServerError"] = err.Error()
		c.HTML(http.StatusOK, "partials/register.html", formData)
		return
	}
	sessionId, err := GenerateRandomState()
	if err != nil {
		log.Err(err).Msg("failed to generate session id")
		formData["ServerError"] = err.Error()
		c.HTML(http.StatusOK, "partials/register.html", formData)
		return
	}

	player = Player{Name: name, DiscordID: discID, Password: string(bytes), SessionId: sessionId}
	err = a.db.NewSelect().Model(&player).Where("discord_id = ?", player.DiscordID).Scan(a.ctx)
	if err != nil {
		player.VerificationCode = uuid.NewString()
		_, err := a.db.NewInsert().Model(&player).Exec(a.ctx)
		if err != nil {
			log.Err(err).Msg("failed to create new player")
			formData["ServerError"] = err.Error()
			c.HTML(http.StatusOK, "partials/register.html", formData)
			return
		}
	} else {
		// update the player's verification code to enable them to re-login
		player.VerificationCode = uuid.NewString()
		_, err = a.db.NewUpdate().Model(&player).Where("discord_id = ?", player.DiscordID).Exec(a.ctx)
		if err != nil {
			log.Err(err).Msg("failed to update player with new verification code")
			formData["ServerError"] = err.Error()
			c.HTML(http.StatusOK, "partials/register.html", formData)
			return
		}
	}

	channel, err := a.dg.UserChannelCreate(discID)
	if err != nil {
		log.Err(err).Any("player", player).Msg("failed to create private channel to send verification code")
		formData["ServerError"] = "Invalid Discord ID"
		c.HTML(http.StatusOK, "partials/register.html", formData)
		return
	}
	_, err = a.dg.ChannelMessageSend(
		channel.ID,
		fmt.Sprintf("Here is your verification code from the PlayDate application!\n`%s`\nUse this to complete your signup/login.", player.VerificationCode),
	)
	if err != nil {
		log.Err(err).Any("player", player).Msg("failed to send verification code to user directly")
		formData["ServerError"] = err.Error()
		c.HTML(http.StatusOK, "partials/register.html", formData)
		return
	}

	formData["Verifying"] = true
	c.HTML(http.StatusOK, "partials/register.html", formData)
}

func (a *Api) verifyPlayer(c *gin.Context) {
	name := c.PostForm("name")
	discID := c.PostForm("discID")
	verificationCode := c.PostForm("verificationCode")

	formData := gin.H{"Name": name, "DiscID": discID, "Verifying": true, "VerificationCode": verificationCode}
	errors := map[string]string{}
	if name == "" {
		errors["name"] = "name is required"
	}
	if discID == "" {
		errors["discID"] = "discID is required"
	}
	if verificationCode == "" {
		errors["verificationCode"] = "verificationCode is required"
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
		log.Err(err).Any("player", player).Msg("failed to find player")
		formData["ServerError"] = err.Error()
		c.HTML(http.StatusOK, "partials/register.html", formData)
		return
	}

	// check if the verifcation codes match, otherwise reroute back to registration
	if player.VerificationCode != verificationCode {
		formData["VerificationCode"] = "" // required to show error message
		errors["verificationCode"] = "invalid verification code provided"
		formData["Errors"] = errors
		log.Debug().Any("player", player).Any("errors", errors).Msg("provided verification code didn't match our records")
		c.HTML(http.StatusOK, "partials/register.html", formData)
		return
	}

	a.createPlayDateCookie(c, player.SessionId)
}

func (a *Api) fetchPoppedDates() {
	log.Info().Msg("Any PlayDates??")
	state := gin.H{"Errors": map[string]string{}}

	now := time.Now()
	playdates := []*PlayDate{}
	err := a.db.NewSelect().
		Model(&playdates).
		Relation("Owner").
		Relation("Attendances").
		Relation("Attendances.Player"). // NOTE: this will prefetch the nested attendance relationship's player relationship :fire:
		Where("date <= ?", now.Format("2006-01-02T15:04")).
		Where("status = ?", PlayDateStatusPending).
		Scan(a.ctx)
	if err != nil {
		log.Error().Err(err).Msg("Watch is Kill")
		state["ServerError"] = "Watch Dead"
	}

	log.Info().Any("playdates", playdates).Msg("Found the following playdates")
	for _, playdate := range playdates {
		atAttendingPlayers := ""
		for _, attendance := range playdate.Attendances {
			if attendance.Attending == AttendanceNo {
				continue
			}
			atAttendingPlayers = atAttendingPlayers + fmt.Sprintf("<@%s>", attendance.Player.DiscordID)
		}
		msg := fmt.Sprintf("Playdate %s created by <@%s> is happening now! Make sure to join :video_game:!\n%s", playdate.Game, playdate.Owner.DiscordID, atAttendingPlayers)
		_, err = a.dg.ChannelMessageSend(Config.DiscordConfig.ChannelID, msg)
		if err != nil {
			log.Err(err).Any("playdate", playdate).Msg("failed to send message for playdate")
		}
		// mark a playdate as done if its "popped"
		playdate.Status = PlayDateStatusDone
		_, err = a.db.NewUpdate().Model(playdate).WherePK().Exec(a.ctx)
		if err != nil {
			log.Err(err).Any("playdate", playdate).Msg("failed to update playdate status")
		}
		log.Info().Any("playdate", playdate).Str("notification", msg).Msg("sent notification for playdate starting")
	}
}

func (a *Api) findPlayerFromCookie(c *gin.Context) (*Player, error) {
	cookie, err := c.Cookie("playdate")
	if err != nil {
		msg := "failed to parse user's cookie"
		log.Err(err).Msg(msg)
		return nil, errors.New(msg)
	}

	player := &Player{SessionId: cookie}
	err = a.db.NewSelect().Model(player).Where("session_id = ?", player.SessionId).Scan(c.Request.Context())
	if err != nil {
		// if the given player id doesn't exist just return the called to the home page
		msg := "failed to find the player from their cookie"
		log.Err(err).Str("cookie", cookie).Any("player", player).Msg(msg)
		return nil, errors.New(msg)
	}
	return player, nil
}

func (a *Api) goToRegisterUser(c *gin.Context) {
	state := gin.H{}
	state["ServerError"] = nil
	c.HTML(http.StatusOK, "partials/register.html", state)
}

func (a *Api) goToLogin(c *gin.Context) {
	state := gin.H{}
	state["ServerError"] = nil
	c.HTML(http.StatusOK, "partials/login.html", state)
}

func (a *Api) userLogin(c *gin.Context) {
	name := c.PostForm("name")
	pass := c.PostForm("password")

	formData := gin.H{"Name": name, "Password": pass}
	errors := map[string]string{}
	if name == "" {
		errors["name"] = "name is required"
	}
	if pass == "" {
		errors["pass"] = "password is required"
	}
	if len(errors) > 0 {
		formData["Errors"] = errors
		log.Debug().Any("errors", errors).Msg("form data didn't meet validation")
		c.HTML(http.StatusOK, "partials/login.html", formData)
		return
	}

	player := &Player{Name: name}
	err := a.db.NewSelect().Model(player).Where("name = ?", player.Name).Scan(a.ctx)
	if err != nil {
		log.Err(err).Any("player", player).Msg("failed to find player")
		formData["ServerError"] = "User doesn't exist"
		c.HTML(http.StatusOK, "partials/login.html", formData)
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(player.Password), []byte(pass))
	if err != nil {
		log.Err(err).Any("player", player).Msg("Invalid Password")
		formData["ServerError"] = "Invalid Password"
		c.HTML(http.StatusOK, "partials/login.html", formData)
		return
	}

	newSessionId, err := GenerateRandomState()
	if err != nil {
		log.Err(err).Msg("failed to generate session id")
		formData["ServerError"] = err.Error()
		c.HTML(http.StatusOK, "partials/register.html", formData)
		return
	}
	player.SessionId = newSessionId
	_, err = a.db.NewUpdate().Model(player).WherePK().Exec(a.ctx)
	if err != nil {
		log.Err(err).Msg("failed to update users session id")
		formData["ServerError"] = err.Error()
		c.HTML(http.StatusOK, "partials/register.html", formData)
		return
	}

	a.createPlayDateCookie(c, newSessionId)
}

// TODO: This isn't working
func (a *Api) userLogout(c *gin.Context) {
	// TODO: delete existing session id
	c.SetCookie("playdate", "", -1, "/", "", false, true)
	if c.Request.Header.Get("HX-Request") != "" {
		c.Header("HX-Location", "/")
	} else {
		c.Redirect(http.StatusMovedPermanently, "/")
	}
}

// create cookie and redirect to index
func (a *Api) createPlayDateCookie(c *gin.Context, sessionId string) {
	c.SetCookie("playdate", string(sessionId), 2000000, "/", "", false, true)
	if c.Request.Header.Get("HX-Request") != "" {
		c.Header("HX-Location", "/")
	} else {
		c.Redirect(http.StatusMovedPermanently, "/")
	}
}

func (a *Api) handleOAuthLogin(c *gin.Context) {
	authURL, err := DiscordOAuthLogin(a)
	if err != nil {
		log.Err(err).Msg("failed to oauth login")
		return
	}

	// Redirect the user's browser to Discord's authorization page
	log.Info().Str("discordAuthUrl", authURL).Msg("Redirecting user to discord oauth")
	c.Redirect(http.StatusFound, authURL)
	log.Info().Str("authURL", authURL).Msg("Redirecting user to Discord for authorization")
}

func (a *Api) handleOAuthCallback(c *gin.Context) {
	player, err := DiscordOAuthCallback(c, a)
	if err != nil {
		log.Err(err).Msg("failed oauth callback")
		return
	}

	a.createPlayDateCookie(c, player.SessionId)
}

func (a *Api) healthCheck(c *gin.Context) {
	// Check PostgreSQL connection
	err := a.db.Ping()
	if err != nil {
		log.Error().Err(err).Msg("PostgreSQL health check failed")
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "unhealthy",
			"message": "PostgreSQL connection failed",
			"error":   err.Error(),
		})
		return
	}

	// Check Discord connection
	_, err = a.dg.User("@me")
	if err != nil {
		log.Error().Err(err).Msg("Discord health check failed")
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "unhealthy",
			"message": "Discord connection failed",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"message": "All services are healthy",
	})
}

func getGithubReleaseNotes() (g GitHubRelease) {
	url := "https://api.github.com/repos/colehassett/playdate/releases/latest"

	// Create a new HTTP client with a timeout
	client := &http.Client{Timeout: 10 * time.Second}

	// Make the GET request to the GitHub API
	resp, err := client.Get(url)
	if err != nil {
		log.Err(err).Msg("Unable to retrieve github release notes")
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Err(err).Msg("Unable to read github response")
		return
	}

	var release GitHubRelease
	err = json.Unmarshal(body, &release)
	if err != nil {
		log.Err(err).Msg("Unable to unmarshal response body into githubrelease object")
		return
	}

	return release
}

func (a *Api) sendPatchNotes() {
	// get release notes
	release := getGithubReleaseNotes()

	log.Debug().Any("Release", release).Msg("Github Release")

	releaseBody := strings.ReplaceAll(release.Body, "@ColeHassett", "<@108736074557239296>")
	releaseBody = strings.ReplaceAll(releaseBody, "@colinthatcher", "<@128629520742744065>")

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("🤯 New PlayDate Release: %s 🤯", release.Name),
		Description: getFirstNRunes(releaseBody, 4096),
		Color:       0xfadde6,
		Timestamp:   release.PublishedAt.Format(time.RFC3339), // Discord expects ISO 8601 for timestamp
	}

	_, err := a.dg.ChannelMessageSendEmbed(Config.DiscordConfig.ChannelID, embed)
	if err != nil {
		log.Err(err).Msg("Failed to send patch notes embed")
	}
}

// In case the string is too long for the embed somehow
func getFirstNRunes(s string, n int) string {
	body := strings.Split(s, "\r")
	changelog := strings.Join(body[len(body)-1:], " ")
	nolog := strings.Join(body[:len(body)-1], " ")

	runes := []rune(nolog)
	size := n - (len(changelog) + 4)
	if len(runes) > size {
		return string(runes[:size]) + fmt.Sprintf("\n...%s", changelog)
	}
	return s
}
