package bot

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/brfrs/Discord-ACM-Bot-Go/pkg/leetcode"
	"github.com/jackc/pgx/v4"
)

const (
	CHANNEL_TYPE_DAILY       = "daily"
	CHANNEL_TYPE_SET         = "set"
	CHANNEL_TYPE_ACTION_SACK = "action sack"
)

const (
	PICKING_METHOD_ANY    = "any"
	PICKING_METHOD_EASY   = "easy"
	PICKING_METHOD_MEDIUM = "medium"
	PICKING_METHOD_HARD   = "hard"
	PICKING_METHOD_NONE   = "none"
)

/*
	The future of this file *really* should be to move all the sql to postgre functions.

	Some other thoughts about this while I'm here... Wow, I don't remember as much useful
	sql from CS 122a as I would have hoped. I don't know if pgx transactions are atomic, i.e.
	having an insert statement dependent on a previous query result in the same transaction.
	Now that I think, this might be littered with race conditions. Next step is to definitely
	refactor most of this.
*/

func (bot *Bot) addNewChannel(guildID, channelID, channelType string) error {
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

	switch channelType {
	case CHANNEL_TYPE_DAILY:
		sql = "INSERT INTO daily_channel (channel_id, pick, current_prob) VALUES ($1::VARCHAR, $2, 0);"
		_, err = tx.Exec(context.Background(), sql, channelID, PICKING_METHOD_ANY)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unimplemented channel type %s", channelType)
	}

	tx.Commit(context.Background())
	return nil
}

func (bot *Bot) getDailyChannels() ([]string, error) {
	result := make([]string, 0)
	sql := "SELECT channel_id FROM daily_channel;"
	rows, err := bot.DB.Query(context.Background(), sql)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var channelID string
		err = rows.Scan(&channelID)

		if err != nil {
			return nil, err
		}

		result = append(result, channelID)
	}

	return result, nil
}

func (bot *Bot) getAllGuilds() ([]string, error) {
	result := make([]string, 0)
	sql := "SELECT guild_id FROM guild;"
	rows, err := bot.DB.Query(context.Background(), sql)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

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

func (bot *Bot) registerUser(userID, channelID, guildID, leetcodeUser string) error {
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
		SELECT guild_id, user_id FROM score WHERE guild_id=$1::VARCHAR AND user_id=$2::VARCHAR
	);`
	_, err = tx.Exec(context.Background(), sql, guildID, userID)

	if err != nil {
		return err
	}

	sql = `
	INSERT INTO daily_participant (user_id, channel_id)
	SELECT $1::VARCHAR, $2::VARCHAR WHERE NOT EXISTS (
		SELECT user_id, channel_id FROM daily_participant WHERE user_id=$1::VARCHAR AND channel_id=$2::VARCHAR
	) AND EXISTS (
		SELECT channel_id FROM daily_channel WHERE channel_id=$2::VARCHAR
	);
	`
	if _, err := tx.Exec(context.Background(), sql, userID, channelID); err != nil && err != pgx.ErrNoRows {
		return err
	}

	tx.Commit(context.Background())
	return nil
}

func (bot *Bot) getLeetCodeUser(userID string) (*string, error) {
	var res string
	err := bot.DB.QueryRow(context.Background(), "SELECT leetcode_user FROM member where user_id=$1;", userID).Scan(&res)

	if err == pgx.ErrNoRows {
		DebugLogger.Println("No entry found.")
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	return &res, nil
}

func (bot *Bot) getScore(userID, guildID string) (*int, error) {
	var res int
	err := bot.DB.QueryRow(context.Background(), "SELECT value FROM score WHERE user_id=$1 AND guild_id=$2;", userID, guildID).Scan(&res)

	if err == pgx.ErrNoRows {
		DebugLogger.Println("No entry found.")
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	return &res, nil
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

func (bot *Bot) enqueueProblem(channel_id, problemSlug string) error {
	tx, err := bot.DB.Begin(context.Background())

	if err != nil {
		return err
	}

	defer tx.Rollback(context.Background())

	// This will work for now, but is ultimately a bug. We need to allow users to submit again, even if the same question posted is the same as their last solved... Oh well, how often will we run into that.
	sql := `
	INSERT INTO schedule (prob_number, channel_id, problem_slug)
		SELECT COALESCE(
			(SELECT MAX(prob_number)+1 FROM schedule WHERE channel_id=$1::VARCHAR GROUP BY (channel_id)),
			0
		), $1::VARCHAR, $2::VARCHAR;`

	DebugLogger.Println("Trying to enqueue problem")

	if _, err := tx.Exec(context.Background(), sql, channel_id, problemSlug); err != nil {
		return err
	}

	tx.Commit(context.Background())
	return nil
}

type problemRow struct {
	Slug string
	Diff int
}

func (bot *Bot) getAllProblems() ([]problemRow, error) {
	sql := "SELECT slug, difficulty FROM problem;"

	res := make([]problemRow, 0)

	rows, err := bot.DB.Query(context.Background(), sql)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var prob problemRow
		err := rows.Scan(&prob.Slug, &prob.Diff)

		if err != nil {
			return nil, err
		}

		res = append(res, prob)
	}

	return res, nil
}

func (bot *Bot) pickAnyProblem() (problemRow, error) {
	probs, err := bot.getAllProblems()

	if err != nil {
		return problemRow{}, err
	}

	return probs[rand.Intn(len(probs))], nil
}

func (bot *Bot) getDailyProblem(channelID string, generateProb bool) (*problemRow, error) {
	tx, err := bot.DB.Begin(context.Background())

	if err != nil {
		return nil, err
	}

	defer tx.Rollback(context.Background())

	var pick string
	if generateProb {
		DebugLogger.Println("Updating daily channel current problem")
		sql := "SELECT pick FROM daily_channel WHERE channel_id=$1;"
		if err := tx.QueryRow(context.Background(), sql, channelID).Scan(&pick); err != nil {
			return nil, err
		}

		sql = `
		UPDATE daily_channel SET current_prob=(current_prob+1) 
		WHERE channel_id=$1::VARCHAR AND EXISTS (
			SELECT * 
			FROM schedule JOIN daily_channel ON schedule.prob_number=daily_channel.current_prob AND schedule.channel_id=daily_channel.channel_id
			WHERE schedule.channel_id=$1::VARCHAR
		);`

		if err := tx.QueryRow(context.Background(), sql, channelID).Scan(&pick); err != nil && err != pgx.ErrNoRows {
			return nil, err
		}
	}

	sql := `
	SELECT schedule.problem_slug, problem.difficulty
	FROM daily_channel JOIN schedule ON daily_channel.channel_id=schedule.channel_id AND daily_channel.current_prob=schedule.prob_number JOIN problem ON schedule.problem_slug=problem.slug
	WHERE daily_channel.channel_id=$1::VARCHAR;`

	var res problemRow
	err = tx.QueryRow(context.Background(), sql, channelID).Scan(&res.Slug, &res.Diff)

	DebugLogger.Printf("Slug: %s, Diff: %d, Pick: %s\n", res.Slug, res.Diff, pick)

	if err == pgx.ErrNoRows {
		if !generateProb {
			return nil, nil
		}

		switch pick {
		case PICKING_METHOD_NONE:
			return nil, nil

		case PICKING_METHOD_ANY:
			if res, err = bot.pickAnyProblem(); err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("error unimplemented problem picking scheme %s", pick)
		}

		if err = bot.enqueueProblem(channelID, res.Slug); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	tx.Commit(context.Background())
	return &res, nil
}

func (bot *Bot) markSolved(value int, userID, guildID, slug, channelID string) (*int, bool, error) {
	var res int
	tx, err := bot.DB.Begin(context.Background())

	if err != nil {
		return nil, false, err
	}

	defer tx.Rollback(context.Background())

	// I guess I'm assuming that user_id and channel_id exist here lel
	sql := `
	UPDATE daily_participant SET last_solved=$3::VARCHAR
	WHERE user_id=$1::VARCHAR AND channel_id=$2::VARCHAR AND (last_solved IS NULL OR last_solved!=$3::VARCHAR);`

	cmdTag, err := tx.Exec(context.Background(), sql, userID, channelID, slug)

	if err != nil {
		return nil, false, err
	}
	DebugLogger.Printf("Rows affected: %d\n", cmdTag.RowsAffected())
	if cmdTag.RowsAffected() != 1 {
		return nil, true, nil
	}

	sql = `
	UPDATE score SET value = value + $3
	WHERE user_id=$1::VARCHAR AND guild_id=$2::VARCHAR
	RETURNING value;`

	err = tx.QueryRow(context.Background(), sql, userID, guildID, value).Scan(&res)

	if err != nil {
		return nil, false, err
	}

	tx.Commit(context.Background())
	return &res, false, nil
}
