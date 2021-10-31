package bot

import (
	"context"

	"github.com/brfrs/Discord-ACM-Bot-Go/pkg/leetcode"
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
	sql := "SELECT channel_id FROM channel;"
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

func (bot *Bot) getLeetCodeUser(userID string) (*string, error) {
	var res *string
	rows, err := bot.DB.Query(context.Background(), "SELECT leetcode_user FROM member where user_id=$1", userID)

	if err != nil {
		return nil, err
	}

	if rows.Next() {
		res = new(string)
		err = rows.Scan(res)

		if err != nil {
			return nil, err
		}
	} else {
		DebugLogger.Println("No entry found.")
	}

	return res, nil
}

func (bot *Bot) getScore(userID, guildID string) (*int, error) {
	var res *int
	rows, err := bot.DB.Query(context.Background(), "SELECT value FROM score WHERE user_id=$1 AND guild_id=$2;", userID, guildID)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	if rows.Next() { // Needs to be called to prep the values if there are any
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

func (bot *Bot) addProblems(probs []leetcode.Problem) error {
	tx, err := bot.DB.Begin(context.Background())

	if err != nil {
		return err
	}

	defer tx.Rollback(context.Background())

	// Possible deadlock?
	sql := `
	INSERT INTO problem (slug, title, total_accept, total_subs, difficulty) VALUES ($1::VARCHAR, $2::VARCHAR, $3, $4, $5)
	ON CONFLICT ON CONSTRAINT problem_pkey 
	DO
		UPDATE SET title=$2::VARCHAR, total_accept=$3, total_subs=$4, difficulty=$5;
	`

	for _, prob := range probs {
		if prob.PaidOnly {
			continue
		}

		tx.Exec(context.Background(), sql, prob.Stat.Slug, prob.Stat.Title, prob.Stat.TotalAccepts, prob.Stat.TotalSubmissions, prob.Difficulty.Level)
	}

	tx.Commit(context.Background())
	return nil
}

func (bot *Bot) getTodaysProblem(date, channelID string) (*string, error) {
	sql := `
	SELECT problem_slug FROM schedule WHERE channel_id=$1 AND channel_id=$2;
	`

	var res *string
	rows, err := bot.DB.Query(context.Background(), sql, date, channelID)

	if err != nil {
		return nil, err
	}

	if rows.Next() {
		res = new(string)
		err := rows.Scan(res)
		if err != nil {
			return nil, err
		}
	}

	return res, nil
}
