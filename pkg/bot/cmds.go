package bot

import (
	"fmt"
)

var SetupCmd = Cmd{
	Type:              CMD_TYPE_CHAT_INPUT,
	Name:              "setup",
	Desc:              "Registers this channel of this guild to have a daily challenge posted to it.",
	Opts:              nil,
	DefaultPermission: true,
	Handler: func(i Interaction, bot *Bot) (InteractionCallback, error) {
		err := bot.addNewChannel(i.GuildID, i.ChannelID)

		if err != nil {
			return InteractionCallback{}, err
		}

		fmt.Println("Register guild commands")
		err = RegisterGuildCmds(bot.CmdMap, []Cmd{FlexCmd, RegisterCmd}, bot.AppID, bot.Token, i.GuildID)
		if err != nil {
			return InteractionCallback{}, err
		}

		fmt.Println("Return response")
		return InteractionCallback{
			Type: RESP_TYPE_CHANNEL_MSG_WITH_SOURCE,
			Data: &CallbackData{
				Content: "Abandon all hope ye who enter here.",
			},
		}, err
	},
}

var RegisterCmd = Cmd{
	Type: CMD_TYPE_CHAT_INPUT,
	Name: "register",
	Desc: "Register your leetcode username with your name in this guild.",
	Opts: []CmdOption{
		{
			CmdOptType: CMD_OPTYPE_STRING,
			Name:       "uname",
			Desc:       "LeetCode username",
			Required:   true,
		},
	},

	DefaultPermission: true,
	Handler: func(i Interaction, bot *Bot) (InteractionCallback, error) {
		userID := i.MemberInfo.User.ID
		if userID == "" {
			return InteractionCallback{}, fmt.Errorf("expected valid user for this interaction: %+v", i)
		}

		leetUser := *i.CmdData.Opts[0].Value

		if err := bot.registerUser(userID, i.GuildID, leetUser); err != nil {
			return InteractionCallback{}, err
		}

		return InteractionCallback{
			Type: RESP_TYPE_CHANNEL_MSG_WITH_SOURCE,
			Data: &CallbackData{
				Content: "Welcome to the challenge! We offer free pizza.",
			},
		}, nil
	},
}

var FlexCmd = Cmd{
	Type:              CMD_TYPE_CHAT_INPUT,
	Name:              "flex",
	Desc:              "Prints your current score for this guild and whether you completed the current challenge or not.",
	Opts:              nil,
	DefaultPermission: true,
	Handler: func(i Interaction, bot *Bot) (InteractionCallback, error) {
		userID := i.MemberInfo.User.ID
		if userID == "" {
			return InteractionCallback{}, fmt.Errorf("expected valid user for this interaction: %+v", i)
		}

		res, err := bot.getScore(userID, i.GuildID)

		if err != nil {
			return InteractionCallback{}, err
		}

		var msg string
		if res == nil {
			msg = "You are not registerd in this guild, scrub."
		} else {
			msg = fmt.Sprintf("You have %d points.", *res)
		}

		return InteractionCallback{
			Type: RESP_TYPE_CHANNEL_MSG_WITH_SOURCE,
			Data: &CallbackData{
				Content: msg,
			},
		}, nil
	},
}

var GlobalCmds = []Cmd{
	SetupCmd,
}

var GuildCmds = []Cmd{
	RegisterCmd,
	FlexCmd,
}
