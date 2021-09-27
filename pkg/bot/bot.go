package bot

import (
	"fmt"
	"net/http"
	"os"

	"github.com/brfrs/Discord-ACM-Bot-Go/pkg/bot/cmds"
)

const ENV_APP_ID = "ACM_APP_ID"
const ENV_TOKEN_ID = "ACM_BOT_TOKEN"

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
	Port    int
}

func (bot *Bot) New(port int) {
	bot.appId = os.Getenv(ENV_APP_ID)
	bot.token = os.Getenv(ENV_TOKEN_ID)
	bot.Port = port
	bot.started = false
}

func (bot *Bot) InitGuild(guildId string) {
	cmds.RegisterCommands(basicCmds, bot.appId, guildId, bot.token)
	bot.started = true
}

func (bot *Bot) Serve() error {
	if !bot.started {
		return fmt.Errorf("Bot not init'd")
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("received request")

	})

	return http.ListenAndServe(fmt.Sprintf(":%d", bot.Port), nil)
}
