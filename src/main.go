package main

import (
	"log"
	"noclouds/config"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {
	// Load the configuration
	config.LoadConfig()

	// Create Bot instance
	bot, err := tgbotapi.NewBotAPI(config.AppConfig.TelegramBotToken)
	if err != nil {
		log.Panic(err)
	}
	log.Printf("INFO: Authorized on account %s", bot.Self.UserName)

	// Start 24h check in background
	checkNext24H(bot, config.AppConfig)
	log.Println("INFO: Background cron job activated")

	// Start Bot and process user input
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	log.Println("INFO: Bot started")
	for update := range updates {
		handleChat(bot, update, config.AppConfig)
	}
}
