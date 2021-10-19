package interactions

import (
	"encoding/json"
	"fmt"
	"io"
)

// Incoming Types
const (
	PING          = 1
	SLASH_COMMAND = 2
)

// Outgoing Types
const (
	PONG = 1
)

func HandleInteraction(i Interaction, w io.Writer) error {
	var resp Response

	switch i.Type {
	case PING:
		resp.Type = PONG
	case SLASH_COMMAND:

	default:
		return fmt.Errorf("unrecognized interaction type: %d", i.Type)
	}

	data, err := json.Marshal(resp)

	if err != nil {
		return err
	}

	w.Write(data)
	return nil
}
