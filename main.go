package main

import (
	"fmt"

	"github.com/brfrs/Discord-ACM-Bot-Go/pkg/bot"
)

func main() {
	var bot bot.Bot

	if err := bot.New(); err != nil {
		fmt.Printf("Error: \"%v\"", err)
	}

	err := bot.Serve()

	fmt.Printf("Main: \"%v\"", err)
}
