package interactions

type Member struct {
}

type Interaction struct {
	Id            string `json:"id"`
	ApplicationID string `json:"application_id"`
	Type          int    `json:"type"`
	//data to do
	GuildId   string  `json:"guild_id"`
	ChannelID string  `json:"channel_id"`
	Member    *Member `json:"member,omitempty"`
	// User probably not going to support for now
	Token   string `json:"token"`
	Version int    `json:"version"`
	// Message probably not going to implement for now
}

type Response struct {
	Type int `json:"type"`
	// data to do
}

type SlashCommands interface {
	do(input Interaction) error
}
