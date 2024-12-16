package config

import (
	"os"
	"strconv"
)

// AppConfig is a global variable for configuration
var AppConfig Config

// Config holds all environment variables
type Config struct {
	TelegramBotToken       string
	TelegramChatID         string
	OpenMeteoApiEndpoint   string
	OpenMeteoRequestParams string
	StateFilePath          string
	MaxCloudCover          int64
	MaxWind                float64
	NightStartsAt          int
	NightEndsAt            int
	GoodWeatherWindow      int
	CronExpression         string
	Latitude               string
	Lognitude              string
}

// LoadConfig initializes AppConfig from environment variables
func LoadConfig() {
	AppConfig = Config{
		TelegramBotToken:       getEnv("TG_BOT_TOKEN", ""),
		TelegramChatID:         getEnv("CHAT_ID", ""),
		Latitude:               getEnv("LAT", ""),
		Lognitude:              getEnv("LON", ""),
		OpenMeteoApiEndpoint:   getEnv("API_ENDPOINT", "https://api.open-meteo.com/v1/forecast?"),
		OpenMeteoRequestParams: getEnv("REQUEST_PARAMS", "temperature_2m,cloud_cover_low,cloud_cover_mid,cloud_cover_high,wind_speed_10m,wind_gusts_10m"),
		StateFilePath:          getEnv("STATE_FILE_PATH", "state.txt"),
		MaxCloudCover:          toInt64(getEnv("MAX_CLOUD_COVER", "25")),
		MaxWind:                strToFloat(getEnv("MAX_WIND", "15")),
		NightStartsAt:          toInt(getEnv("NIGHT_STARTS_AT", "22")),
		NightEndsAt:            toInt(getEnv("NIGHT_ENDS_AT", "5")),
		GoodWeatherWindow:      toInt(getEnv("GOOD_WEATHER_WINDOW", "4")),
		CronExpression:         getEnv("CRON_EXPRESSION", "0 8,12,16,20 * * *"),
	}
}

// getEnv reads an environment variable or returns a default value
func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

// toInt64 converts a string to int and returns the value
func toInt(s string) int {
	if out, err := strconv.Atoi(s); err == nil {
		return out
	}
	return 0
}

// toInt64 converts a string to int64 and returns the value
func toInt64(s string) int64 {
	if out, err := strconv.Atoi(s); err == nil {
		return int64(out)
	}
	return 0
}

// strToFloat converts a string to float64 and returns the value
func strToFloat(input string) float64 {
	// Convert string to float64
	floatValue, _ := strconv.ParseFloat(input, 64)
	return floatValue
}
