package bot

import (
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/brfrs/Discord-ACM-Bot-Go/pkg/bot/cmds"
)

const ENV_APP_ID = "ACM_APP_ID"
const ENV_TOKEN_ID = "ACM_BOT_TOKEN"
const ENV_PORT = "ACM_BOT_PORT"

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
	appId   string
	token   string
	started bool
	port    int
}

func (bot *Bot) New() error {
	var err error
	bot.appId = os.Getenv(ENV_APP_ID)
	bot.token = os.Getenv(ENV_TOKEN_ID)
	bot.port, err = strconv.Atoi(os.Getenv(ENV_PORT))
	bot.started = true

	if bot.appId == "" {
		return fmt.Errorf("missing env var %s", ENV_APP_ID)
	}

	if bot.token == "" {
		return fmt.Errorf("missing env var %s", ENV_TOKEN_ID)
	}

	if err != nil {
		return err
	}

	return nil
}

func (bot *Bot) InitGuild(guildId string) {
	cmds.RegisterCommands(basicCmds, bot.appId, guildId, bot.token)
}

func (bot *Bot) Serve() error {
	if !bot.started {
		return fmt.Errorf("Bot not init'd")
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("received request")

	})

	return http.ListenAndServe(fmt.Sprintf(":%d", bot.port), nil)
}
