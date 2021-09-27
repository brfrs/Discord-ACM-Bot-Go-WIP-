package cmds

type AppCmdChoice struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type AppCmdOption struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Required    bool           `json:"required"`
	Type        int            `json:"type"`
	Choices     []AppCmdChoice `json:"choices"`
}

type CmdData struct {
	Name        string
	Description string
	Opts        []AppCmdOption
}

type AppCmd struct {
	Id                 int            `json:"id"`
	Type               int            `json:"type,omitempty"`
	appId              string         `json:"application_id"`
	GuildId            string         `json:"guild_id,omitempty"`
	Name               string         `json:"name"`
	Description        string         `json:"description"`
	Options            []AppCmdOption `json:"options"`
	DefaultPermissions bool           `json:"default_permissions"`
}

type Interaction struct {
	Id    string `json:"id"`
	AppId string `json:"id"`
	Type  int    `json:"type"`
}
