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
	stateFilePath = "state.txt" // 0 - last message was about bad forecast, 1 - about good
)

type MBCloudsResponse struct {
	Metadata Metadata `json:"metadata"`
	Units    Units    `json:"units"`
	Data1H   Data1H   `json:"data_1h"`
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

type Metadata struct {
	Name                  string  `json:"name"`
	Latitude              float64 `json:"latitude"`
	Longitude             float64 `json:"longitude"`
	Height                int64   `json:"height"`
	TimezoneAbbrevation   string  `json:"timezone_abbrevation"`
	UTCTimeoffset         float64 `json:"utc_timeoffset"`
	ModelrunUTC           string  `json:"modelrun_utc"`
	ModelrunUpdatetimeUTC string  `json:"modelrun_updatetime_utc"`
}

type Units struct {
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
}

type DataPoints []DataPoint
type State bool

// isGood() returns true if Low, Mid and High clouds percentage is less than MAX_CLOUD_COVER
func (d DataPoint) isGood() bool {
	maxCloudCoverInt, _ := strconv.Atoi(os.Getenv("MAX_CLOUD_COVER"))
	maxCloudCoverInt64 := int64(maxCloudCoverInt)

	if d.HighClouds <= maxCloudCoverInt64 && d.MidClouds <= maxCloudCoverInt64 && d.LowClouds <= maxCloudCoverInt64 {
		return true
	} else {
		return false
	}
}

// atNight() returns true if time is between NIGHT_START_HOUR and NIGHT_END_HOUR
func (d DataPoint) atNight() bool {
	nightStart, _ := strconv.Atoi(os.Getenv("NIGHT_START_HOUR"))
	nightEnd, _ := strconv.Atoi(os.Getenv("NIGHT_END_HOUR"))

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
	hoursStreak, _ := strconv.Atoi(os.Getenv("NIGHT_HOURS_STREAK"))

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

func (dp DataPoints) Print() string {
	out := "`"
	for _, point := range dp {
		out += fmt.Sprintln(point.Time.Format("🟢 Mon - Jan 02 15:04"), "|", point.LowClouds, point.MidClouds, point.HighClouds)
	}
	out += "`"

	return out
}

// Init() goes to MB_API_ENDPOINT makes HTTPS request and stores result as MBCloudsResponse object
func (mbresponse *MBCloudsResponse) Init() {
	client := &http.Client{}
	MeteoblueAPIEndpoint := os.Getenv("MB_API_ENDPOINT")

	// Set paramenters
	params := url.Values{}
	params.Add("apikey", os.Getenv("MB_API_KEY"))
	params.Add("lat", os.Getenv("MB_LAT"))
	params.Add("lon", os.Getenv("MB_LON"))
	params.Add("asl", os.Getenv("MB_ALT"))
	params.Add("format", "json")

	// Make request to Meteoblue API
	req, _ := http.NewRequest("GET", MeteoblueAPIEndpoint+params.Encode(), nil)

	parseFormErr := req.ParseForm()
	if parseFormErr != nil {
		fmt.Println(parseFormErr)
	}

	resp, err := client.Do(req)

	if err != nil {
		fmt.Println("Failure : ", err)
	}

	// Read Response Body
	if resp.StatusCode != 200 {
		fmt.Println("response Status : ", resp.Status)
		log.Fatal("Exit. Response from Meteoblue API is not 200 OK")
	}

	respBody, _ := io.ReadAll(resp.Body)

	// Save response as MBCloudsResponse object
	err = json.Unmarshal(respBody, mbresponse)
	if err != nil {
		log.Fatal(err)
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

func (s State) Init() {

	d := []byte("0")
	err := os.WriteFile(stateFilePath, d, 0644)
	if err != nil {
		log.Panic(err)
	}
}

func (s State) Set(b bool) {
	if b {
		d := []byte("1")
		err := os.WriteFile(stateFilePath, d, 0644)
		if err != nil {
			log.Panic(err)
		}
	} else {
		d := []byte("0")
		err := os.WriteFile(stateFilePath, d, 0644)
		if err != nil {
			log.Panic(err)
		}
	}
}

func (s State) isGood() bool {
	dat, err := os.ReadFile(stateFilePath)
	if err != nil {
		log.Panic(err)
	}

	if string(dat) == "0" {
		return false
	} else {
		return true
	}
}

func getAllStartPoints() DataPoints {
	data := MBCloudsResponse{}
	data.Init()

	points := data.Points()
	pointsGood := points.Good()
	startPoints := pointsGood.onlyStartPoints()

	return startPoints
}

func checkNext24H(bot *tgbotapi.BotAPI) {

	chatID, _ := strconv.Atoi(os.Getenv("CHAT_ID"))
	cronExpression := os.Getenv("CRON_EXPRESSION")

	msg := tgbotapi.NewMessage(int64(chatID), "")
	msg.ChatID = int64(chatID)
	msg.ParseMode = "MarkdownV2"

	s := gocron.NewScheduler(time.UTC)
	var state State
	state.Init()

	_, err := s.Cron(cronExpression).Do(func() {
		startPoints := getAllStartPoints()
		next24HStartPoints := startPoints.next24H()
		if len(next24HStartPoints) > 0 && !state.isGood() {
			msg.Text = "🥳 Хороша погода сьогодні!"
			msg.Text += next24HStartPoints.Print()

			if _, err := bot.Send(msg); err != nil {
				log.Panic(err)
			}
			state.Set(true)
		} else if len(next24HStartPoints) == 0 && state.isGood() {
			msg.Text = "Охрана, отмєна! Хорошої погоди не буде наступні 24 години 🥺"
			if _, err := bot.Send(msg); err != nil {
				log.Panic(err)
			}
			state.Set(false)
		} else {
			log.Println("No changes in weather for the next 24 hour")
		}
	})
	if err != nil {
		log.Println(err)
	}

	s.StartAsync()
}

func handleChat(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	var numericKeyboard = tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Прогноз на 7 днів"),
		),
	)

	if update.Message != nil {
		log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")
		msg.ReplyMarkup = numericKeyboard

		if update.Message.IsCommand() && update.Message.Command() == "start" {
			msg.Text = "Розпочнімо. Тицяй кнопку!"
		} else if update.Message.Text == "Прогноз на 7 днів" {
			msg.Text = getAllStartPoints().Print()
			msg.ParseMode = "MarkdownV2"
		} else {
			msg.Text = "Не розумію!"
		}

		if _, err := bot.Send(msg); err != nil {
			log.Panic(err)
		}
	}
}

func main() {
	bot, err := tgbotapi.NewBotAPI(os.Getenv("TG_BOT_TOKEN"))
	if err != nil {
		log.Panic(err)
	}
	bot.Debug = true
	log.Printf("Authorized on account %s", bot.Self.UserName)

	// Start 24h check in background
	checkNext24H(bot)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		handleChat(bot, update)
	}
}

// TODO
// default values for env variables
// start cron on the very beginning +
// add env var for CHAT_ID +
// add logging
// add exception handling
// remove debug
// set messages as constants
// implement cron logic which prevents message duplication (save state) +
// run cron only from 9 to 9
