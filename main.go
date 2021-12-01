package main

import (
	"os"

	"github.com/brfrs/Discord-ACM-Bot-Go/pkg/bot"
)

func main() {
	bot.InitLoggers(bot.LOG_LEVEL_DEBUG, os.Stderr)

	b, err := bot.New()
	if err != nil {
		bot.ErrorLogger.Printf("Error: \"%v\"\n", err)
		return
	}

	defer b.End()
	err = b.Serve()

	bot.ErrorLogger.Printf("Main: \"%v\"", err)
}
