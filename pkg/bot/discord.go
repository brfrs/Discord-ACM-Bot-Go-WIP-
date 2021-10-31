package bot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const (
	CMD_TYPE_CHAT_INPUT = 1
	CMD_TYPE_USER       = 2
	CMD_TYPE_MESSAGE    = 3
)

const (
	CMD_OPTYPE_SUB_COMMAND       = 1
	CMD_OPTYPE_SUB_COMMAND_GROUP = 2
	CMD_OPTYPE_STRING            = 3
	CMD_OPTYPE_INTEGER           = 4
	CMD_OPTYPE_BOOLEAN           = 5
	CMD_OPTYPE_USER              = 6
	CMD_OPTYPE_CHANNEL           = 7
	CMD_OPTYPE_ROLE              = 8
	CMD_OPTYPE_MENTIONABLE       = 9
	CMD_OPTYPE_NUMBER            = 10
)

const (
	CHAN_TYPE_GUILD_TEXT           = 0
	CHAN_TYPE_DM                   = 1
	CHAN_TYPE_GUILD_VOICE          = 2
	CHAN_TYPE_GROUP_DM             = 3
	CHAN_TYPE_GUILD_CATEGORY       = 4
	CHAN_TYPE_GUILD_NEWS           = 5
	CHAN_TYPE_GUILD_STORE          = 6
	CHAN_TYPE_GUILD_NEWS_THREAD    = 10 // Gap is intentional, lol
	CHAN_TYPE_GUILD_PUBLIC_THREAD  = 11
	CHAN_TYPE_GUILD_PRIVATE_THREAD = 12
	CHAN_TYPE_GUILD_STAGE_VOICE    = 13
)

const (
	INT_TYPE_PING        = 1
	INT_TYPE_APP_COMMAND = 2
	INT_TYPE_MSG_COMP    = 3
)

const (
	RESP_TYPE_PONG                        = 1
	RESP_TYPE_CHANNEL_MSG_WITH_SOURCE     = 4
	RESP_TYPE_DEF_CHANNEL_MSG_WITH_SOURCE = 5
	RESP_DEF_UPDATE_MESSAGE               = 6
	RESP_UPDATE_MSG                       = 7
)

const (
	ALLOWED_MENTION_ROLES    = "roles"
	ALLOWED_MENTION_USERS    = "users"
	ALLOWED_MENTION_EVERYONE = "everyone"
)

type CmdChoice struct {
	Name string      `json:"name"`
	Val  interface{} `json:"value"`
}

type CmdHandler func(Interaction, *Bot) (InteractionCallback, error)

type CmdOption struct {
	CmdOptType int         `json:"type"`
	Name       string      `json:"name"`
	Desc       string      `json:"description"`
	Required   bool        `json:"required"`
	Choices    []CmdChoice `json:"choices,omitempty"`
	Opts       []CmdOption `json:"options,omitempty"`
	ChanTypes  []int       `json:"channel_type,omitempty"`
	Value      *string     `json:"value,omitempty"`
}

type Cmd struct {
	Type              int         `json:"type"`
	appID             string      `json:"application_id"`
	Name              string      `json:"name"`
	Desc              string      `json:"description"`
	Opts              []CmdOption `json:"options,omitempty"` // Only valid for cmd input
	DefaultPermission bool        `json:"default_permission"`
	//Ver               string      `json:"version"`
	Handler CmdHandler `json:"-"`
}

type UserObj struct {
	ID       string `json:"id"`
	User     string `json:"username"`
	Discrim  string `json:"discriminator"`
	IsBot    bool   `json:"bot"`
	IsSystem bool   `json:"system"`
	Email    string `json:"email"`
}

type Member struct {
	User  *UserObj `json:"user,omitempty"`
	Nick  string   `json:"nick"`
	Roles []string `json:"roles,omitempty"`
}

type Data struct {
	Id   string `json:"id"`
	Name string `json:"name"`
	Type int    `json:"type"`
	// skipping resolved lol
	Opts     []CmdOption `json:"options,omitempty"`
	CustomId string      `json:"custom_id"`
	CompType int         `json:"component_type"`
	// Skipping values
	TargetId string `json:"target_id"`
}

type Embed struct {
}

type Message struct {
	Content string  `json:"content"`
	TTS     bool    `json:"tts"`
	Embeds  []Embed `json:"embeds"`
}

type Interaction struct {
	Id            string   `json:"id"`
	ApplicationID string   `json:"application_id"`
	Type          int      `json:"type"`
	CmdData       *Data    `json:"data,omitempty"`
	GuildID       string   `json:"guild_id"`
	ChannelID     string   `json:"channel_id"`
	MemberInfo    *Member  `json:"member,omitempty"`
	User          *UserObj `json:"user,omitempty"`
	Token         string   `json:"token"`
	Version       int      `json:"version"`
	// Message probably not going to implement for now
}

type AllowedMentions struct {
	Parse        []string `json:"parse,omitempty"`
	Roles        []string `json:"roles,omitempty"`
	Users        []string `json:"users,omitempty"`
	RepliedUsers bool     `json:"replied_users,omitempty"`
}

// Requires custom marshalling
type CallbackData struct {
	// There is more to do here, but idgaf rn
	Content string `json:"content"`
}

type InteractionCallback struct {
	Type int `json:"type"`
	// data to do
	Data *CallbackData `json:"data"`
}

type MessageParams struct {
	Content string  `json:"content"`
	TTS     bool    `json:"tts"`
	Embeds  []Embed `json:"embeds,omitempty"`
	// we are skipping the rest of this for now.
}

type CmdMap = map[string]CmdHandler

const GLOBAL_APP_CMD_URL = "https://discord.com/api/v8/applications/%s/commands"
const GUILD_APP_CMD_URL = "https://discord.com/api/v8/applications/%s/guilds/%s/commands"
const CHANNEL_MSG_CREATE_URL = "https://discord.com/api/v8/channels/%s/messages"

func registerCmds(cmds []Cmd, url, appId, appToken string) error {
	for _, cmd := range cmds {
		cmd.appID = appId
		data, err := json.Marshal(cmd)
		DebugLogger.Printf("To send: %s", string(data))
		if err != nil {
			return err
		}

		DebugLogger.Printf("Sending command registration request to url: %s", url)

		req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(data))
		req.Header.Add("Authorization", fmt.Sprintf("Bot %s", appToken))
		req.Header.Add("Content-Type", "application/json")

		if err != nil {
			return err
		}

		resp, err := http.DefaultClient.Do(req)

		if err != nil {
			return err
		}

		if resp.StatusCode != 201 && resp.StatusCode != 200 {
			return fmt.Errorf("response to command creation not 200 nor 201. Response code: %d", resp.StatusCode)
		}
	}

	return nil
}

func RegisterGlobalCmds(cmds []Cmd, appId, appToken string) error {
	url := fmt.Sprintf(GLOBAL_APP_CMD_URL, appId)
	err := registerCmds(cmds, url, appId, appToken)

	if err != nil {
		return err
	}

	return nil
}

func RegisterGuildCmds(cmds []Cmd, appId, appToken, guildId string) error {
	url := fmt.Sprintf(GUILD_APP_CMD_URL, appId, guildId)
	err := registerCmds(cmds, url, appId, appToken)

	if err != nil {
		return err
	}

	return nil
}

func PostToChannel(channelID, appToken string, msg MessageParams) error {
	url := fmt.Sprintf(CHANNEL_MSG_CREATE_URL, channelID)

	data, err := json.Marshal(msg)

	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(data))
	req.Header.Add("Authorization", fmt.Sprintf("Bot %s", appToken))
	req.Header.Add("Content-Type", "application/json")

	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		return err
	}

	body, _ := io.ReadAll(resp.Body)
	DebugLogger.Printf(string(body))

	if resp.StatusCode != 201 && resp.StatusCode != 200 {
		return fmt.Errorf("response to msg creation not 200 nor 201. Response code: %d", resp.StatusCode)
	}

	return nil
}
