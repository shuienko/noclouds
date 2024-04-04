package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/go-co-op/gocron"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	defaultStateFilePath    = "state.txt" // 0 - last message was about bad forecast, 1 - about good
	defaultMaxCloudCover    = "25"
	defaultNightStartHour   = "22"
	defaultNightEndHour     = "5"
	defaultNightHoursStreak = "4"
	cloudsMBApiEndpoint     = "https://my.meteoblue.com/packages/clouds-1h?"
	sunmoonMBApiEndpoint    = "https://my.meteoblue.com/packages/sunmoon?"
	defaultCronExpression   = "0 8,12,16,20 * * *"

	badWeatherAlert   = "Сьогодні хмарно 🥺"
	goodWeatherAlert  = "Хороша погода сьогодні! 🥳"
	startMessage      = "Розпочнімо. Тицяй кнопку."
	badRequestMessage = "Не розумію..."
	noGoodWeather7d   = "Хмарно наступні 7 днів 🥺"
)

type MBSunMoonResponse struct {
	Metadata SunMoonMetadata `json:"metadata"`
	Units    SunMoonUnits    `json:"units"`
	DataDay  DataDay         `json:"data_day"`
}

type DataDay struct {
	Time                 []string  `json:"time"`
	Moonrise             []string  `json:"moonrise"`
	Moonset              []string  `json:"moonset"`
	Moonphaseangle       []float64 `json:"moonphaseangle"`
	Sunset               []string  `json:"sunset"`
	Moonphasename        []string  `json:"moonphasename"`
	Moonphasetransittime []string  `json:"moonphasetransittime"`
	Sunrise              []string  `json:"sunrise"`
	Moonage              []float64 `json:"moonage"`
}

type SunMoonMetadata struct {
	ModelrunUpdatetimeUTC string  `json:"modelrun_updatetime_utc"`
	Name                  string  `json:"name"`
	Height                int64   `json:"height"`
	TimezoneAbbrevation   string  `json:"timezone_abbrevation"`
	Latitude              float64 `json:"latitude"`
	ModelrunUTC           string  `json:"modelrun_utc"`
	Longitude             float64 `json:"longitude"`
	UTCTimeoffset         float64 `json:"utc_timeoffset"`
	GenerationTimeMS      float64 `json:"generation_time_ms"`
}

type SunMoonUnits struct {
	Time string `json:"time"`
}

type MBCloudsResponse struct {
	Metadata CloudsMetadata `json:"metadata"`
	Units    CloudsUnits    `json:"units"`
	Data1H   Data1H         `json:"data_1h"`
}

type Data1H struct {
	Time            []string `json:"time"`
	Sunshinetime    []*int64 `json:"sunshinetime"`
	Lowclouds       []int64  `json:"lowclouds"`
	Midclouds       []int64  `json:"midclouds"`
	Highclouds      []int64  `json:"highclouds"`
	Visibility      []int64  `json:"visibility"`
	Totalcloudcover []int64  `json:"totalcloudcover"`
}

type CloudsMetadata struct {
	Name                  string  `json:"name"`
	Latitude              float64 `json:"latitude"`
	Longitude             float64 `json:"longitude"`
	Height                int64   `json:"height"`
	TimezoneAbbrevation   string  `json:"timezone_abbrevation"`
	UTCTimeoffset         float64 `json:"utc_timeoffset"`
	ModelrunUTC           string  `json:"modelrun_utc"`
	ModelrunUpdatetimeUTC string  `json:"modelrun_updatetime_utc"`
}

type CloudsUnits struct {
	Time         string `json:"time"`
	Cloudcover   string `json:"cloudcover"`
	Sunshinetime string `json:"sunshinetime"`
	Visibility   string `json:"visibility"`
}

type DataPoint struct {
	Time       time.Time
	LowClouds  int64
	MidClouds  int64
	HighClouds int64
	MoonPhase  int64
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

// mono() returns monospaced escaped Markdown
func mono(s string) string {
	return "`" + tgbotapi.EscapeText("MarkdownV2", s) + "`"
}

// isGood() returns true if Low, Mid and High clouds percentage is less than MAX_CLOUD_COVER
func (d DataPoint) isGood() bool {
	maxCloudCoverInt64 := toInt64(getEnv("MAX_CLOUD_COVER", defaultMaxCloudCover))

	if d.HighClouds <= maxCloudCoverInt64 && d.MidClouds <= maxCloudCoverInt64 && d.LowClouds <= maxCloudCoverInt64 {
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
func (dp *DataPoints) setMoonPhase() {
	data := MBSunMoonResponse{}
	data.Init()

	for _, point := range *dp {
		point.MoonPhase = data.getMoonPhase(point.Time)
		fmt.Println(point)
	}

}

// Print() returns Markdown string which represents DataPoints
func (dp DataPoints) Print() string {
	out := ""
	for _, point := range dp {
		out += fmt.Sprintln(point.MoonPhase, "|", point.Time.Format("Mon - Jan 02 15:04"), "|", point.LowClouds, point.MidClouds, point.HighClouds)
	}

	return out
}

// Init() goes to MB_API_ENDPOINT makes HTTPS request and stores result as MBCloudsResponse object
func (mbresponse *MBCloudsResponse) Init() {
	log.Println("INFO: Making request to Meteoblue API /clouds and parsing response")
	client := &http.Client{}
	MeteoblueAPIEndpoint := getEnv("MB_API_ENDPOINT", cloudsMBApiEndpoint)

	// Set paramenters
	params := url.Values{}
	params.Add("apikey", os.Getenv("MB_API_KEY"))
	params.Add("lat", os.Getenv("MB_LAT"))
	params.Add("lon", os.Getenv("MB_LON"))
	params.Add("asl", os.Getenv("MB_ALT"))
	params.Add("format", "json")

	// Make request to Meteoblue API
	req, err := http.NewRequest("GET", MeteoblueAPIEndpoint+params.Encode(), nil)
	if err != nil {
		log.Println("ERROR: Couldn't create New Meteoblue API request", err)
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
		fmt.Println("ERROR: Meteoblue response code:", resp.Status)
		return
	}

	log.Println("INFO: Got API response", resp.Status)
	respBody, _ := io.ReadAll(resp.Body)

	// Save response as MBCloudsResponse object
	err = json.Unmarshal(respBody, mbresponse)
	if err != nil {
		log.Println("ERROR: cannot Unmarshal JSON", err)
		return
	}
}

// Init() goes to /sunmoon makes HTTPS request and stores result as MBSunMoonResponse object
func (mbresponse *MBSunMoonResponse) Init() {
	log.Println("INFO: Making request to Meteoblue API /sunmoon and parsing response")
	client := &http.Client{}

	// Set paramenters
	params := url.Values{}
	params.Add("apikey", os.Getenv("MB_API_KEY"))
	params.Add("lat", os.Getenv("MB_LAT"))
	params.Add("lon", os.Getenv("MB_LON"))

	// Make request to Meteoblue API
	req, err := http.NewRequest("GET", sunmoonMBApiEndpoint+params.Encode(), nil)
	if err != nil {
		log.Println("ERROR: Couldn't create New Meteoblue API request", err)
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
		fmt.Println("ERROR: Meteoblue response code:", resp.Status)
		return
	}

	log.Println("INFO: Got API response", resp.Status)
	respBody, _ := io.ReadAll(resp.Body)

	// Save response as MBCloudsResponse object
	err = json.Unmarshal(respBody, mbresponse)
	if err != nil {
		log.Println("ERROR: cannot Unmarshal JSON", err)
		return
	}
}

// getMoonPhase() return Moon Phase in %
func (mbresponse MBSunMoonResponse) getMoonPhase(t time.Time) int64 {
	now := time.Now()
	diff := t.Sub(now)
	index := int(diff.Hours() / 24)

	fmt.Println("index =", index)

	fmt.Println(mbresponse.DataDay.Moonphaseangle)
	angle := mbresponse.DataDay.Moonphaseangle[index]

	fmt.Println("angle = ", angle)
	if angle > 180 {
		return int64(100 * (360 - angle) / 180)
	} else {
		return int64(100 * (180 - angle) / 180)
	}

}

// Points() return DataPoints object based on MBCloudsResponse fields
func (data MBCloudsResponse) Points() DataPoints {
	points := DataPoints{}

	for i := 0; i < len(data.Data1H.Time); i++ {
		timeString := data.Data1H.Time[i]
		time, _ := time.ParseInLocation("2006-01-02 15:04", timeString, time.FixedZone(data.Metadata.TimezoneAbbrevation, 1*60*60))

		point := DataPoint{
			Time:       time,
			LowClouds:  data.Data1H.Lowclouds[i],
			MidClouds:  data.Data1H.Midclouds[i],
			HighClouds: data.Data1H.Highclouds[i],
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
	data := MBCloudsResponse{}
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
			next24HStartPoints.setMoonPhase()
			msg.Text = mono(goodWeatherAlert + "\n\n" + next24HStartPoints.Print())

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
			points := getAllStartPoints()
			points.setMoonPhase()

			forecast := points.Print()
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
