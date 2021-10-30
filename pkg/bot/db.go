package bot

import (
	"context"
)

func (bot *Bot) addNewChannel(guildID, channelID string) error {
	tx, err := bot.DB.Begin(context.Background())
	if err != nil {
		return err
	}

	defer tx.Rollback(context.Background())

	sql := "INSERT INTO guild (guild_id) SELECT $1::VARCHAR WHERE NOT EXISTS (SELECT guild_id FROM guild WHERE guild_id=$1::VARCHAR);"
	_, err = tx.Exec(context.Background(), sql, guildID)

	if err != nil {
		return err
	}

	sql = "INSERT INTO channel (channel_id, guild_id) SELECT $1::VARCHAR, $2::VARCHAR WHERE NOT EXISTS (SELECT channel_id, guild_id FROM channel WHERE channel_id=$1::VARCHAR AND guild_id=$2::VARCHAR);"
	_, err = tx.Exec(context.Background(), sql, channelID, guildID)
	if err != nil {
		return err
	}

	tx.Commit(context.Background())
	return nil
}

func (bot *Bot) getAllChannels() ([]string, error) {
	result := make([]string, 0)
	sql := "SELECT guild_id FROM guild;"
	rows, err := bot.DB.Query(context.Background(), sql)

	defer rows.Close()

	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var guildID string
		err = rows.Scan(&guildID)

		if err != nil {
			return nil, err
		}

		result = append(result, guildID)
	}

	return result, nil
}

func (bot *Bot) registerUser(userID, guildID, leetcodeUser string) error {
	tx, err := bot.DB.Begin(context.Background())
	if err != nil {
		return err
	}

	defer tx.Rollback(context.Background())

	sql := `
	INSERT INTO member (user_id, leetcode_user) 
	SELECT $1::VARCHAR, $2::VARCHAR WHERE NOT EXISTS (
		SELECT user_id, leetcode_user FROM member WHERE user_id=$1::VARCHAR AND leetcode_user=$2::VARCHAR
	);
	`

	_, err = tx.Exec(context.Background(), sql, userID, leetcodeUser)

	if err != nil {
		return err
	}

	sql = `
	INSERT INTO score (guild_id, user_id, value) 
	SELECT $1::VARCHAR, $2::VARCHAR, 0 WHERE NOT EXISTS (
		SELECT guild_id, user_id, 0 FROM score WHERE guild_id=$1::VARCHAR AND user_id=$1::VARCHAR
	);`
	_, err = tx.Exec(context.Background(), sql, guildID, userID)

	if err != nil {
		return err
	}

	tx.Commit(context.Background())
	return nil
}

func (bot *Bot) getScore(userID, guildID string) (*int, error) {
	var res *int
	rows, err := bot.DB.Query(context.Background(), "SELECT value FROM score WHERE user_id=$1 AND guild_id=$2;", userID, guildID)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	if rows.Next() {
		res = new(int)
		err = rows.Scan(res)

		if err != nil {
			return nil, err
		}
	} else {
		DebugLogger.Println("No entry found.")
	}

	return res, nil
}
