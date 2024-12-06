package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/go-co-op/gocron"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	OpenMeteoEndpoint   = "https://api.open-meteo.com/v1/forecast?"
	OpenMeteoParameters = "temperature_2m,cloud_cover_low,cloud_cover_mid,cloud_cover_high,wind_speed_10m,wind_gusts_10m"

	// 0 - last message was about bad forecast, 1 - about good
	defaultStateFilePath = "state.txt"

	defaultMaxCloudCover = "25" // all sky covered - 100%
	defaultMaxWind       = "20" // km/h

	defaultNightStartHour   = "22"
	defaultNightEndHour     = "5"
	defaultNightHoursStreak = "4"

	defaultCronExpression = "0 8,12,16,20 * * *"

	badWeatherAlert   = "Сьогодні хмарно 🥺"
	goodWeatherAlert  = "Хороша погода сьогодні! 🥳"
	startMessage      = "Розпочнімо. Тицяй кнопку."
	badRequestMessage = "Не розумію..."
	noGoodWeather7d   = "Хмарно наступні 7 днів 🥺"
)

type OpenMeteoAPIResponse struct {
	Latitude             float64     `json:"latitude"`
	Longitude            float64     `json:"longitude"`
	GenerationtimeMS     float64     `json:"generationtime_ms"`
	UTCOffsetSeconds     int64       `json:"utc_offset_seconds"`
	Timezone             string      `json:"timezone"`
	TimezoneAbbreviation string      `json:"timezone_abbreviation"`
	Elevation            float64     `json:"elevation"`
	HourlyUnits          HourlyUnits `json:"hourly_units"`
	Hourly               Hourly      `json:"hourly"`
}

type Hourly struct {
	Time           []string  `json:"time"`
	Temperature2M  []float64 `json:"temperature_2m"`
	CloudCoverLow  []int64   `json:"cloud_cover_low"`
	CloudCoverMid  []int64   `json:"cloud_cover_mid"`
	CloudCoverHigh []int64   `json:"cloud_cover_high"`
	WindSpeed10M   []float64 `json:"wind_speed_10m"`
	WindGusts10M   []float64 `json:"wind_gusts_10m"`
}

type HourlyUnits struct {
	Time           string `json:"time"`
	Temperature2M  string `json:"temperature_2m"`
	CloudCoverLow  string `json:"cloud_cover_low"`
	CloudCoverMid  string `json:"cloud_cover_mid"`
	CloudCoverHigh string `json:"cloud_cover_high"`
	WindSpeed10M   string `json:"wind_speed_10m"`
	WindGusts10M   string `json:"wind_gusts_10m"`
}

type DataPoint struct {
	Time       time.Time
	LowClouds  int64
	MidClouds  int64
	HighClouds int64
	MoonIllum  int64
	WindSpeed  float64
	WindGusts  float64
}

type DataPoints []DataPoint

type State bool

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func toInt(s string) int {
	if out, err := strconv.Atoi(s); err == nil {
		return out
	}
	return 0
}

func toInt64(s string) int64 {
	if out, err := strconv.Atoi(s); err == nil {
		return int64(out)
	}
	return 0
}

// strToFloat converts a string to float64 and returns the value and an error
func strToFloat(input string) (float64, error) {
	// Convert string to float64
	floatValue, err := strconv.ParseFloat(input, 64)
	if err != nil {
		return 0, fmt.Errorf("Error: failed to convert string to float64: %w", err)
	}
	return floatValue, nil
}

// mono() returns monospaced escaped Markdown
func mono(s string) string {
	return "`" + tgbotapi.EscapeText("MarkdownV2", s) + "`"
}

// MoonIllumination calculates the Moon's illumination percentage for a given date.
func moonIllumination(date time.Time) float64 {
	const synodicMonth float64 = 29.53059      // Average length of a synodic month in days
	const newMoonReference float64 = 2451549.5 // Julian date for a known new moon (Jan 6, 2000 18:14 UTC)

	// Convert the date to Julian Day
	julianDate := dateToJulianDate(date)

	// Calculate days since known new moon
	daysSinceNewMoon := julianDate - newMoonReference

	// Normalize to the Moon phase cycle (0 to 1)
	moonPhase := math.Mod(daysSinceNewMoon/synodicMonth, 1.0)
	if moonPhase < 0 {
		moonPhase += 1.0
	}

	// Calculate illumination percentage
	illumination := (1.0 - math.Cos(2.0*math.Pi*moonPhase)) / 2.0 * 100.0

	return illumination
}

// dateToJulianDate converts a time.Time to Julian Date
func dateToJulianDate(date time.Time) float64 {
	year, month, day := date.Date()
	hour, min, sec := date.Clock()

	// If the month is January or February, adjust the year and month
	if month <= 2 {
		year--
		month += 12
	}

	// Calculate Julian Day Number (JDN)
	a := int(float64(year) / 100.0)
	b := 2 - a + int(float64(a)/4.0)
	jdn := int(365.25*float64(year)) + int(30.6001*float64(month+1)) + day + 1720994 + b

	// Add fractional day for the time of day
	fracDay := (float64(hour) + float64(min)/60.0 + float64(sec)/3600.0) / 24.0

	return float64(jdn) + fracDay
}

// isGood() returns true if Low, Mid and High clouds percentage is less than MAX_CLOUD_COVER
func (d DataPoint) isGood() bool {
	maxCloudCoverInt64 := toInt64(getEnv("MAX_CLOUD_COVER", defaultMaxCloudCover))
	maxWindInt64, _ := strToFloat(getEnv("MAX_WIND", defaultMaxWind))

	if d.HighClouds <= maxCloudCoverInt64 && d.MidClouds <= maxCloudCoverInt64 && d.LowClouds <= maxCloudCoverInt64 && d.WindSpeed <= maxWindInt64 && d.WindGusts <= maxWindInt64 {
		return true
	} else {
		return false
	}
}

// atNight() returns true if time is between NIGHT_START_HOUR and NIGHT_END_HOUR
func (d DataPoint) atNight() bool {
	nightStart := toInt(getEnv("NIGHT_START_HOUR", defaultNightStartHour))
	nightEnd := toInt(getEnv("NIGHT_END_HOUR", defaultNightEndHour))

	if d.Time.Hour() >= nightStart || d.Time.Hour() <= nightEnd {
		return true
	} else {
		return false
	}
}

// Good() returns DataPoints which are after "Now", are "Good" and within "Night" defined by NIGHT_START_HOUR and NIGHT_END_HOUR
func (dp DataPoints) Good() DataPoints {
	good := DataPoints{}
	for _, v := range dp {
		if v.Time.After(time.Now()) && v.isGood() && v.atNight() {
			good = append(good, v)
		}
	}

	return good
}

// onlyStartPoints() returns DataPoints correstonding to the beginning of NIGHT_HOURS_STREAK sets
// For example, if NIGHT_HOURS_STREAK = 4 then this algorythm will try to find all 4-hours long sets of points.
// Should be applied to "Good" points.
func (dp DataPoints) onlyStartPoints() DataPoints {
	onlyStartPoints := DataPoints{}
	hoursStreak := toInt(getEnv("NIGHT_HOURS_STREAK", defaultNightHoursStreak))

	i := 0
	for i < len(dp)-hoursStreak {
		sum := 0
		for j := 1; j <= hoursStreak; j++ {
			diff := dp[i+j].Time.Sub(dp[i+j-1].Time)
			sum += int(diff.Hours())
		}

		// sum > hoursStreak means interval between points is not 1 hour long.
		if sum > hoursStreak {
			i++
			continue
		} else {
			// This allows to exclude 2 sequential 4-hours streaks on the same night.
			if i > 0 && dp[i].Time.Sub(dp[i-1].Time).Hours() == 1 {
				i++
				continue
			}
			onlyStartPoints = append(onlyStartPoints, dp[i])
			i = i + hoursStreak
		}
	}

	return onlyStartPoints
}

// next24H() returns DataPoints within 24 hour range from now
func (dp DataPoints) next24H() DataPoints {
	lessThan24H := DataPoints{}
	now := time.Now()

	for _, point := range dp {
		diff := point.Time.Sub(now)
		if diff.Hours() < 24 {
			lessThan24H = append(lessThan24H, point)
		}
	}

	return lessThan24H
}

// setMoonPhase() sets MoonPhase value for point in DataPoints
func (dp DataPoints) setMoonIllumination() DataPoints {
	updatedPoints := DataPoints{}

	for _, point := range dp {
		point.MoonIllum = int64(moonIllumination(point.Time))
		updatedPoints = append(updatedPoints, point)
	}

	return updatedPoints
}

// Print() returns Markdown string which represents DataPoints
func (dp DataPoints) Print() string {
	out := ""
	for _, point := range dp {
		out += fmt.Sprintf("%3d%% | %4.1f | %s |%2d %2d %2d\n", point.MoonIllum, point.WindGusts, point.Time.Format("Mon - 02 15h"), point.LowClouds, point.MidClouds, point.HighClouds)
	}

	return out
}

// Init() goes to OpenMeteoEndpoint makes HTTPS request and stores result as OpenMeteoAPIResponse object
func (response *OpenMeteoAPIResponse) Init() {
	log.Println("INFO: Making request to Open-Meteo API and parsing response")
	client := &http.Client{}
	APIEndpoint := getEnv("API_ENDPOINT", OpenMeteoEndpoint)

	// Set paramenters
	params := url.Values{}
	params.Add("latitude", os.Getenv("LAT"))
	params.Add("longitude", os.Getenv("LON"))
	params.Add("hourly", OpenMeteoParameters)

	// Make request to Open-Meteo API
	req, err := http.NewRequest("GET", APIEndpoint+params.Encode(), nil)
	if err != nil {
		log.Println("ERROR: Couldn't create new Open-Meteo API request", err)
		return
	}

	parseFormErr := req.ParseForm()
	if parseFormErr != nil {
		log.Println(parseFormErr)
		return
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Println("ERROR:", err)
		return
	}

	// Read Response Body
	if resp.StatusCode != 200 {
		fmt.Println("ERROR: Open-Meteo API response code:", resp.Status)
		return
	}

	log.Println("INFO: Got API response", resp.Status)
	respBody, _ := io.ReadAll(resp.Body)

	// Save response as OpenMeteoAPIResponse object
	err = json.Unmarshal(respBody, response)
	if err != nil {
		log.Println("ERROR: cannot Unmarshal JSON", err)
		return
	}
}

// Points() return DataPoints object based on OpenMeteoAPIResponse fields
func (data OpenMeteoAPIResponse) Points() DataPoints {
	points := DataPoints{}

	for i := 0; i < len(data.Hourly.Time); i++ {
		timeString := data.Hourly.Time[i]
		time, _ := time.ParseInLocation("2006-01-02T15:04", timeString, time.FixedZone(data.TimezoneAbbreviation, 1*60*60))

		point := DataPoint{
			Time:       time,
			LowClouds:  data.Hourly.CloudCoverLow[i],
			MidClouds:  data.Hourly.CloudCoverMid[i],
			HighClouds: data.Hourly.CloudCoverHigh[i],
			WindSpeed:  data.Hourly.WindSpeed10M[i],
			WindGusts:  data.Hourly.WindGusts10M[i],
		}

		points = append(points, point)
	}

	return points
}

// Init() writes forcast state default value for the state file. "0" means bad weather next 24 hours
func (s State) Init() {
	d := []byte("0")
	err := os.WriteFile(defaultStateFilePath, d, 0644)
	if err != nil {
		log.Println("ERROR: can't write to Status file", err)
	}
	log.Println("INFO: Status file initialized with 0 value")
}

// Set() writes state to the state file
func (s State) Set(b bool) {
	if b {
		d := []byte("1")
		err := os.WriteFile(defaultStateFilePath, d, 0644)
		if err != nil {
			log.Println("ERROR: can't write to Status file", err)
		}
	} else {
		d := []byte("0")
		err := os.WriteFile(defaultStateFilePath, d, 0644)
		if err != nil {
			log.Println("ERROR: can't write to Status file", err)
		}
	}
	log.Println("INFO: Status file updated with", b)
}

// isGood() returns true if state file contains "1" and false when "0"
func (s State) isGood() bool {
	dat, err := os.ReadFile(defaultStateFilePath)
	if err != nil {
		log.Println("ERROR: can't read from Status file", err)
	}

	if string(dat) == "0" {
		return false
	} else {
		return true
	}
}

// getAllStartPoints() return all time DataPoints with good weather
func getAllStartPoints() DataPoints {
	data := OpenMeteoAPIResponse{}
	data.Init()

	points := data.Points().Good().onlyStartPoints()

	return points
}

// checkNext24H() is cron job which monitors good/bad weather next 24 hours
func checkNext24H(bot *tgbotapi.BotAPI) {
	chatID, err := strconv.Atoi(os.Getenv("CHAT_ID"))
	if err != nil {
		log.Println("ERROR: cannot convert CHAT_ID value to int")
	}
	cronExpression := getEnv("CRON_EXPRESSION", defaultCronExpression)

	msg := tgbotapi.NewMessage(int64(chatID), "")
	msg.ChatID = int64(chatID)
	msg.ParseMode = "MarkdownV2"

	s := gocron.NewScheduler(time.UTC)
	var state State
	state.Init()

	_, err = s.Cron(cronExpression).Do(func() {
		log.Println("INFO: starting cron job")
		startPoints := getAllStartPoints()
		next24HStartPoints := startPoints.next24H()

		if len(next24HStartPoints) > 0 && !state.isGood() {
			log.Println("INFO: good weather in the next 24h. Sending message")
			msg.Text = mono(goodWeatherAlert + "\n\n" + next24HStartPoints.setMoonIllumination().Print())

			if _, err := bot.Send(msg); err != nil {
				log.Println("ERROR: can't send message to Telegram", err)
			}
			state.Set(true)
		} else if len(next24HStartPoints) == 0 && state.isGood() {
			log.Println("INFO: No more good forecast for the next 24h. Sending message")
			msg.Text = mono(badWeatherAlert)

			if _, err := bot.Send(msg); err != nil {
				log.Println("ERROR: can't send message to Telegram", err)
			}
			state.Set(false)
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
func authChat(chatID int64) bool {
	chatIDString := strconv.Itoa(int(chatID))
	return chatIDString == os.Getenv("CHAT_ID")
}

// handleChat() is telegram bot handler for chat interactions
func handleChat(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	if !authChat(update.Message.Chat.ID) {
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
			forecast := getAllStartPoints().setMoonIllumination().Print()
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

func main() {
	bot, err := tgbotapi.NewBotAPI(os.Getenv("TG_BOT_TOKEN"))
	if err != nil {
		log.Panic(err)
	}
	log.Printf("INFO: Authorized on account %s", bot.Self.UserName)

	// Start 24h check in background
	checkNext24H(bot)
	log.Println("INFO: Background cron job activated")

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		handleChat(bot, update)
	}
}
