package main

import (
	"log"
	"noclouds/config"
	"strconv"
	"time"

	"github.com/go-co-op/gocron"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	badWeatherAlert   = "Сьогодні хмарно 🥺"
	goodWeatherAlert  = "Хороша погода сьогодні! 🥳"
	startMessage      = "Розпочнімо. Тицяй кнопку."
	badRequestMessage = "Не розумію..."
	noGoodWeather7d   = "Хмарно наступні 7 днів 🥺"
)

// mono() returns monospaced escaped Markdown
func mono(s string) string {
	return "`" + tgbotapi.EscapeText("MarkdownV2", s) + "`"
}

// checkNext24H() is cron job which monitors good/bad weather next 24 hours
func checkNext24H(bot *tgbotapi.BotAPI, config config.Config) {
	chatID, err := strconv.Atoi(config.TelegramChatID)
	if err != nil {
		log.Println("ERROR: cannot convert CHAT_ID value to int")
	}

	msg := tgbotapi.NewMessage(int64(chatID), "")
	msg.ChatID = int64(chatID)
	msg.ParseMode = "MarkdownV2"

	s := gocron.NewScheduler(time.UTC)
	var state State
	state.Init(config.StateFilePath)

	_, err = s.Cron(config.CronExpression).Do(func() {
		log.Println("INFO: starting cron job")
		startPoints := getAllStartPoints(config)
		next24HStartPoints := startPoints.next24H()

		if len(next24HStartPoints) > 0 && !state.isGood(config.StateFilePath) {
			log.Println("INFO: good weather in the next 24h. Sending message")
			msg.Text = mono(goodWeatherAlert + "\n\n" + next24HStartPoints.setMoonIllumination().Print())

			if _, err := bot.Send(msg); err != nil {
				log.Println("ERROR: can't send message to Telegram", err)
			}
			state.Set(true, config.StateFilePath)
		} else if len(next24HStartPoints) == 0 && state.isGood(config.StateFilePath) {
			log.Println("INFO: No more good forecast for the next 24h. Sending message")
			msg.Text = mono(badWeatherAlert)

			if _, err := bot.Send(msg); err != nil {
				log.Println("ERROR: can't send message to Telegram", err)
			}
			state.Set(false, config.StateFilePath)
		} else {
			log.Println("INFO: No changes in weather forecast for the next 24 hours")
		}
	})
	if err != nil {
		log.Println(err)
	}

	s.StartAsync()
}

// authChat() makes sure no one else excet me can interact with this bot
func authChat(chatID int64, allowedChatID string) bool {
	chatIDString := strconv.Itoa(int(chatID))
	return chatIDString == allowedChatID
}

// handleChat() is telegram bot handler for chat interactions
func handleChat(bot *tgbotapi.BotAPI, update tgbotapi.Update, config config.Config) {
	if !authChat(update.Message.Chat.ID, config.TelegramChatID) {
		log.Printf("Chat ID %d unauthorized. Exit.\n", update.Message.Chat.ID)
		return
	}

	// Set keyboard
	var numericKeyboard = tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Прогноз на 7 днів"),
		),
	)
	// Listen for updates
	if update.Message != nil {
		log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")
		msg.ReplyMarkup = numericKeyboard
		msg.ParseMode = "MarkdownV2"

		if update.Message.IsCommand() && update.Message.Command() == "start" {
			msg.Text = mono(startMessage)
		} else if update.Message.Text == "Прогноз на 7 днів" {
			forecast := getAllStartPoints(config).setMoonIllumination().Print()
			if forecast == "" {
				msg.Text = mono(noGoodWeather7d)
			} else {
				msg.Text = mono(forecast)
			}
		} else {
			msg.Text = mono(badRequestMessage)
		}

		log.Println("INFO: sending message to Telegram")
		if _, err := bot.Send(msg); err != nil {
			log.Println("ERROR: cannot send message", err)
		}
	}
}
