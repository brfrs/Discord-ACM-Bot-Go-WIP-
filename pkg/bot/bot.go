package bot

import (
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	html2md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/brfrs/Discord-ACM-Bot-Go/pkg/leetcode"
	"github.com/jackc/pgx/v4"
)

// Env vars for the app to use
const (
	ENV_APP_ID         = "ACM_APP_ID"
	ENV_APP_PUBLIC_KEY = "ACM_APP_PUBLIC_KEY"
	ENV_TOKEN_ID       = "ACM_BOT_TOKEN"
	ENV_PORT           = "ACM_BOT_PORT"
	ENV_KEY_FILE       = "ACM_BOT_KEY_FILE"
	ENV_CERT_FILE      = "ACM_BOT_CERT_FILE"
	ENV_DB_URL         = "ACM_DB_URL"
)

var (
	DebugLogger   *log.Logger
	InfoLogger    *log.Logger
	WarningLogger *log.Logger
	ErrorLogger   *log.Logger
)

const (
	LOG_LEVEL_DEBUG   = 0
	LOG_LEVEL_INFO    = 1
	LOG_LEVEL_WARNING = 2
	LOG_LEVEL_ERROR   = 3
)

const DailyPostPeriod = 12 * time.Hour

func InitLoggers(logLevel int, outfile io.Writer) {
	if LOG_LEVEL_DEBUG >= logLevel {
		DebugLogger = log.New(outfile, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)
	}
	if LOG_LEVEL_INFO >= logLevel {
		InfoLogger = log.New(outfile, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	}
	if LOG_LEVEL_WARNING >= logLevel {
		WarningLogger = log.New(outfile, "WARNING: ", log.Ldate|log.Ltime|log.Lshortfile)
	}
	if LOG_LEVEL_ERROR >= logLevel {
		ErrorLogger = log.New(outfile, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
	}
}

// Private constants for discord interaction signatures
const (
	sig_header       = "X-Signature-Ed25519"
	timestamp_header = "X-Signature-Timestamp"
)

var DifficultyToColorCode = map[int]int{
	leetcode.DIFFICULTY_EASY:   44955,
	leetcode.DIFFICULTY_MEDIUM: 16758784,
	leetcode.DIFFICULTY_HARD:   16723284,
}

type Bot struct {
	AppID        string
	Token        string
	AppPublicKey []byte
	Started      bool
	Port         int
	CmdMap       CmdMap
	DB           *pgx.Conn
	done         chan bool
}

func (bot *Bot) New(conn *pgx.Conn) error {
	var err error
	bot.AppID = os.Getenv(ENV_APP_ID)
	bot.Token = os.Getenv(ENV_TOKEN_ID)
	bot.AppPublicKey, err = hex.DecodeString(os.Getenv(ENV_APP_PUBLIC_KEY))
	if err != nil {
		return err
	}

	bot.CmdMap = make(CmdMap)
	bot.DB = conn

	bot.Port, err = strconv.Atoi(os.Getenv(ENV_PORT))

	if err != nil {
		return err
	}

	if bot.AppID == "" {
		return fmt.Errorf("missing env var %s", ENV_APP_ID)
	}

	if bot.Token == "" {
		return fmt.Errorf("missing env var %s", ENV_TOKEN_ID)
	}

	if len(bot.AppPublicKey) != ed25519.PublicKeySize {
		return fmt.Errorf("bot public key is not ed25519 public key size. Size of key: %d", len(bot.AppPublicKey))
	}

	if err != nil {
		return err
	}

	// It is dumb of me to make this the way CmdHandlers are registered to the bot
	err = bot.RegisterGlobalCmds(GlobalCmds)

	if err != nil {
		return err
	}

	guilds, err := bot.getAllGuilds()

	if err != nil {
		return err
	}

	for _, guild := range guilds {
		err = bot.RegisterGuildCmds(GuildCmds, guild)

		if err != nil {
			return err
		}
	}

	err = bot.GetProblems()

	if err != nil {
		return err
	}

	err = bot.PostDailiesToChannels(true)

	if err != nil {
		return err
	}

	go bot.DailyPosting()

	InfoLogger.Println("Bot init'd")
	bot.Started = true
	return nil
}

func verifyInteraction(bot *Bot, header http.Header, body []byte) (bool, error) {
	sigEntry := header[sig_header]
	if sigEntry == nil || len(sigEntry) != 1 {
		return false, fmt.Errorf("request error: signature header missing or unexpected number of entries")
	}

	sig, err := hex.DecodeString(sigEntry[0])
	if err != nil {
		return false, err
	}

	if len(sig) != ed25519.PrivateKeySize {
		return false, fmt.Errorf("request error: signature size doesn't match ed25519 private key size. Size of key: %d", len(sig))
	}

	timestampEntry := header[timestamp_header]
	if timestampEntry == nil || len(timestampEntry) != 1 {
		return false, fmt.Errorf("request error: timestamp header missing or unexpected number of entries")
	}
	timestamp := timestampEntry[0]

	message := append([]byte(timestamp), body...)
	return ed25519.Verify(bot.AppPublicKey, message, sig), nil
}

func (bot *Bot) handleInteraction(i Interaction, w http.ResponseWriter) error {
	var resp InteractionCallback
	var err error

	switch i.Type {
	case INT_TYPE_PING:
		DebugLogger.Println("INT TYPE PING")
		resp.Type = RESP_TYPE_PONG
	case INT_TYPE_APP_COMMAND:
		DebugLogger.Println("INT TYPE APP_COMMAND")
		if i.CmdData == nil {
			return fmt.Errorf("found no data for slash command. Interaction: %+v", i)
		}
		if handler, ok := bot.CmdMap[i.CmdData.Name]; ok {
			resp, err = handler(i, bot)

			if err != nil {
				return err
			}
		} else {
			WarningLogger.Println("Unrecognized command name. Is this command registered with this bot?")
		}

	default:
		WarningLogger.Printf("INT TYPE UNKNOWN: %d\n", i.Type)
		return fmt.Errorf("unrecognized interaction type: %d", i.Type)
	}

	data, err := json.Marshal(resp)
	DebugLogger.Printf("To send: '%s'", string(data))
	if err != nil {
		return err
	}

	w.Write(data)
	return nil
}

func (bot *Bot) Serve() error {
	if !bot.Started {
		return fmt.Errorf("Bot not init'd")
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		InfoLogger.Println("Request received...")

		body, err := io.ReadAll(r.Body)
		DebugLogger.Printf("Body: %s", string(body))
		if r.Method != http.MethodPost {
			WarningLogger.Printf("Error with request method. Discord only sends POST requests.")
			w.WriteHeader(http.StatusBadRequest)
		}

		if err != nil {
			ErrorLogger.Printf("Error encountered while reading the body of the request: %v\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		verified, err := verifyInteraction(bot, r.Header, body)

		if err != nil {
			ErrorLogger.Printf("Error encountered trying to verify interaction: %v\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if !verified {
			WarningLogger.Println("message failed verification. Returning 401")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		var interaction Interaction
		err = json.Unmarshal(body, &interaction)
		DebugLogger.Printf("Interaction: %+v", interaction)
		if err != nil {
			ErrorLogger.Printf("Error unmarshalling interaction: %v\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Add("Content-Type", "application/json")
		w.Header().Add("Authorization", fmt.Sprintf("Bot %s", bot.AppID))
		err = bot.handleInteraction(interaction, w)
		if err != nil {
			ErrorLogger.Printf("Error with handling interaction: %v\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		DebugLogger.Printf("ResponseWriter: %+v", w)
	})

	InfoLogger.Printf("Starting to serve on port=%d...\n", bot.Port)
	return http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", bot.Port), nil)
}

func (bot *Bot) GetProblems() error {
	probs, err := leetcode.GetLeetCodeProblems()

	if err != nil {
		return err
	}

	return bot.addProblems(probs)
}

func (bot *Bot) AddHandlers(cmds []Cmd) {
	for _, cmd := range cmds {
		bot.CmdMap[cmd.Name] = cmd.Handler
	}
}

func (bot *Bot) RegisterGlobalCmds(cmds []Cmd) error {
	if err := RegisterGlobalCmds(cmds, bot.AppID, bot.Token); err != nil {
		return err
	}

	bot.AddHandlers(cmds)
	return nil
}

func (bot *Bot) RegisterGuildCmds(cmds []Cmd, guildID string) error {
	if err := RegisterGuildCmds(cmds, bot.AppID, bot.Token, guildID); err != nil {
		return err
	}

	bot.AddHandlers(cmds)
	return nil
}

func (bot *Bot) PostDailyToChannel(date, channel string, generateProb bool) error {
	DebugLogger.Printf("Attempting to post daily (%s) problem to channel: %s\n", date, channel)
	prob, err := bot.getDailyProblem(channel, generateProb)

	if err != nil {
		return err
	}

	if prob == nil {
		DebugLogger.Printf("No problem found.")
		return nil
	}

	probDesc, err := leetcode.GetProblemDesc(prob.Slug)

	if err != nil {
		return err
	}

	converter := html2md.NewConverter("", true, &html2md.Options{})

	md, err := converter.ConvertString(probDesc.Content)

	if err != nil {
		WarningLogger.Printf("html2md conversion failed for this string: %s", probDesc.Content)
		md = probDesc.Content // This is a little troll on my part
	}

	problemURL := leetcode.GetProblemURL(probDesc.Slug)
	color := DifficultyToColorCode[prob.Diff]

	msg := Message{
		Content: fmt.Sprintf("Daily Problem: %s", date),
		Embeds: []Embed{
			{
				Title: &probDesc.Title,
				Desc:  &md,
				URL:   &problemURL,
				Color: &color,
			},
		},
		TTS: false,
	}

	DebugLogger.Printf("Posting daily prob=%s to channel=%s\n", prob.Slug, channel)
	return PostToChannel(channel, bot.Token, msg)
}

func (bot *Bot) PostDailiesToChannels(generateProb bool) error {
	channels, err := bot.getDailyChannels()
	date := getDate()

	DebugLogger.Printf("Today is: %s", date)

	if err != nil {
		return err
	}

	for _, channel := range channels {
		if err := bot.PostDailyToChannel(date, channel, generateProb); err != nil {
			return err
		}
	}

	return nil
}

func (bot *Bot) DailyPosting() {
	tick := time.NewTicker(DailyPostPeriod)
	for {
		select {
		case <-bot.done:
			return
		case <-tick.C:
			err := bot.PostDailiesToChannels(true)

			if err != nil {
				ErrorLogger.Printf("Error encountered while posting daillies: %v\n", err)
			}
		}
	}
}

func (bot *Bot) End() {
	bot.done <- true
}
