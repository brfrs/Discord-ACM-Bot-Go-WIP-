package bot

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	html2md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/Netflix/go-env"
	"github.com/brfrs/Discord-ACM-Bot-Go/pkg/leetcode"
	"github.com/jackc/pgx/v4"
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

// Ugly hack to get unmarshal into a ed25519 way
type PublicKey ed25519.PublicKey

func (pubkey *PublicKey) UnmarshalEnvironmentValue(data string) error {
	d, err := hex.DecodeString(data)
	*pubkey = d
	return err
}

type Bot struct {
	AppID        string    `env:"ACM_APP_ID,required=true"`
	Token        string    `env:"ACM_BOT_TOKEN,required=true"`
	AppPublicKey PublicKey `env:"ACM_APP_PUBLIC_KEY,required=true"`
	Port         int       `env:"ACM_BOT_PORT,default=6267"`
	DbUri        string    `env:"ACM_BOT_DB_URI,required=true"`
	DB           *pgx.Conn
	CmdMap       CmdMap
	Started      bool
	done         chan bool
}

func New() (Bot, error) {
	var bot Bot
	if _, err := env.UnmarshalFromEnviron(&bot); err != nil {
		return Bot{}, err
	}

	bot.CmdMap = make(CmdMap)

	var err error
	if bot.DB, err = pgx.Connect(context.Background(), bot.DbUri); err != nil {
		return Bot{}, err
	}

	if len(bot.AppPublicKey) != ed25519.PublicKeySize {
		return Bot{}, fmt.Errorf("bot public key is not ed25519 public key size. Size of key: %d", len(bot.AppPublicKey))
	}

	// It is dumb of me to make this the way CmdHandlers are registered to the bot
	if err := bot.RegisterGlobalCmds(GlobalCmds); err != nil {
		return Bot{}, err
	}

	guilds, err := bot.getAllGuilds()
	if err != nil {
		return Bot{}, err
	}

	for _, guild := range guilds {
		err = bot.RegisterGuildCmds(GuildCmds, guild)

		if err != nil {
			return Bot{}, err
		}
	}

	if err := bot.GetProblems(); err != nil {
		return Bot{}, err
	}

	if err := bot.PostDailiesToChannels(true); err != nil {
		return Bot{}, err
	}

	go bot.DailyPosting()

	InfoLogger.Println("Bot init'd")
	bot.Started = true
	return bot, nil
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
	return ed25519.Verify([]byte(bot.AppPublicKey), message, sig), nil
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
	bot.DB.Close(context.Background())
}
