package bot

import (
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"

	"github.com/brfrs/Discord-ACM-Bot-Go/pkg/bot/cmds"
)

// Env vars for the app to use
const (
	ENV_APP_ID         = "ACM_APP_ID"
	ENV_APP_PUBLIC_KEY = "ACM_APP_PUBLIC_KEY"
	ENV_TOKEN_ID       = "ACM_BOT_TOKEN"
	ENV_PORT           = "ACM_BOT_PORT"
	ENV_KEY_FILE       = "ACM_BOT_KEY_FILE"
	ENV_CERT_FILE      = "ACM_BOT_CERT_FILE"
)

// Private constants for discord interaction signatures
const (
	sig_header       = "X-Signature-Ed25519"
	timestamp_header = "X-Signature-Timestamp"
)

var basicCmds = []cmds.CmdData{
	{
		Name:        "register",
		Description: "Register your LeetCode username with your user in this Discord Server.",
		Opts: []cmds.AppCmdOption{
			{
				Name:        "username",
				Description: "LeetCode username. Yours, ideally.",
				Required:    true,
				Type:        3,
				Choices:     nil,
			},
		},
	},
	{
		Name:        "finished",
		Description: "Declare to the world (and the bot) that you've finished the LeetCode question from the challenge. Reap your rewards.",
		Opts:        nil,
	},
}

type Bot struct {
	appId        string
	token        string
	appPublicKey []byte
	started      bool
	port         int
}

func (bot *Bot) New() error {
	var err error
	bot.appId = os.Getenv(ENV_APP_ID)
	bot.token = os.Getenv(ENV_TOKEN_ID)
	bot.appPublicKey, err = hex.DecodeString(os.Getenv(ENV_APP_PUBLIC_KEY))

	if err != nil {
		return err
	}

	bot.port, err = strconv.Atoi(os.Getenv(ENV_PORT))

	if err != nil {
		return err
	}

	bot.started = true

	if bot.appId == "" {
		return fmt.Errorf("missing env var %s", ENV_APP_ID)
	}

	if bot.token == "" {
		return fmt.Errorf("missing env var %s", ENV_TOKEN_ID)
	}

	if len(bot.appPublicKey) != ed25519.PublicKeySize {
		return fmt.Errorf("bot public key is not ed25519 public key size. Size of key: %d", len(bot.appPublicKey))
	}

	if err != nil {
		return err
	}

	return nil
}

func verifyInteraction(bot *Bot, r *http.Request) (bool, error) {
	sigEntry := r.Header[sig_header]
	if sigEntry == nil || len(sigEntry) != 1 {
		return false, fmt.Errorf("request error: signature header missing or unexpected number of entries.")
	}
	sig, err := hex.DecodeString(sigEntry[0])

	if err != nil {
		return false, err
	}

	if len(sig) != ed25519.PrivateKeySize {
		return false, fmt.Errorf("request error: signature size doesn't match ed25519 private key size. Size of key: %d", len(sig))
	}

	timestampEntry := r.Header[timestamp_header]
	if timestampEntry == nil || len(timestampEntry) != 1 {
		return false, fmt.Errorf("request error: timestamp header missing or unexpected number of entries.")
	}
	timestamp := timestampEntry[0]

	fmt.Printf("Sig: %s\n", sigEntry[0])
	fmt.Printf("Timestamp: %s\n", timestamp)

	body, err := io.ReadAll(r.Body)

	if err != nil {
		return false, err
	}

	message := append([]byte(timestamp), body...)
	return ed25519.Verify(bot.appPublicKey, message, sig), nil
}

func (bot *Bot) InitGuild(guildId string) {
	cmds.RegisterCommands(basicCmds, bot.appId, guildId, bot.token)
}

func (bot *Bot) Serve() error {
	if !bot.started {
		return fmt.Errorf("Bot not init'd")
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			fmt.Printf("Error parsing form: %v\n", err)
			return
		}
		fmt.Printf("Request received. Method: %s\n", r.Method)
		verified, err := verifyInteraction(bot, r)

		if err != nil {
			fmt.Printf("Error encountered trying to verify interaction: %v\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if !verified {
			fmt.Println("Error encountered trying to verify message. Returning 401")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		encoder := json.NewEncoder(w)
		responseData := map[string]int{
			"type": 1,
		}

		encoder.Encode(responseData)
	})

	return http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", bot.port), nil)
}
