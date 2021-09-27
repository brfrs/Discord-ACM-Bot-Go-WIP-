package cmds

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

const urlFormat = "https://discord.com/api/v8/applications/%s/guilds/%s/commands"

func getGuildCommandUrl(appId string, guildId string) string {
	return fmt.Sprintf(urlFormat, appId, guildId)

}

func RegisterCommands(cmds []CmdData, appId string, guildId string, botToken string) error {
	url := getGuildCommandUrl(appId, guildId)
	for i, cmdData := range cmds {
		cmd := AppCmd{
			i, // cmd id
			1, // cmd type, lets hardcode this for now
			appId,
			guildId,
			cmdData.Name,
			cmdData.Description,
			cmdData.Opts,
			true, // default permissions
		}
		body, err := json.Marshal(cmd)

		if err != nil {
			return err
		}

		req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))

		if err != nil {
			return err
		}

		req.Header = map[string][]string{
			"Authorization": {botToken},
		}

		resp, err := http.DefaultClient.Do(req)

		if err != nil {
			return err
		}

		if resp.StatusCode != 200 {
			return fmt.Errorf("RegisterCommands: response code %d != 200", resp.StatusCode)
		}
	}

	return nil
}
