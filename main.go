package main

import (
	"context"
	"fmt"
	"os"

	"github.com/brfrs/Discord-ACM-Bot-Go/pkg/bot"
	"github.com/jackc/pgx/v4"
)

func main() {
	bot.InitLoggers(bot.LOG_LEVEL_DEBUG, os.Stderr)
	var b bot.Bot
	conn, err := pgx.Connect(context.Background(), os.Getenv(bot.ENV_DB_URL))

	if err != nil {
		fmt.Printf("Error: \"%v\"\n", err)
		return
	}
	defer conn.Close(context.Background())

	if err := b.New(conn); err != nil {
		bot.ErrorLogger.Printf("Error: \"%v\"\n", err)
		return
	}

	defer b.End()
	err = b.Serve()

	bot.ErrorLogger.Printf("Main: \"%v\"", err)
}
